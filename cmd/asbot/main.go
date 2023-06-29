package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"

	goecies "github.com/ecies/go"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/btcjson"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/atomic-swap-bot/bot"
)

var (
	dbFile            = "bot.db"
	bchPrivKeyWIF     = "" // only used for test
	sbchPrivKeyHex    = "" // only used for test
	bchRpcUrl         = "https://user:pass@localhost:8333"
	sbchRpcUrl        = "https://localhost:8545"
	sbchHtlcAddr      = "0x"
	sbchGasPrice      = 1.05
	bchTimeLock       = uint64(12 * 6)    // 12h
	sbchTimeLock      = uint64(12 * 3600) // 12h
	penaltyRatio      = uint64(500)       // 5%
	feeRatio          = uint64(100)       // 1%
	minSwapVal        = 0.0               // in BCH
	maxSwapVal        = 5.0               // in BCH
	bchSendFeeRate    = uint64(2)         // sats/byte
	bchReceiveFeeRate = uint64(2)         // sats/byte
	bchRefundFeeRate  = uint64(2)         // sats/byte
	sbchOpenGas       = uint64(500_000)
	sbchCloseGas      = uint64(500_000)
	sbchExpireGas     = uint64(500_000)
	bchConfirmations  = uint64(10)
	debugMode         = true
)

func main() {
	flag.StringVar(&dbFile, "db-file", dbFile, "sqlite3 database file")
	flag.StringVar(&bchPrivKeyWIF, "bch-key", bchPrivKeyWIF, "BCH private key (WIF, only used for test)")
	flag.StringVar(&sbchPrivKeyHex, "sbch-key", sbchPrivKeyHex, "sBCH private key (hex, only used for test)")
	flag.StringVar(&bchRpcUrl, "bch-rpc-url", bchRpcUrl, "BCH RPC URL")
	flag.StringVar(&sbchRpcUrl, "sbch-rpc-url", sbchRpcUrl, "sBCH RPC URL")
	flag.StringVar(&sbchHtlcAddr, "sbch-htlc-addr", sbchHtlcAddr, "sBCH HTLC contract address")
	flag.Float64Var(&sbchGasPrice, "sbch-gas-price", sbchGasPrice, "sBCH gas price (in Gwei)")
	flag.Uint64Var(&bchTimeLock, "bch-timelock", bchTimeLock, "BCH HTLC time-lock (in blocks)")
	flag.Uint64Var(&sbchTimeLock, "sbch-timelock", sbchTimeLock, "sBCH HTLC time-lock (in seconds)")
	flag.Uint64Var(&penaltyRatio, "penalty-ratio", penaltyRatio, "penalty ratio of HTLC refund (in BPS)")
	flag.Uint64Var(&feeRatio, "fee-ratio", feeRatio, "service fee ratio (in BPS)")
	flag.Float64Var(&minSwapVal, "min-swap-val", minSwapVal, "min swap value (in BCH)")
	flag.Float64Var(&maxSwapVal, "max-swap-val", maxSwapVal, "max swap value (in BCH)")
	flag.Uint64Var(&bchConfirmations, "bch-confirmations", bchConfirmations, "required confirmations of BCH tx ")
	flag.Uint64Var(&bchSendFeeRate, "bch-send-fee-rate", bchSendFeeRate, "miner fee rate of BCH HTLC sending tx (Sats/byte)")
	flag.Uint64Var(&bchReceiveFeeRate, "bch-receive-fee-rate", bchReceiveFeeRate, "miner fee rate of BCH HTLC receiving tx (Sats/byte)")
	flag.Uint64Var(&bchRefundFeeRate, "bch-refund-fee-rate", bchReceiveFeeRate, "miner fee rate of BCH HTLC refund tx (Sats/byte)")
	flag.Uint64Var(&sbchOpenGas, "sbch-open-gas", sbchOpenGas, "gas limit of sBCH HTLC open tx")
	flag.Uint64Var(&sbchCloseGas, "sbch-close-gas", sbchCloseGas, "gas limit of sBCH HTLC close tx")
	flag.Uint64Var(&sbchExpireGas, "sbch-expire-gas", sbchExpireGas, "gas limit of sBCH HTLC expire tx")
	flag.BoolVar(&debugMode, "debug", debugMode, "debug mode")
	flag.Parse()

	if bchPrivKeyWIF == "" || sbchPrivKeyHex == "" || !debugMode {
		bchPrivKeyWIF, sbchPrivKeyHex = readKeys()
	}

	_sbchHtlcAddr := gethcmn.HexToAddress(sbchHtlcAddr)
	_sbchGasPrice := big.NewInt(int64(sbchGasPrice * 1e9))

	_bot, err := bot.NewBot(dbFile, bchPrivKeyWIF, sbchPrivKeyHex,
		bchRpcUrl, sbchRpcUrl, _sbchHtlcAddr, _sbchGasPrice,
		uint16(bchTimeLock), uint32(sbchTimeLock), uint16(penaltyRatio), uint16(feeRatio),
		uint64(math.Round(minSwapVal*1e8)), uint64(math.Round(maxSwapVal*1e8)),
		uint8(bchConfirmations),
		bchSendFeeRate, bchReceiveFeeRate, bchRefundFeeRate,
		sbchOpenGas, sbchCloseGas, sbchExpireGas,
		debugMode,
	)
	if err != nil {
		log.Fatal(err)
	}

	utxos, err := _bot.GetUTXOs()
	if err != nil {
		log.Fatal(err)
	}
	printUTXOs(utxos)

	_bot.PrepareDB()
	_bot.Loop()
}

func printUTXOs(utxos []btcjson.ListUnspentResult) {
	log.Info("BCH UTXOs:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"TXID", "vout", "value", "confirmations"})
	for _, utxo := range utxos {
		table.Append([]string{
			utxo.TxID,
			fmt.Sprintf("%d", utxo.Vout),
			fmt.Sprintf("%f", utxo.Amount),
			fmt.Sprintf("%d", utxo.Confirmations),
		})
	}
	table.Render() // Send output
}

func readKeys() (bchWIF, sbchKey string) {
	eciesPrivKey, err := goecies.GenerateKey()
	if err != nil {
		log.Fatal("failed to gen ecies key: ", err)
	}
	fmt.Println("The ecies pubkey:", hex.EncodeToString(eciesPrivKey.PublicKey.Bytes(true)))

	bchWIF = readKey(eciesPrivKey, "BCH WIF (ASIC)")
	sbchKey = readKey(eciesPrivKey, "sBCH Key (HEX)")
	return
}

func readKey(key *goecies.PrivateKey, keyName string) string {
	var inputHex string
	fmt.Printf("Enter the encrypted %s: ", keyName)
	_, _ = fmt.Scanf("%s", &inputHex)
	bz, err := hex.DecodeString(inputHex)
	if err != nil {
		log.Fatal("cannot decode hex string: ", err)
	}
	bz, err = goecies.Decrypt(key, bz)
	if err != nil {
		log.Fatal("cannot decrypt: ", err)
	}
	//println(string(bz))
	return string(bz)
}
