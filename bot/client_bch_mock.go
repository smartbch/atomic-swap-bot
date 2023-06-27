package bot

import (
	"fmt"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/wire"
)

type MockBchClient struct {
	hFrom         int64
	hTo           int64
	blocks        map[int64]*wire.MsgBlock
	confirmations map[string]int64
}

func newMockBchClient(hFrom, hTo int64) *MockBchClient {
	cli := &MockBchClient{
		hFrom:         hFrom,
		hTo:           hTo,
		blocks:        map[int64]*wire.MsgBlock{},
		confirmations: map[string]int64{},
	}
	for h := hFrom; h <= hTo; h++ {
		cli.blocks[h] = &wire.MsgBlock{}
	}
	return cli
}

func (c *MockBchClient) getBlockCount() (int64, error) {
	return c.hTo, nil
}

func (c *MockBchClient) getBlock(height int64) (*wire.MsgBlock, error) {
	if height < c.hFrom || height > c.hTo {
		return nil, fmt.Errorf("no block#%d", height)
	}
	return c.blocks[height], nil
}

func (*MockBchClient) getAllUTXOs() ([]btcjson.ListUnspentResult, error) {
	return nil, nil
}

func (c *MockBchClient) getUTXOs(minVal, maxCount int64) ([]btcjson.ListUnspentResult, error) {
	return []btcjson.ListUnspentResult{{
		TxID:   gethcmn.Hash{'f', 'a', 'k', 'e', 'u', 't', 'x', 'o'}.String(),
		Vout:   0,
		Amount: float64(minVal) * 2 / 1e8,
	}}, nil
}

func (c *MockBchClient) getTxConfirmations(txHashHex string) (int64, error) {
	return c.confirmations[txHashHex], nil
}

func (c *MockBchClient) sendTx(tx *wire.MsgTx) (*chainhash.Hash, error) {
	txHash := tx.TxHash()
	return &txHash, nil
}
