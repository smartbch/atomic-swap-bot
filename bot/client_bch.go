package bot

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/rpcclient"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"

	log "github.com/sirupsen/logrus"
)

type IBchClient interface {
	GetBlockCount() (int64, error)
	GetBlock(height int64) (*btcjson.GetBlockVerboseTxResult, error)
	GetUTXOs(minVal, maxCount int64) ([]btcjson.ListUnspentResult, error)
	GetAllUTXOs() ([]btcjson.ListUnspentResult, error)
	GetTxConfirmations(txHashHex string) (int64, error)
	SendTx(tx *wire.MsgTx) (*chainhash.Hash, error)
}

type BchClient struct {
	client  *rpcclient.Client
	botAddr bchutil.Address
}

func NewBchClient(rpcUrlStr string, botAddr bchutil.Address) (*BchClient, error) {
	rpcUrl, err := url.Parse(rpcUrlStr)
	if err != nil {
		return nil, err
	}

	pass, _ := rpcUrl.User.Password()
	connCfg := &rpcclient.ConnConfig{
		Host:         rpcUrl.Host,
		User:         rpcUrl.User.Username(),
		Pass:         pass,
		DisableTLS:   rpcUrl.Scheme == "http",
		HTTPPostMode: true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	return &BchClient{client: client, botAddr: botAddr}, nil
}

func (c *BchClient) GetBlockCount() (int64, error) {
	return c.client.GetBlockCount()
}

func (c *BchClient) GetBlock(height int64) (*btcjson.GetBlockVerboseTxResult, error) {
	blockHash, err := c.client.GetBlockHash(height)
	if err != nil {
		return nil, nil
	}
	block, err := c.client.GetBlockVerboseTx(blockHash)
	if err != nil {
		return nil, nil
	}
	return block, err
}

func (c *BchClient) GetAllUTXOs() ([]btcjson.ListUnspentResult, error) {
	minConf := 0
	maxConf := 9999999
	return c.client.ListUnspentMinMaxAddresses(
		minConf, maxConf, []bchutil.Address{c.botAddr})
}

func (c *BchClient) GetUTXOs(minVal, maxCount int64) ([]btcjson.ListUnspentResult, error) {
	minConf := 0 //
	maxConf := 9999999
	allUTXOs, err := c.client.ListUnspentMinMaxAddresses(
		minConf, maxConf, []bchutil.Address{c.botAddr})
	if err != nil {
		return nil, err
	}

	return findUTXOs(allUTXOs, minVal, maxCount)
}

func findUTXOs(allUTXOs []btcjson.ListUnspentResult,
	minVal, maxCount int64) ([]btcjson.ListUnspentResult, error) {

	// try to find one
	for _, unspent := range allUTXOs {
		val := utxoAmtToSats(unspent.Amount)
		if val >= minVal {
			return []btcjson.ListUnspentResult{unspent}, nil
		}
	}

	// sort by value DESC
	slices.SortFunc(allUTXOs, func(a, b btcjson.ListUnspentResult) bool {
		return a.Amount > b.Amount
	})

	var totalAmt int64
	var utxos []btcjson.ListUnspentResult
	for _, unspent := range allUTXOs {
		totalAmt += utxoAmtToSats(unspent.Amount)
		utxos = append(utxos, unspent)
		if totalAmt >= minVal {
			break
		}
	}

	if totalAmt >= minVal && len(utxos) <= int(maxCount) {
		return utxos, nil
	}

	log.Info("allUTXOs:", toJSON(allUTXOs))
	return nil, fmt.Errorf(
		"no available UTXOs (minVal: %d sats, maxCount: %d)", minVal, maxCount)
}

func (c *BchClient) GetTxConfirmations(txHashHex string) (int64, error) {
	var txHash chainhash.Hash
	err := chainhash.Decode(&txHash, txHashHex)
	if err != nil {
		return 0, err
	}

	tx, err := c.client.GetRawTransactionVerbose(&txHash)
	if err != nil {
		return 0, err
	}
	return int64(tx.Confirmations), nil
}

func (c *BchClient) SendTx(tx *wire.MsgTx) (*chainhash.Hash, error) {
	return c.client.SendRawTransaction(tx, false)
}

func isUtxoSpentErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "-26: txn-mempool-conflict") ||
		strings.Contains(msg, "-27: transaction already in block chain") ||
		strings.Contains(msg, "-25: Missing inputs")
}
