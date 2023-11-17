package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

type Info struct {
	FreeBch          float64    `json:"free_bch"`
	FreeSbch         float64    `json:"free_sbch"`
	LockedBch        float64    `json:"locked_bch"`
	LockedSbch       float64    `json:"locked_sbch"`
	ToBeUnlockedBch  float64    `json:"to_be_unlocked_bch"`
	ToBeUnlockedSbch float64    `json:"to_be_unlocked_sbch"`
	S2BSwaps         []SwapInfo `json:"s2b_swaps"`
	B2SSwaps         []SwapInfo `json:"b2s_swaps"`
}

type SwapInfo struct {
	HashLock string  `json:"hash_lock"`
	Value    float64 `json:"value"`
	Status   string  `json:"status"`
}

type Resp struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Result  any    `json:"result,omitempty"`
}

func NewErrResp(err string) Resp {
	return Resp{
		Success: false,
		Error:   err,
	}
}
func NewOkResp(result any) Resp {
	return Resp{
		Success: true,
		Result:  result,
	}
}

func (resp Resp) WriteTo(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	//w.Header().Set("Access-Control-Allow-Methods", "*")
	//w.Header().Set("Access-Control-Allow-Headers", "origin, content-type, accept")

	bytes, _ := json.Marshal(resp)
	_, _ = w.Write(bytes)
}

func (bot *MarketMakerBot) StartHttpServer(listenAddr string) {
	mux := bot.createHttpHandlers()
	server := http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	log.Info("server listening at:", listenAddr, "...")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (bot *MarketMakerBot) createHttpHandlers() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { bot.handlePing(w, r) })
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) { bot.handleLogs(w, r) })
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) { bot.handleInfo(w, r) })
	return mux
}

// return "pong"
func (bot *MarketMakerBot) handlePing(w http.ResponseWriter, r *http.Request) {
	NewOkResp("pong").WriteTo(w)
}

// remove and return a number of logs from queue
func (bot *MarketMakerBot) handleLogs(w http.ResponseWriter, r *http.Request) {
	n := getIntQueryParam(r, "n", 100)
	logs := bot.errLogQueue.removeErrLogs(n)
	NewOkResp(logs).WriteTo(w)
}

// return bot balance info
func (bot *MarketMakerBot) handleInfo(w http.ResponseWriter, r *http.Request) {
	info, err := bot.getBotInfo()
	if err != nil {
		NewErrResp(err.Error()).WriteTo(w)
	} else {
		NewOkResp(info).WriteTo(w)
	}
}

func (bot *MarketMakerBot) getBotInfo() (*Info, error) {
	freeBch, err := bot.getFreeBch()
	if err != nil {
		return nil, fmt.Errorf("failed to query UTXOs: %w", err)
	}

	freeSbch, err := bot.getFreeSbch()
	if err != nil {
		return nil, fmt.Errorf("failed to query sBCH balance: %w", err)
	}

	toBeUnlockedSbch, lockedBch, s2bSwapInfos, err := bot.getSbch2BchInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to query DB: %w", err)
	}

	toBeUnlockedBch, lockedSbch, b2sSwapInfos, err := bot.getBch2SbchInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to query DB: %w", err)
	}

	return &Info{
		FreeBch:          freeBch,
		FreeSbch:         freeSbch,
		LockedBch:        lockedBch,
		LockedSbch:       lockedSbch,
		ToBeUnlockedBch:  toBeUnlockedBch,
		ToBeUnlockedSbch: toBeUnlockedSbch,
		B2SSwaps:         b2sSwapInfos,
		S2BSwaps:         s2bSwapInfos,
	}, nil
}

func (bot *MarketMakerBot) getFreeBch() (float64, error) {
	utxos, err := bot.bchCli.GetAllUTXOs()
	if err != nil {
		return 0, err
	}

	freeBch := float64(0)
	for _, utxo := range utxos {
		freeBch += utxo.Amount
	}
	return freeBch, nil
}

func (bot *MarketMakerBot) getFreeSbch() (float64, error) {
	freeSbch, err := bot.sbchCliRO.getBotBalance()
	if err != nil {
		return 0, err
	}
	return satsToUtxoAmt(weiToSats(freeSbch)), nil
}

func (bot *MarketMakerBot) getSbch2BchInfo() (
	toBeUnlockedSbch, lockedBch float64,
	swapInfos []SwapInfo,
	err error,
) {
	newRecords, err := bot.db.getSbch2BchRecordsByStatus(Sbch2BchStatusNew, 500)
	if err != nil {
		return 0, 0, nil, err
	}

	bchLockedRecords, err := bot.db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 500)
	if err != nil {
		return 0, 0, nil, err
	}

	secretRevealedRecords, err := bot.db.getSbch2BchRecordsByStatus(Sbch2BchStatusSecretRevealed, 500)
	if err != nil {
		return 0, 0, nil, err
	}

	for _, record := range newRecords {
		toBeUnlockedSbch += satsToUtxoAmt(record.Value)
		swapInfos = append(swapInfos, SwapInfo{
			HashLock: record.HashLock,
			Value:    satsToUtxoAmt(record.Value),
			Status:   "New",
		})
	}
	for _, record := range bchLockedRecords {
		toBeUnlockedSbch += satsToUtxoAmt(record.Value)
		lockedBch += satsToUtxoAmt(record.Value)
		swapInfos = append(swapInfos, SwapInfo{
			HashLock: record.HashLock,
			Value:    satsToUtxoAmt(record.Value),
			Status:   "BchLocked",
		})
	}
	for _, record := range secretRevealedRecords {
		toBeUnlockedSbch += satsToUtxoAmt(record.Value)
		lockedBch += satsToUtxoAmt(record.Value)
		swapInfos = append(swapInfos, SwapInfo{
			HashLock: record.HashLock,
			Value:    satsToUtxoAmt(record.Value),
			Status:   "SecretRevealed",
		})
	}

	return
}

func (bot *MarketMakerBot) getBch2SbchInfo() (
	toBeUnlockedBch, lockedSbch float64,
	swapInfos []SwapInfo,
	err error,
) {
	newRecords, err := bot.db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 500)
	if err != nil {
		return 0, 0, nil, err
	}
	sbchLockedRecords, err := bot.db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 500)
	if err != nil {
		return 0, 0, nil, err
	}
	secretRevealedRecords, err := bot.db.getBch2SbchRecordsByStatus(Bch2SbchStatusSecretRevealed, 500)
	if err != nil {
		return 0, 0, nil, err
	}

	for _, record := range newRecords {
		toBeUnlockedBch += satsToUtxoAmt(record.Value)
		swapInfos = append(swapInfos, SwapInfo{
			HashLock: record.HashLock,
			Value:    satsToUtxoAmt(record.Value),
			Status:   "New",
		})
	}
	for _, record := range sbchLockedRecords {
		toBeUnlockedBch += satsToUtxoAmt(record.Value)
		lockedSbch += satsToUtxoAmt(record.Value)
		swapInfos = append(swapInfos, SwapInfo{
			HashLock: record.HashLock,
			Value:    satsToUtxoAmt(record.Value),
			Status:   "SbchLocked",
		})
	}
	for _, record := range secretRevealedRecords {
		toBeUnlockedBch += satsToUtxoAmt(record.Value)
		lockedSbch += satsToUtxoAmt(record.Value)
		swapInfos = append(swapInfos, SwapInfo{
			HashLock: record.HashLock,
			Value:    satsToUtxoAmt(record.Value),
			Status:   "SecretRevealed",
		})
	}

	return
}

func getIntQueryParam(r *http.Request, name string, defaultVal int) int {
	params := r.URL.Query()[name]
	if len(params) == 0 {
		return defaultVal
	}
	n, err := strconv.ParseInt(params[0], 10, 64)
	if err != nil {
		return defaultVal
	}
	return int(n)
}
