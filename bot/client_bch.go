package bot

import (
	"fmt"
	"net/url"

	"golang.org/x/exp/slices"

	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/rpcclient"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

type IBchClient interface {
	getBlockCount() (int64, error)
	getBlock(height int64) (*wire.MsgBlock, error)
	getUTXOs(minVal, maxCount int64) ([]btcjson.ListUnspentResult, error)
	getAllUTXOs() ([]btcjson.ListUnspentResult, error)
	getTxConfirmations(txHashHex string) (int64, error)
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

func (c *BchClient) getBlockCount() (int64, error) {
	return c.client.GetBlockCount()
}

func (c *BchClient) getBlock(height int64) (*wire.MsgBlock, error) {
	blockHash, err := c.client.GetBlockHash(height)
	if err != nil {
		return nil, nil
	}
	block, err := c.client.GetBlock(blockHash)
	if err != nil {
		return nil, nil
	}
	return block, err
}

func (c *BchClient) getAllUTXOs() ([]btcjson.ListUnspentResult, error) {
	minConf := 0
	maxConf := 9999999
	return c.client.ListUnspentMinMaxAddresses(
		minConf, maxConf, []bchutil.Address{c.botAddr})
}

func (c *BchClient) getUTXOs(minVal, maxCount int64) ([]btcjson.ListUnspentResult, error) {
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
	return nil, fmt.Errorf("no available UTXOs (minVal: %d sats)", minVal)
}

func (c *BchClient) getTxConfirmations(txHashHex string) (int64, error) {
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
