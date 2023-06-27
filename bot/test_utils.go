package bot

import (
	"bytes"
	"encoding/binary"
	"github.com/gcash/bchd/txscript"

	"github.com/smartbch/atomic-swap/market-maker-bot/htlcbch"
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

// OP_RETURN "SBAS" <bot pkh> <user pkh> <hash lock> <expiration> <penalty bps> <sbch user address>
func newHtlcDepositOpRet(botPkh, userPkh, hashLock []byte,
	expiration, penaltyBPS uint16, evmAddr []byte) []byte {

	var timeLock [2]byte
	binary.BigEndian.PutUint16(timeLock[:], expiration)
	var penalty [2]byte
	binary.BigEndian.PutUint16(penalty[:], penaltyBPS)

	script, _ := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte{'S', 'B', 'A', 'S'}).
		AddData(botPkh).
		AddData(userPkh).
		AddData(hashLock).
		AddData(timeLock[:]).
		AddData(penalty[:]).
		AddData(evmAddr).
		Script()
	return script
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
