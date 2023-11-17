package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"

	goecies "github.com/ecies/go"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/btcjson"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/smartbch/atomic-swap-bot/bot"
)

var (
	dbFile           = "bot.db"
	bchPrivKeyWIF    = "" // only used for test
	sbchPrivKeyHex   = "" // only used for test
	bchMasterAddr    = "" // only in slave mode
	sbchMasterAddr   = "" // only in slave mode
	bchRpcUrl        = "https://user:pass@localhost:8333"
	sbchRpcUrl       = "https://localhost:8545"
	sbchHtlcAddr     = "0x"
	sbchGasPrice     = 1.05
	bchLockFeeRate   = uint64(2) // sats/byte
	bchUnlockFeeRate = uint64(2) // sats/byte
	bchRefundFeeRate = uint64(2) // sats/byte
	bchConfirmations = uint64(10)
	dbQueryLimit     = uint64(100)
	debugMode        = false
	slaveMode        = false
	lazyMaster       = false
	rpcListenAddr    = ""
	rollingLogFile   = ""
	rollingLogSize   = uint64(100)
)

func main() {
	flag.StringVar(&dbFile, "db-file", dbFile, "sqlite3 database file")
	flag.StringVar(&bchPrivKeyWIF, "bch-key", bchPrivKeyWIF, "BCH private key (WIF, only used for test)")
	flag.StringVar(&sbchPrivKeyHex, "sbch-key", sbchPrivKeyHex, "sBCH private key (hex, only used for test)")
	flag.StringVar(&bchMasterAddr, "bch-master-addr", bchMasterAddr, "BCH master address (only in slave mode)")
	flag.StringVar(&sbchMasterAddr, "sbch-master-addr", sbchMasterAddr, "SBCH master address (only in slave mode)")
	flag.StringVar(&bchRpcUrl, "bch-rpc-url", bchRpcUrl, "BCH RPC URL")
	flag.StringVar(&sbchRpcUrl, "sbch-rpc-url", sbchRpcUrl, "sBCH RPC URL")
	flag.StringVar(&sbchHtlcAddr, "sbch-htlc-addr", sbchHtlcAddr, "sBCH HTLC contract address")
	flag.Float64Var(&sbchGasPrice, "sbch-gas-price", sbchGasPrice, "sBCH gas price (in Gwei)")
	flag.Uint64Var(&bchConfirmations, "bch-confirmations", bchConfirmations, "required confirmations of BCH tx ")
	flag.Uint64Var(&bchLockFeeRate, "bch-lock-fee-rate", bchLockFeeRate, "miner fee rate of BCH HTLC lock tx (Sats/byte)")
	flag.Uint64Var(&bchUnlockFeeRate, "bch-unlock-fee-rate", bchUnlockFeeRate, "miner fee rate of BCH HTLC unlock tx (Sats/byte)")
	flag.Uint64Var(&bchRefundFeeRate, "bch-refund-fee-rate", bchUnlockFeeRate, "miner fee rate of BCH HTLC refund tx (Sats/byte)")
	flag.Uint64Var(&dbQueryLimit, "db-query-limit", dbQueryLimit, "db query limit")
	flag.BoolVar(&debugMode, "debug", debugMode, "debug mode")
	flag.BoolVar(&slaveMode, "slave", slaveMode, "slave mode")
	flag.BoolVar(&lazyMaster, "lazy-master", lazyMaster, "delay to send unlock|refund tx (debug mode only)")
	flag.StringVar(&rpcListenAddr, "rpc-listen-addr", rpcListenAddr, "host:port (will start RPC server if this option is not empty)")
	flag.StringVar(&rollingLogFile, "rolling-log-file", rollingLogFile, "path of rolling log file")
	flag.Uint64Var(&rollingLogSize, "rolling-log-size", rollingLogSize, "max size of rolling log file, in MB")
	flag.Parse()

	if rollingLogFile != "" {
		log.Info("logs are written to:", rollingLogFile)

		// https://stackoverflow.com/questions/28796021/how-can-i-log-in-golang-to-a-file-with-log-rotation
		// https://github.com/natefinch/lumberjack
		log.SetOutput(&lumberjack.Logger{
			Filename:   rollingLogFile,
			MaxSize:    int(rollingLogSize),
			MaxBackups: 3,    //
			MaxAge:     28,   // days
			Compress:   true, // disabled by default
		})
	}

	if bchPrivKeyWIF == "" || sbchPrivKeyHex == "" {
		bchPrivKeyWIF, sbchPrivKeyHex = readKeys(slaveMode)
	}

	_sbchHtlcAddr := gethcmn.HexToAddress(sbchHtlcAddr)
	_sbchGasPrice := big.NewInt(int64(sbchGasPrice * 1e9))

	_bot, err := bot.NewBot(dbFile, bchPrivKeyWIF, sbchPrivKeyHex,
		bchMasterAddr, sbchMasterAddr,
		bchRpcUrl, sbchRpcUrl, _sbchHtlcAddr, _sbchGasPrice,
		uint8(bchConfirmations),
		bchLockFeeRate, bchUnlockFeeRate, bchRefundFeeRate,
		int(dbQueryLimit),
		debugMode, slaveMode, lazyMaster,
	)
	if err != nil {
		log.Fatal("failed to create bot: ", err)
	}

	utxos, err := _bot.GetUTXOs()
	if err != nil {
		log.Fatal("failed to query BCH UTXOs: ", err)
	}
	printUTXOs(utxos)

	_bot.PrepareDB()

	if rpcListenAddr != "" {
		go _bot.StartHttpServer(rpcListenAddr)
	}

	_bot.Loop()
}

func printUTXOs(utxos []btcjson.ListUnspentResult) {
	log.Info("BCH UTXOs:")
	table := tablewriter.NewWriter(log.StandardLogger().Out)
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

func readKeys(slaveMode bool) (bchWIF, sbchKey string) {
	eciesPrivKey, err := goecies.GenerateKey()
	if err != nil {
		log.Fatal("failed to gen ecies key: ", err)
	}
	fmt.Println("The ecies pubkey:",
		hex.EncodeToString(eciesPrivKey.PublicKey.Bytes(true)))

	if !slaveMode {
		// BCH key is only used by master bot
		bchWIF = readKey(eciesPrivKey, "BCH WIF")
	}
	// sBCH key is used by both master and slave bots
	sbchKey = readKey(eciesPrivKey, "sBCH Key")
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
