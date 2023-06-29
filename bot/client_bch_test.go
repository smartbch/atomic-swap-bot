package bot

import (
	"math/rand"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/gcash/bchd/btcjson"
	"github.com/stretchr/testify/require"
)

func TestFindUTXOs(t *testing.T) {
	allUTXOs := []btcjson.ListUnspentResult{
		{TxID: "tx0", Vout: 0, Amount: 0.01},
		{TxID: "tx1", Vout: 1, Amount: 0.1},
		{TxID: "tx2", Vout: 2, Amount: 0.2},
		{TxID: "tx3", Vout: 3, Amount: 0.3},
		{TxID: "tx4", Vout: 4, Amount: 0.4},
		{TxID: "tx5", Vout: 5, Amount: 0.5},
		{TxID: "tx6", Vout: 6, Amount: 0.6},
		{TxID: "tx7", Vout: 7, Amount: 0.7},
		{TxID: "tx8", Vout: 8, Amount: 0.8},
		{TxID: "tx9", Vout: 9, Amount: 0.9},
	}

	// 0.25
	utxos, err := findUTXOs(allUTXOs, 25000000, 5)
	require.NoError(t, err)
	require.Equal(t, []btcjson.ListUnspentResult{
		{TxID: "tx3", Vout: 3, Amount: 0.3},
	}, utxos)

	// shuffle
	slices.SortFunc(allUTXOs, func(a, b btcjson.ListUnspentResult) bool {
		return rand.Int()%2 == 0
	})

	// 2.5
	utxos, err = findUTXOs(allUTXOs, 250000000, 5)
	require.NoError(t, err)
	require.Equal(t, []btcjson.ListUnspentResult{
		{TxID: "tx9", Vout: 9, Amount: 0.9},
		{TxID: "tx8", Vout: 8, Amount: 0.8},
		{TxID: "tx7", Vout: 7, Amount: 0.7},
		{TxID: "tx6", Vout: 6, Amount: 0.6},
	}, utxos)

	_, err = findUTXOs(allUTXOs, 250000000, 3)
	require.ErrorContains(t, err, "no available UTXOs (minVal: 250000000 sats)")
}

//func TestGetTxConfirmations(t *testing.T) {
//	addr, err := bchutil.DecodeAddress("bchtest:qqgy70efq403k2mda04ku6dx7r2nfuq4s5u6xh83hw", &chaincfg.TestNet3Params)
//	require.NoError(t, err)
//	bchCli, err := newBchClient("http://user:pass@127.0.0.1:48334", addr)
//	require.NoError(t, err)
//	n, err := bchCli.getTxConfirmations("2161067da799837b10f175bbfdfcb1beea19a36606b1239c1ee31871eea6c1c0")
//	require.NoError(t, err)
//	require.Equal(t, n, 100)
//}
