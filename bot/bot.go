package bot

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/atomic-swap-bot/htlcbch"
	"github.com/smartbch/atomic-swap-bot/htlcsbch"
)

type MarketMakerBot struct {
	db      DB
	bchCli  IBchClient
	sbchCli ISbchClient

	// BCH key
	bchPrivKey *bchec.PrivateKey
	bchPubKey  []byte
	bchPkh     []byte
	bchAddr    bchutil.Address // P2PKH

	// sBCH key
	sbchPrivKey *ecdsa.PrivateKey
	sbchAddr    gethcmn.Address

	// HTLC params
	bchTimeLock  uint16 // in blocks
	sbchTimeLock uint32 // in seconds
	penaltyRatio uint16 // in BPS

	// bot params
	serviceFeeRatio        uint16 // in BPS
	minSwapVal             uint64 // in sats
	maxSwapVal             uint64 // in sats
	bchConfirmations       uint8
	bchSendMinerFeeRate    uint64 // sats/byte
	bchReceiveMinerFeeRate uint64 // sats/byte
	bchRefundMinerFeeRate  uint64 // sats/byte
}

func NewBot(
	dbFile string,
	bchPrivKeyWIF, sbchPrivKeyHex string,
	bchRpcUrl, sbchRpcUrl string,
	sbchHtlcAddr gethcmn.Address,
	sbchGasPrice *big.Int,
	bchTimeLock uint16,
	sbchTimeLock uint32,
	penaltyRatio uint16,
	feeRatio uint16,
	minSwapVal, maxSwapVal uint64,
	bchConfirmations uint8,
	bchSendMinerFeeRate, bchReceiveMinerFeeRate, bchRefundMinerFeeRate uint64,
	sbchOpenGasLimit, sbchCloseGasLimit, sbchExpireGasLimit uint64,
	debugMode bool,
) (*MarketMakerBot, error) {

	// load BCH key
	wif, err := bchutil.DecodeWIF(bchPrivKeyWIF)
	if err != nil {
		return nil, fmt.Errorf("failed to load sBCH private key: %w", err)
	}
	bchPrivKey := wif.PrivKey
	bchPbk := bchPrivKey.PubKey().SerializeCompressed()
	bchPkh := bchutil.Hash160(bchPbk)
	bchAddr, err := bchutil.NewAddressPubKeyHash(bchPkh, getBchParams(debugMode))
	if err != nil {
		return nil, fmt.Errorf("failed to derive BCH recipient address")
	}

	// load sBCH key
	sbchPrivKey, err := gethcrypto.HexToECDSA(sbchPrivKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to load sBCH private key: %w", err)
	}
	sbchAddr := gethcrypto.PubkeyToAddress(sbchPrivKey.PublicKey)

	// create RPC clients
	bchCli, err := newBchClient(bchRpcUrl, bchAddr)
	if err != nil {
		return nil, fmt.Errorf("faield to create BCH RPC client: %w", err)
	}
	sbchCli, err := newSbchClient(sbchRpcUrl, 5*time.Second, sbchPrivKey, sbchAddr, sbchHtlcAddr,
		sbchGasPrice, sbchOpenGasLimit, sbchCloseGasLimit, sbchExpireGasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to create sBCH RPC client: %w", err)
	}

	// open DB
	db, err := OpenDB(dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB file: %w", err)
	}

	// print bot info
	log.Info("BCH pubkey  : ", "0x"+hex.EncodeToString(bchPbk))
	log.Info("BCH PKH     : ", "0x"+hex.EncodeToString(bchPkh))
	log.Info("BCH address : ", bchAddr.String())
	log.Info("sBCH address: ", sbchAddr.String())

	return &MarketMakerBot{
		db:                     db,
		bchCli:                 bchCli,
		bchPrivKey:             bchPrivKey,
		bchPubKey:              bchPbk,
		bchPkh:                 bchPkh,
		bchAddr:                bchAddr,
		sbchCli:                sbchCli,
		sbchPrivKey:            sbchPrivKey,
		sbchAddr:               sbchAddr,
		bchTimeLock:            bchTimeLock,
		sbchTimeLock:           sbchTimeLock,
		penaltyRatio:           penaltyRatio,
		serviceFeeRatio:        feeRatio,
		minSwapVal:             minSwapVal,
		maxSwapVal:             maxSwapVal,
		bchSendMinerFeeRate:    bchSendMinerFeeRate,
		bchReceiveMinerFeeRate: bchReceiveMinerFeeRate,
		bchRefundMinerFeeRate:  bchRefundMinerFeeRate,
		bchConfirmations:       bchConfirmations,
	}, nil
}

func getBchParams(debugMode bool) *chaincfg.Params {
	if debugMode {
		return &chaincfg.TestNet3Params
	}
	return &chaincfg.MainNetParams
}

func (bot *MarketMakerBot) PrepareDB() {
	_, err := bot.db.getLastHeights()
	if err == nil || !strings.HasPrefix(err.Error(), "no such table") {
		return
	}

	log.Info("init DB, sync schemas ...")
	if err = bot.db.syncSchemas(); err != nil {
		log.Fatal(err)
	}
	log.Info("init last BCH & sBCH heights ...")
	if err = bot.db.initLastHeights(0, 0); err != nil {
		log.Fatal(err)
	}
}

func (bot *MarketMakerBot) GetUTXOs() ([]btcjson.ListUnspentResult, error) {
	return bot.bchCli.getAllUTXOs()
}

func (bot *MarketMakerBot) Loop() {
	for {
		log.Info("---------- ", time.Now(), "' ----------")
		gotNewBlocks := bot.scanBchBlocks()
		bot.handleBchRefunds(gotNewBlocks)
		bot.handleBchUserDeposits()
		bot.unlockBchUserDeposits()
		bot.handleSbchRefunds()
		bot.scanSbchEvents()
		bot.handleSbchUserDeposits()
		bot.unlockSbchUserDeposits()
		time.Sleep(2 * time.Second)
	}
}

func (bot *MarketMakerBot) scanBchBlocks() (gotNewBlocks bool) {
	log.Info("scan BCH blocks ...")
	lastBlockNum, err := bot.db.getLastBchHeight()
	if err != nil {
		log.Fatal("DB error, failed to get last BCH height: ", err)
		return
	}
	log.Info("last BCH height: ", lastBlockNum)

	latestBlockNum, err := bot.bchCli.getBlockCount()
	if err != nil {
		log.Error("RPC error, failed to get BCH height: ", err)
		return
	}
	log.Info("latest BCH height: ", latestBlockNum)

	safeNewBlockNum := latestBlockNum - int64(bot.bchConfirmations)

	if lastBlockNum == 0 {
		lastBlockNum = uint64(safeNewBlockNum) - 1
		log.Info("init last BCH height: ", lastBlockNum)
	}

	for h := int64(lastBlockNum) + 1; h <= safeNewBlockNum; h++ {
		if !bot.handleBchBlock(h) {
			break
		}
	}

	gotNewBlocks = safeNewBlockNum > int64(lastBlockNum)
	return gotNewBlocks
}

func (bot *MarketMakerBot) handleBchBlock(h int64) bool {
	//log.Info("get BCH block#", h, " ...")
	block, err := bot.bchCli.getBlock(h)
	if err != nil {
		log.Error("RPC error, failed to get BCH block#", h, " : ", err)
		return false
	}
	log.Info("got BCH block#", h)

	bot.handleBchDepositTxs(uint64(h), block)
	bot.handleBchReceiptTxs(block)
	bot.handleBchRefundTxs(block)

	err = bot.db.setLastBchHeight(uint64(h))
	if err != nil {
		log.Fatal("DB error, failed to update last BCH height: ", err)
	}

	return true
}

// find BCH lock txs, create bch2sbch records (status=new)
func (bot *MarketMakerBot) handleBchDepositTxs(h uint64, block *wire.MsgBlock) {
	deposits := htlcbch.GetHtlcDeposits(block, bot.bchPkh)
	log.Info("HTLC deposits: ", len(deposits))
	for _, deposit := range deposits {
		log.Info("HTLC deposit: ", toJSON(deposit))
		if deposit.Expiration != bot.bchTimeLock {
			log.Infof("invalid expiration: %d != %d",
				deposit.Expiration, bot.bchTimeLock)
			continue
		}
		if deposit.PenaltyBPS != bot.penaltyRatio {
			log.Infof("invalid penaltyRatio: %d != %d",
				deposit.PenaltyBPS, bot.penaltyRatio)
			continue
		}
		if deposit.Value < bot.minSwapVal ||
			(bot.maxSwapVal > 0 && deposit.Value > bot.maxSwapVal) {

			log.Infof("value out of range: %d ∉ [%d, %d]",
				deposit.Value, bot.minSwapVal, bot.maxSwapVal)
			continue
		}

		err := bot.db.addBch2SbchRecord(&Bch2SbchRecord{
			BchLockHeight:  h,
			BchLockTxHash:  deposit.TxHash,
			Value:          deposit.Value,
			RecipientPkh:   toHex(deposit.RecipientPkh),
			SenderPkh:      toHex(deposit.SenderPkh),
			HashLock:       toHex(deposit.HashLock),
			TimeLock:       uint32(deposit.Expiration),
			PenaltyBPS:     deposit.PenaltyBPS,
			SenderEvmAddr:  toHex(deposit.SenderEvmAddr),
			HtlcScriptHash: toHex(deposit.ScriptHash),
		})
		if err != nil {
			log.Error("DB error, failed to save Bch2SbchRecord: ", err)
		}
	}
}

// find BCH unlock txs
// for sbch2bch records, change status from BchLocked to SecretRevealed
// for bch2sbch records, change status from SecretRevealed to BchUnlocked
func (bot *MarketMakerBot) handleBchReceiptTxs(block *wire.MsgBlock) {
	receipts := htlcbch.GetHtlcReceipts(block)
	log.Info("HTLC receipts: ", len(receipts))

	// sbch2bch
	for _, receipt := range receipts {
		log.Info("HTLC receipt:", toJSON(receipt))
		record, err := bot.db.getSbch2BchRecordByBchLockTxHash(receipt.PrevTxHash)
		if err != nil {
			log.Error(fmt.Errorf("DB error, can not get Sbch2BchRecord, SbchLockTxHash=%s, err=%w", receipt.TxHash, err))
			continue
		}
		//log.Info(record)

		hashLock := secretToHashLock(gethcmn.FromHex(receipt.Secret))
		if hashLock != record.HashLock {
			log.Warnf("hashLock not match! secret: %s => hashLock: %s, DB hashLock: %s, ",
				receipt.Secret, hashLock, record.HashLock)
			continue
		}

		if record.Status != Sbch2BchStatusBchLocked {
			log.Infof("wrong status: %s", toJSON(record))
			continue
		}

		record.Secret = receipt.Secret
		record.BchUnlockTxHash = receipt.TxHash
		record.Status = Sbch2BchStatusSecretRevealed
		err = bot.db.updateSbch2BchRecord(record)
		if err != nil {
			log.Error("DB error, failed to update Sbch2Bch record: ", err)
		}
	}

	// bch2sbch
	// TODO: add more checks
	for _, receipt := range receipts {
		hashLock := secretToHashLock(gethcmn.FromHex(receipt.Secret))
		record, err := bot.db.getBch2SbchRecordByHashLock(hashLock)
		if err != nil {
			continue
		}
		if record.Status != Bch2SbchStatusSecretRevealed {
			continue
		}

		log.Info("HTLC receipt (BCH unlocked by others):", toJSON(receipt))
		record.Status = Bch2SbchStatusBchUnlocked
		record.BchUnlockTxHash = receipt.TxHash
		err = bot.db.updateBch2SbchRecord(record)
		if err != nil {
			log.Error("failed to update status of Bch2SbchRecord: ", err)
		}
	}
}

// find BCH refund txs
// for sbch2bch records, change status from BchRefundable to BchRefunded
func (bot *MarketMakerBot) handleBchRefundTxs(block *wire.MsgBlock) {
	// TODO: add more checks
	//refunds := htlcbch.GetHtlcRefunds(block)
	//log.Info("HTLC refunds: ", len(refunds))
	//for _, refund := range refunds {
	//	record, err := bot.db.getSbch2BchRecordByBchLockTxHash(refund.PrevTxHash)
	//	if err != nil {
	//		continue
	//	}
	//	if record.Status != Sbch2BchStatusBchRefundable {
	//		continue
	//	}
	//
	//	record.Status = Sbch2BchStatusBchRefunded
	//	record.BchUnlockTxHash = refund.TxHash
	//	err = bot.db.updateSbch2BchRecord(record)
	//	if err != nil {
	//		log.Error("failed to update status of Sbch2BchRecord: ", err)
	//	}
	//}
}

func (bot *MarketMakerBot) scanSbchEvents() {
	log.Info("scan sBCH events ...")
	lastBlockNum, err := bot.db.getLastSbchHeight()
	if err != nil {
		log.Fatal("failed to get last height of smartBCH from DB: ", err)
		return
	}
	log.Info("last sBCH height: ", lastBlockNum)

	newBlockNum, err := bot.sbchCli.getBlockNumber()
	if err != nil {
		log.Error("failed to get height of smartBCH: ", err)
		return
	}
	log.Info("latest sBCH height: ", newBlockNum)

	if lastBlockNum == 0 {
		lastBlockNum = newBlockNum - 1
		log.Info("init last sBCH height: ", lastBlockNum)
	}

	blockBatch := uint64(200)
	for fromH := lastBlockNum + 1; fromH <= newBlockNum; fromH += blockBatch {
		toH := fromH + blockBatch - 1
		if toH > newBlockNum {
			toH = newBlockNum
		}
		if !bot.handleSbchEvents(fromH, toH) {
			break
		}
	}
}

func (bot *MarketMakerBot) handleSbchEvents(fromH, toH uint64) bool {
	logs, err := bot.sbchCli.getHtlcLogs(fromH, toH)
	if err != nil {
		log.Error("failed to get smartBCH logs: ", err)
		return false
	}
	log.Infof("sBCH logs (block#%d ~ block#%d): %d",
		fromH, toH, len(logs))

	for _, ethLog := range logs {
		log.Info("sBCH log: ", toJSON(ethLog))
		switch ethLog.Topics[0] {
		case htlcsbch.OpenEventId:
			bot.handleSbchOpenEvents(ethLog)
		case htlcsbch.CloseEventId:
			bot.handleSbchCloseEvents(ethLog)
		case htlcsbch.ExpireEventId:
			bot.handleSbchExpireEvents(ethLog)
		}
	}

	err = bot.db.setLastSbchHeight(toH)
	if err != nil {
		log.Fatal("DB error, failed to update last sBCH height: ", err)
	}

	return true
}

func (bot *MarketMakerBot) handleSbchOpenEvents(ethLog gethtypes.Log) {
	openLog := htlcsbch.ParseHtlcOpenLog(ethLog)
	if openLog == nil {
		return
	}

	if openLog.UnlockerAddr != bot.sbchAddr {
		log.Info("not locked to me",
			", unlockerAddr: ", openLog.UnlockerAddr.String(),
			//", botAddr: ", bot.sbchAddr.String(),
		)
		return
	}

	zeroAddr := gethcmn.Address{}
	if openLog.BchRecipientPkh == zeroAddr {
		log.Info("BchRecipientPkh is zero, skip")
		return
	}

	penaltyBPS := openLog.PenaltyBPS
	if penaltyBPS != bot.penaltyRatio {
		log.Infof("invalid penaltyRatio: %d != %d",
			penaltyBPS, bot.penaltyRatio)
		return
	}

	sbchTimeLock := uint32(openLog.UnlockTime - openLog.CreatedTime)
	if sbchTimeLock != bot.sbchTimeLock {
		log.Infof("invalid TimeLock: %d != %d",
			sbchTimeLock, bot.sbchTimeLock)
		return
	}

	valSats := weiToSats(openLog.Value)
	if valSats < bot.minSwapVal ||
		(bot.maxSwapVal > 0 && valSats > bot.maxSwapVal) {

		log.Infof("value out of range: %d ∉ [%d, %d]",
			valSats, bot.minSwapVal, bot.maxSwapVal)
		return
	}

	log.Info("got a sBCH Open log: ", toJSON(openLog))
	bchTimeLock := sbchTimeLockToBlocks(sbchTimeLock) / 2
	covenant, err := htlcbch.NewMainnetCovenant(bot.bchPkh,
		openLog.BchRecipientPkh[:], openLog.HashLock[:], bchTimeLock, 0)
	if err != nil {
		log.Error("failed to create HTLC covenant: ", err)
		return
	}

	scriptHash, err := covenant.GetRedeemScriptHash()
	if err != nil {
		log.Error("failed to get script hash: ", err)
		return
	}

	err = bot.db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:    openLog.CreatedTime,
		SbchLockTxHash:  toHex(ethLog.TxHash[:]),
		Value:           valSats,
		SbchSenderAddr:  toHex(openLog.LockerAddr[:]),
		BchRecipientPkh: toHex(openLog.BchRecipientPkh[:]),
		HashLock:        toHex(openLog.HashLock[:]),
		TimeLock:        sbchTimeLock,
		PenaltyBPS:      penaltyBPS,
		HtlcScriptHash:  toHex(scriptHash),
	})
	if err != nil {
		log.Error("DB error, failed to save Sbch2BchRecord: ", err)
	}
}

func (bot *MarketMakerBot) handleSbchCloseEvents(ethLog gethtypes.Log) {
	closeLog := htlcsbch.ParseHtlcCloseLog(ethLog)
	if closeLog == nil {
		return
	}

	log.Info("got a sBCH Close log: ", toJSON(closeLog))
	hashLock := toHex(closeLog.HashLock[:])
	record, err := bot.db.getBch2SbchRecordByHashLock(hashLock)
	//log.Info(record)
	if err != nil {
		// TODO: change to log.Info
		log.Error(fmt.Errorf("DB error, can not get Bch2SbchRecord, hashLock=%s, err=%w",
			hashLock, err))
		return
	}

	hashLock2 := secretToHashLock(closeLog.Secret[:])
	if hashLock2 != hashLock {
		log.Warnf("hashLock not match! secret: %s => hashLock: %s, DB hashLock: %s, ",
			toHex(closeLog.Secret[:]), hashLock2, hashLock)
		return
	}

	record.Secret = toHex(closeLog.Secret[:])
	record.SbchUnlockTxHash = toHex(ethLog.TxHash[:])
	record.Status = Bch2SbchStatusSecretRevealed
	err = bot.db.updateBch2SbchRecord(record)
	if err != nil {
		log.Error("DB error, failed to update Bch2SbchRecord: ", err)
		return
	}
}

func (bot *MarketMakerBot) handleSbchExpireEvents(ethLog gethtypes.Log) {
	// TODO
}

func (bot *MarketMakerBot) handleBchUserDeposits() {
	log.Info("handle BCH user deposits ...")
	records, err := bot.db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	if err != nil {
		log.Error("DB error, failed to get BCH2SBCH records: ", err)
		return
	}
	log.Info("unhandled BCH user deposits: ", len(records))

	for _, record := range records {
		log.Info("handle BCH user deposit: ", toJSON(record))

		//confirmations := currBlockNum - int64(record.BchLockHeight) + 1
		confirmations, err := bot.bchCli.getTxConfirmations(record.BchLockTxHash)
		if err != nil {
			log.Error("RPC error, failed to get tx confirmations: ", err)
			continue
		}

		// do not send sBCH to user if it's too late!
		if confirmations > int64(bot.bchTimeLock)/3 {
			log.Info("too late to lock sBCH",
				", confirmations: ", confirmations,
				", timeLock: ", record.TimeLock)
			record.Status = Bch2SbchStatusTooLateToLockSbch
			err = bot.db.updateBch2SbchRecord(record)
			if err != nil {
				log.Error("failed to update status of Bch2SbchRecord: ", err)
			}

			continue
		}

		sbchTimeLock := bchTimeLockToSeconds(record.TimeLock) / 2
		bchValMinusFee := record.Value * (10000 - uint64(bot.serviceFeeRatio)) / 10000
		log.Info("sbchTimeLock: ", sbchTimeLock, " , bchValMinusFee: ", bchValMinusFee)

		txHash, err := bot.sbchCli.lockSbchToHtlc(
			gethcmn.HexToAddress(record.SenderEvmAddr),
			gethcmn.HexToHash(record.HashLock),
			sbchTimeLock,
			satsToWei(bchValMinusFee),
		)
		if err != nil {
			log.Error("RPC error, failed to lock sBCH to HTLC: ", err)
			continue
		}

		log.Info("lock sBCH successful",
			", hashLock: ", record.HashLock,
			", txHash: ", txHash.String())
		record.Status = Bch2SbchStatusSbchLocked
		record.SbchLockTxHash = toHex(txHash[:])
		record.SbchLockTxTime = uint64(time.Now().Unix())
		err = bot.db.updateBch2SbchRecord(record)
		if err != nil {
			log.Error("failed to update status of Bch2SbchRecord: ", err)
		}
	}
}

func (bot *MarketMakerBot) handleSbchUserDeposits() {
	log.Info("handle sBCH user deposits ...")

	lastBlockNum, err := bot.db.getLastBchHeight()
	if err != nil {
		log.Fatal("DB error, failed to get last BCH height: ", err)
		return
	}
	log.Info("last BCH height: ", lastBlockNum)

	records, err := bot.db.getSbch2BchRecordsByStatus(Sbch2BchStatusNew, 100)
	if err != nil {
		log.Error("DB error, failed to get unhandled sBCH user deposits: ", err)
		return
	}
	log.Info("unhandled sBCH user deposits: ", len(records))

	for _, record := range records {
		log.Info("SBCH2BCH record: ", toJSON(record))

		bchValMinusFee := int64(record.Value) * (10000 - int64(bot.serviceFeeRatio)) / 10000
		utxos, err := bot.bchCli.getUTXOs(bchValMinusFee+5000, 10)
		if err != nil {
			log.Error("failed to get UTXOs: ", err)
			continue
		}
		log.Info("bchValMinusFee: ", bchValMinusFee, ", UTXOs:", toJSON(utxos))

		inputs := make([]htlcbch.InputInfo, len(utxos))
		for i, utxo := range utxos {
			inputs[i] = htlcbch.InputInfo{
				TxID:   gethcmn.FromHex(utxo.TxID),
				Vout:   utxo.Vout,
				Amount: utxoAmtToSats(utxo.Amount),
			}
		}

		currTime, err := bot.sbchCli.getBlockTimeLatest()
		if err != nil {
			log.Error("RPC error, failed to get sBCH time: ", err)
			continue
		}

		// do not send BCH to user if its too late!
		timeElapsed := currTime - record.SbchLockTime
		if uint32(timeElapsed) > bot.sbchTimeLock/3 {
			log.Info("too late to lock BCH, time elapsed: ", timeElapsed, ", timeLock: ", record.TimeLock)
			record.Status = Sbch2BchStatusTooLateToLockSbch
			err = bot.db.updateSbch2BchRecord(record)
			if err != nil {
				log.Error("failed to update status of Bch2SbchRecord: ", err)
			}

			continue
		} else {
			log.Info("time elapsed: ", timeElapsed, ", timeLock: ", record.TimeLock)
		}

		bchTimeLock := sbchTimeLockToBlocks(record.TimeLock) / 2
		log.Info("BCH timeLock: ", bchTimeLock)

		covenant, err := htlcbch.NewMainnetCovenant(
			bot.bchPkh,
			gethcmn.FromHex(record.BchRecipientPkh),
			gethcmn.FromHex(record.HashLock),
			bchTimeLock,
			0,
		)
		if err != nil {
			log.Error("failed to create HTLC covenant: ", err)
			continue
		}

		tx, err := covenant.MakeLockTx(
			bot.bchPrivKey,
			inputs,
			bchValMinusFee,
			bot.bchSendMinerFeeRate,
		)
		if err != nil {
			log.Error("failed to create BCH tx: ", err)
			continue
		}
		log.Info("BCH tx hex: ", htlcbch.MsgTxToHex(tx))

		txHash, err := bot.bchCli.sendTx(tx)
		if err != nil {
			log.Error("failed to send BCH tx: ", err)

			// more debug info
			//prevPkScript, _ := htlcbch.PayToPubKeyHashPkScript(bot.bchPkh)
			//log.Infof("meep debug --tx=%s --idx=%d --amt=%d --pkscript=%s",
			//	htlcbch.MsgTxToHex(tx), 0, utxoAmtToSats(utxo.Amount), toHex(prevPkScript))
			continue
		}
		log.Info("BCH tx sent, hash: ", txHash.String())

		record.Status = Sbch2BchStatusBchLocked
		record.BchLockTxHash = txHash.String()
		record.BchLockBlockNum = lastBlockNum
		err = bot.db.updateSbch2BchRecord(record)
		if err != nil {
			log.Error("failed to update status of Bch2SbchRecord: ", err)
		}
	}
}

func (bot *MarketMakerBot) unlockBchUserDeposits() {
	log.Info("unlock BCH user deposits ...")
	records, err := bot.db.getBch2SbchRecordsByStatus(Bch2SbchStatusSecretRevealed, 100)
	if err != nil {
		log.Error("failed to get BCH2SBCH records from DB: ", err)
		return
	}
	log.Info("secret-revealed BCH user deposits: ", len(records))

	for _, record := range records {
		log.Info("record: ", toJSON(record))
		covenant, err := htlcbch.NewMainnetCovenant(
			gethcmn.FromHex(record.SenderPkh),
			gethcmn.FromHex(record.RecipientPkh),
			gethcmn.FromHex(record.HashLock),
			uint16(record.TimeLock),
			record.PenaltyBPS,
		)
		if err != nil {
			log.Error("failed to create HTLC covenant: ", err)
			continue
		}
		p2shAddr, _ := covenant.GetP2SHAddress()
		log.Info("covenant: ", p2shAddr)

		tx, err := covenant.MakeReceiveTx(
			gethcmn.FromHex(record.BchLockTxHash),
			0,
			int64(record.Value),
			bot.bchAddr,
			bot.bchReceiveMinerFeeRate,
			gethcmn.FromHex(record.Secret),
			bot.bchPrivKey,
		)
		if err != nil {
			log.Error("failed to create BCH tx: ", err)
			continue
		}
		log.Info("tx: ", htlcbch.MsgTxToHex(tx))
		txHash, err := bot.bchCli.sendTx(tx)
		if err != nil {
			log.Error("failed to send BCH tx: ", err)
			continue
		}
		log.Info("tx hash: ", txHash.String())

		record.Status = Bch2SbchStatusBchUnlocked
		record.BchUnlockTxHash = txHash.String()
		err = bot.db.updateBch2SbchRecord(record)
		if err != nil {
			log.Error("failed to update status of Bch2SbchRecord: ", err)
		}
	}
}

func (bot *MarketMakerBot) unlockSbchUserDeposits() {
	log.Info("unlock sBCH user deposits ...")
	records, err := bot.db.getSbch2BchRecordsByStatus(Sbch2BchStatusSecretRevealed, 100)
	if err != nil {
		log.Error("DB error, failed to get SBCH2BCH records from DB: ", err)
		return
	}
	log.Info("secret-revealed sBCH user deposits: ", len(records))

	for _, record := range records {
		log.Info("SBCH2BCH record: ", toJSON(record))
		txHash, err := bot.sbchCli.unlockSbchFromHtlc(
			gethcmn.HexToHash(record.HashLock),
			gethcmn.HexToHash(record.Secret))
		if err != nil {
			log.Error("RPC error, failed to unlock sBCH: ", err)
			continue
		}

		record.Status = Sbch2BchStatusSbchUnlocked
		record.SbchUnlockTxHash = toHex(txHash[:])
		err = bot.db.updateSbch2BchRecord(record)
		if err != nil {
			log.Error("DB error, failed to update status of Bch2SbchRecord: ", err)
		}
	}
}

func (bot *MarketMakerBot) handleBchRefunds(gotNewBlocks bool) {
	if !gotNewBlocks {
		return
	}

	log.Info("handle BCH refunds ...")

	// TODO: order by BchLockBlockNum ASC
	records, err := bot.db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 100)
	if err != nil {
		log.Error("DB error, failed to get SBCH2BCH records: ", err)
		return
	}
	log.Info("BchLocked SBCH2BCH records: ", len(records))

	for _, record := range records {
		log.Info("record: ", record.ID, ", txHash: ", record.BchLockTxHash)
		bchTimeLock := sbchTimeLockToBlocks(record.TimeLock) / 2
		//log.Info("BCH timeLock: ", bchTimeLock)

		confirmations, err := bot.bchCli.getTxConfirmations(record.BchLockTxHash)
		if err != nil {
			log.Error("RPC error, failed to get tx confirmations: ", err)
			continue
		}

		log.Info("confirmations: ", confirmations, " , bchTimeLock: ", bchTimeLock)
		if confirmations <= int64(bchTimeLock) {
			continue
		}

		covenant, err := htlcbch.NewMainnetCovenant(
			bot.bchPkh,
			gethcmn.FromHex(record.BchRecipientPkh),
			gethcmn.FromHex(record.HashLock),
			bchTimeLock,
			0,
		)
		if err != nil {
			log.Error("failed to create HTLC covenant: ", err)
			log.Error("record:", toJSON(record))
			continue
		}

		bchValMinusFee := int64(record.Value) * (10000 - int64(bot.serviceFeeRatio)) / 10000
		tx, err := covenant.MakeRefundTx(
			gethcmn.FromHex(record.BchLockTxHash),
			0,
			bchValMinusFee,
			bot.bchAddr,
			bot.bchRefundMinerFeeRate,
			bot.bchPrivKey,
		)
		if err != nil {
			log.Error("failed to make refund tx: ", err)
			continue
		}

		//record.Status = Sbch2BchStatusBchRefundable
		//err = bot.db.updateSbch2BchRecord(record)
		//if err != nil {
		//	log.Error("DB error, failed to save SBCH2BCH record: ", err)
		//}

		log.Info("refund tx: ", htlcbch.MsgTxToHex(tx))
		txHash, err := bot.bchCli.sendTx(tx)
		if err != nil {
			log.Error("failed to send refund tx: ", err)
			continue
		}

		log.Info("refund tx sent, hash: ", txHash.String())
		record.Status = Sbch2BchStatusBchRefunded
		record.BchRefundTxHash = txHash.String()
		err = bot.db.updateSbch2BchRecord(record)
		if err != nil {
			log.Error("DB error, failed to save SBCH2BCH record: ", err)
		}
	}
}

func (bot *MarketMakerBot) handleSbchRefunds() {
	log.Info("handle sBCH refunds ...")

	// TODO: order by SbchLockTxTime ASC
	records, err := bot.db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 100)
	if err != nil {
		log.Error("DB error, failed to get BCH2SBCH records: ", err)
		return
	}

	log.Info("SbchLocked BCH2SBCH records: ", len(records))
	if len(records) == 0 {
		return
	}

	localNow := time.Now().Unix()
	sbchNow, err := bot.sbchCli.getBlockTimeLatest()
	if err != nil {
		log.Error("RPC error, failed to get sBCH time: ", err)
		return
	}
	log.Info("localNow: ", localNow, ", sbchNow: ", sbchNow)

	for _, record := range records {
		log.Info("record: ", record.ID,
			" , SbchLockTxHash: ", record.SbchLockTxHash,
			" , SbchLockTxTime: ", record.SbchLockTxTime)
		sbchTimeLock := bchTimeLockToSeconds(record.TimeLock) / 2
		if uint64(localNow) < record.SbchLockTxTime+uint64(sbchTimeLock) {
			continue
		}

		txTime, err := bot.sbchCli.getTxTime(record.SbchLockTxHash)
		if err != nil {
			log.Error("RPC error, failed to get tx time: ", err)
			continue
		}

		if sbchNow <= txTime+uint64(sbchTimeLock) {
			log.Info("txTime: ", txTime)
			continue
		}

		//record.Status = Bch2SbchStatusSbchRefundable
		//err = bot.db.updateBch2SbchRecord(record)
		//if err != nil {
		//	log.Error("DB error, failed to update BCH2SBCH record: ", err)
		//}

		hashLock := gethcmn.HexToHash(record.HashLock)
		txHash, err := bot.sbchCli.refundSbchFromHtlc(hashLock)
		if err != nil {
			log.Error("RPC error, failed to refund sBCH: ", err)
			continue
		}

		record.Status = Bch2SbchStatusSbchRefunded
		record.SbchRefundTxHash = toHex(txHash.Bytes())
		err = bot.db.updateBch2SbchRecord(record)
		if err != nil {
			log.Error("DB error, failed to update BCH2SBCH record: ", err)
		}
	}
}

func secretToHashLock(secret []byte) string {
	hashLock := sha256.Sum256(secret)
	return toHex(hashLock[:])
}

func bchTimeLockToSeconds(nBlocks uint32) uint32 {
	return nBlocks * 10 * 60
}
func sbchTimeLockToBlocks(nSeconds uint32) uint16 {
	return uint16(nSeconds / (10 * 60))
}

func satsToWei(amt uint64) *big.Int {
	return big.NewInt(0).Mul(big.NewInt(int64(amt)), big.NewInt(1e10))
}
func weiToSats(amt *big.Int) uint64 {
	return big.NewInt(0).Div(amt, big.NewInt(1e10)).Uint64()
}

func utxoAmtToSats(amt float64) int64 {
	return int64(math.Round(amt * 1e8))
}

func toHex(bs []byte) string {
	return hex.EncodeToString(bs)
}

func toJSON(v interface{}) string {
	bs, _ := json.Marshal(v)
	return string(bs)
}
