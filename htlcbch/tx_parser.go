package htlcbch

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/txscript"
)

const (
	protoID = "SBAS" // SmartBCH AtomicSwap
)

type HtlcLockInfo struct {
	//BlockNum      uint64
	TxHash        string        // 32 bytes, hex
	RecipientPkh  hexutil.Bytes // 20 bytes
	SenderPkh     hexutil.Bytes // 20 bytes
	HashLock      hexutil.Bytes // 32 bytes, sha256
	Expiration    uint16        //  2 bytes, big endian
	PenaltyBPS    uint16        //  2 bytes, big endian
	SenderEvmAddr hexutil.Bytes // 20 bytes
	ScriptHash    hexutil.Bytes // 20 bytes, hash160
	Value         uint64        // in sats
	ExpectedPrice uint64        // 8 decimals
}

type HtlcUnlockInfo struct {
	PrevTxHash string // 32 bytes, hex
	TxHash     string // 32 bytes, hex
	Secret     string // 32 bytes, hex
}

// === Lock ===

func GetHtlcLocksInfo(block *btcjson.GetBlockVerboseTxResult) (deposits []*HtlcLockInfo) {
	for _, tx := range block.Tx {
		depositInfo := isHtlcLockTx(tx)
		if depositInfo != nil {
			deposits = append(deposits, depositInfo)
		}
	}
	return
}

func decodeHex(s string) []byte {
	bz, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return bz
}

// output#0: deposit, output#1: op_return
func isHtlcLockTx(tx btcjson.TxRawResult) *HtlcLockInfo {
	if len(tx.Vout) < 2 {
		return nil
	}

	// output#0 must be locked by P2SH script
	scriptHash := getP2SHash(decodeHex(tx.Vout[0].ScriptPubKey.Hex))
	if scriptHash == nil {
		return nil
	}

	// output#1 must be NULL DATA that contains the HTLC info
	depositInfo := getHtlcLockInfo(decodeHex(tx.Vout[1].ScriptPubKey.Hex))
	if depositInfo == nil {
		return nil
	}

	c, err := NewMainnetCovenant(depositInfo.SenderPkh,
		depositInfo.RecipientPkh, depositInfo.HashLock,
		depositInfo.Expiration, depositInfo.PenaltyBPS)
	if err != nil {
		return nil
	}
	cScriptHash, err := c.GetRedeemScriptHash()
	if err != nil {
		return nil
	}
	if !bytes.Equal(cScriptHash, scriptHash) {
		return nil
	}

	depositInfo.TxHash = tx.Txid
	depositInfo.ScriptHash = scriptHash
	depositInfo.Value = utxoAmtToSats(tx.Vout[0].Value)
	return depositInfo
}

// https://github.com/bitcoincashorg/bitcoincash.org/blob/master/spec/op_return-prefix-guideline.md
// OP_RETURN "SBAS" <recipient pkh> <sender pkh> <hash lock> <expiration> <penalty bps> <sbch user address> <expected price>
func getHtlcLockInfo(pkScript []byte) *HtlcLockInfo {
	if len(pkScript) == 0 ||
		pkScript[0] != txscript.OP_RETURN {
		return nil
	}

	retData, err := txscript.PushedData(pkScript)
	if err != nil ||
		len(retData) != 8 ||
		string(retData[0]) != protoID || // "SBAS"
		len(retData[1]) != 20 || // recipient pkh
		len(retData[2]) != 20 || // sender pkh
		len(retData[3]) != 32 || // hash lock
		len(retData[4]) != 2 || // expiration
		len(retData[5]) != 2 || // penalty bps
		len(retData[6]) != 20 || // sender evm addr
		len(retData[7]) != 8 { // expected price

		return nil
	}

	return &HtlcLockInfo{
		RecipientPkh:  retData[1],
		SenderPkh:     retData[2],
		HashLock:      retData[3],
		Expiration:    binary.BigEndian.Uint16(retData[4]),
		PenaltyBPS:    binary.BigEndian.Uint16(retData[5]),
		SenderEvmAddr: retData[6],
		ExpectedPrice: binary.BigEndian.Uint64(retData[7]),
	}
}

// OP_HASH160 <20 bytes script hash> OP_EQUAL
func getP2SHash(pkScript []byte) (scriptHash []byte) {
	if len(pkScript) != 23 ||
		pkScript[0] != txscript.OP_HASH160 ||
		pkScript[1] != txscript.OP_DATA_20 ||
		pkScript[22] != txscript.OP_EQUAL {
		return nil
	}
	return pkScript[2:22]
}

// === Unlock ===

func GetHtlcUnlocksInfo(block *btcjson.GetBlockVerboseTxResult) (receipts []*HtlcUnlockInfo) {
	for _, tx := range block.Tx {
		receiptInfo := isHtlcUnlockTx(tx)
		if receiptInfo != nil {
			receipts = append(receipts, receiptInfo)
		}
	}
	return
}

func isHtlcUnlockTx(tx btcjson.TxRawResult) *HtlcUnlockInfo {
	if len(tx.Vin) != 1 {
		return nil
	}
	if tx.Vin[0].ScriptSig == nil {
		return nil
	}
	sigScript := decodeHex(tx.Vin[0].ScriptSig.Hex)
	receiptInfo := getHtlcUnlockInfo(sigScript)
	if receiptInfo != nil {
		receiptInfo.PrevTxHash = tx.Vin[0].Txid
		receiptInfo.TxHash = tx.Txid
	}
	return receiptInfo
}

func getHtlcUnlockInfo(sigScript []byte) *HtlcUnlockInfo {
	if !bytes.HasSuffix(sigScript, redeemScriptWithoutConstructorArgs) {
		return nil
	}
	pushes, err := txscript.PushedData(sigScript)
	if err != nil {
		return nil
	}
	if len(pushes) != 3 {
		return nil
	}
	if len(pushes[0]) != 32 {
		return nil
	}

	return &HtlcUnlockInfo{
		Secret: hex.EncodeToString(pushes[0]),
	}
}

// utils

func utxoAmtToSats(amt float64) uint64 {
	return uint64(math.Round(amt * 1e8))
}
