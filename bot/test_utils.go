package bot

import (
	"bytes"
	"math/big"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/txscript"

	"github.com/smartbch/atomic-swap-bot/htlcbch"
)

func getHtlcP2shPkScript(senderPkh, recipientPkh, hashLock []byte, expiration, penaltyBPS uint16) []byte {
	c, err := htlcbch.NewMainnetCovenant(senderPkh, recipientPkh, hashLock, expiration, penaltyBPS)
	if err != nil {
		panic(err)
	}
	h, err := c.GetRedeemScriptHash()
	if err != nil {
		panic(err)
	}
	return newP2SHPkScript(h)
}

// OP_HASH160 <20 bytes script hash> OP_EQUAL
func newP2SHPkScript(pkh []byte) []byte {
	script, _ := txscript.NewScriptBuilder().
		AddOp(txscript.OP_HASH160).
		AddData(pkh).
		AddOp(txscript.OP_EQUAL).
		Script()
	return script
}

// OP_RETURN "SBAS" <recipient pkh> <sender pkh> <hash lock> <expiration> <penalty bps> <sbch user address>
func newHtlcDepositOpRet(recipientPkh, senderPkh, hashLock []byte,
	expiration, penaltyBPS uint16, evmAddr []byte) []byte {

	c, _ := htlcbch.NewTestnet3Covenant(senderPkh, recipientPkh, hashLock, expiration, penaltyBPS)
	opRetScript, _ := c.BuildOpRetPkScript(evmAddr)
	return opRetScript
}

func reverseBytes(bs []byte) []byte {
	n := len(bs)
	sb := make([]byte, n)
	for i := 0; i < n; i++ {
		sb[i] = bs[n-1-i]
	}
	return sb
}

//func reverseHex(s string) string {
//	s2 := hex.EncodeToString(reverseBytes(gethcmn.FromHex(s)))
//	if strings.HasPrefix(s, "0x") {
//		return "0x" + s2
//	}
//	return s2
//}

func joinBytes(s ...[]byte) []byte {
	return bytes.Join(s, nil)
}

func leftPad0(bs []byte, n int) []byte {
	return append(make([]byte, n), bs...)
}
func rightPad0(bs []byte, n int) []byte {
	return append(bs, make([]byte, n)...)
}

func gethAddr(s string) gethcmn.Address {
	addr := gethcmn.Address{}
	copy(addr[:], s[:])
	return addr
}
func gethAddrBytes(s string) []byte {
	return gethAddr(s).Bytes()
}

func gethHash32(s string) gethcmn.Hash {
	hash := gethcmn.Hash{}
	copy(hash[:], s)
	return hash
}
func gethHash32Bytes(s string) []byte {
	return gethHash32(s).Bytes()
}

func bchHash32(s string) chainhash.Hash {
	hash := chainhash.Hash{}
	copy(hash[:], s)
	return hash
}

func gethAddrToHash32(addr gethcmn.Address) gethcmn.Hash {
	return gethcmn.BytesToHash(leftPad0(addr.Bytes(), 12))
}

func int64ToBytes32(n int64) []byte {
	return big.NewInt(n).FillBytes(make([]byte, 32))
}

func satsToWeiBytes32(amt uint64) []byte {
	return satsToWei(amt).FillBytes(make([]byte, 32))
}
