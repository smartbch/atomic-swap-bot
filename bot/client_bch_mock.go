package bot

import (
	"fmt"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/wire"
)

var _ IBchClient = (*MockBchClient)(nil)

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

func (c *MockBchClient) GetBlockCount() (int64, error) {
	return c.hTo, nil
}

func (c *MockBchClient) GetBlock(height int64) (*btcjson.GetBlockVerboseTxResult, error) {
	if height < c.hFrom || height > c.hTo {
		return nil, fmt.Errorf("no block#%d", height)
	}
	return msgBlockToVerbose(c.blocks[height]), nil
}

func (*MockBchClient) GetAllUTXOs() ([]btcjson.ListUnspentResult, error) {
	return nil, nil
}

func (c *MockBchClient) GetUTXOs(minVal, maxCount int64) ([]btcjson.ListUnspentResult, error) {
	return []btcjson.ListUnspentResult{{
		TxID:   gethcmn.Hash{'f', 'a', 'k', 'e', 'u', 't', 'x', 'o'}.String(),
		Vout:   0,
		Amount: float64(minVal) * 2 / 1e8,
	}}, nil
}

func (c *MockBchClient) GetTxConfirmations(txHashHex string) (int64, error) {
	return c.confirmations[txHashHex], nil
}

func (c *MockBchClient) SendTx(tx *wire.MsgTx) (*chainhash.Hash, error) {
	txHash := tx.TxHash()
	return &txHash, nil
}

func msgBlockToVerbose(block *wire.MsgBlock) *btcjson.GetBlockVerboseTxResult {
	return &btcjson.GetBlockVerboseTxResult{
		Tx: cast(block.Transactions, msgTxToVerbose),
	}
}
func msgTxToVerbose(tx *wire.MsgTx) btcjson.TxRawResult {
	return btcjson.TxRawResult{
		Txid: tx.TxHash().String(),
		Vin:  cast(tx.TxIn, txInToVin),
		Vout: cast(tx.TxOut, txOutToVout),
	}
}
func txInToVin(txIn *wire.TxIn) btcjson.Vin {
	return btcjson.Vin{
		Txid: txIn.PreviousOutPoint.Hash.String(),
		Vout: txIn.PreviousOutPoint.Index,
		ScriptSig: &btcjson.ScriptSig{
			Hex: toHex(txIn.SignatureScript),
		},
	}
}
func txOutToVout(txIn *wire.TxOut) btcjson.Vout {
	return btcjson.Vout{
		Value: float64(txIn.Value) / 1e8,
		ScriptPubKey: btcjson.ScriptPubKeyResult{
			Hex: toHex(txIn.PkScript),
		},
	}
}

func cast[A, B any](s []A, fn func(A) B) []B {
	t := make([]B, len(s))
	for i, x := range s {
		t[i] = fn(x)
	}
	return t
}
