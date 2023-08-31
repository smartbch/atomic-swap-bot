package htlcbch

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
)

const (
	// cashc --hex ../atomic-swap-covenants/covenants/HTLC4.cash
	RedeemScriptWithoutConstructorArgsHex = "0x5579009c63c0009d567aa8537a880376a9147b7e0288ac7e00cd8800cc00c602d00794a2696d6d5167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975680376a914547a7e0288ac7e00cd8800cc7b02d00794a2696d6d755168"
)

var (
	redeemScriptWithoutConstructorArgs = gethcmn.FromHex(RedeemScriptWithoutConstructorArgsHex)
)

type InputInfo struct {
	TxID   []byte
	Vout   uint32
	Amount int64
}

type HtlcCovenant struct {
	senderPkh    []byte // 20 bytes
	recipientPkh []byte // 20 bytes
	hashLock     []byte // 32 bytes
	expiration   uint16
	penaltyBPS   uint16
	net          *chaincfg.Params
}

func NewMainnetCovenant(
	senderPkh, recipientPkh, hashLock []byte, expiration, penaltyBPS uint16,
) (*HtlcCovenant, error) {

	return NewCovenant(senderPkh, recipientPkh, hashLock, expiration, penaltyBPS,
		&chaincfg.MainNetParams)
}

func NewTestnet3Covenant(
	senderPkh, recipientPkh, hashLock []byte, expiration, penaltyBPS uint16,
) (*HtlcCovenant, error) {

	return NewCovenant(senderPkh, recipientPkh, hashLock, expiration, penaltyBPS,
		&chaincfg.TestNet3Params)
}

func NewCovenant(
	senderPkh, recipientPkh, hashLock []byte, expiration, penaltyBPS uint16,
	net *chaincfg.Params,
) (*HtlcCovenant, error) {

	if len(senderPkh) != 20 {
		return nil, fmt.Errorf("senderPkh is not 20 bytes")
	}
	if len(recipientPkh) != 20 {
		return nil, fmt.Errorf("recipientPkh is not 20 bytes")
	}
	if len(hashLock) != 32 {
		return nil, fmt.Errorf("hashLock is not 32 bytes")
	}

	return &HtlcCovenant{
		senderPkh:    senderPkh,
		recipientPkh: recipientPkh,
		hashLock:     hashLock,
		expiration:   expiration,
		penaltyBPS:   penaltyBPS,
		net:          net,
	}, nil
}

func (c *HtlcCovenant) String() string {
	return "HtlcCovenant {" +
		"senderPkh: " + hex.EncodeToString(c.senderPkh) +
		", recipientPkh: " + hex.EncodeToString(c.recipientPkh) +
		", hashLock: " + hex.EncodeToString(c.hashLock) +
		", expiration: " + fmt.Sprintf("%d", c.expiration) +
		", penaltyBPS: " + fmt.Sprintf("%d", c.penaltyBPS) +
		"}"
}

func (c *HtlcCovenant) GetRedeemScriptHash() ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}
	return bchutil.Hash160(redeemScript), nil
}

func (c *HtlcCovenant) GetP2SHAddress() (string, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return "", err
	}

	redeemHash := bchutil.Hash160(redeemScript)
	addr, err := bchutil.NewAddressScriptHashFromHash(redeemHash, c.net)
	if err != nil {
		return "", err
	}

	return c.net.CashAddressPrefix + ":" + addr.EncodeAddress(), nil
}

func (c *HtlcCovenant) MakeUnlockTx(
	txid []byte, vout uint32, inAmt int64, // input info
	minerFeeRate uint64,
	secret []byte,
) (*wire.MsgTx, error) {
	// estimate miner fee
	tx, err := c.makeUnlockTx(txid, vout, inAmt, secret, 1000)
	if err != nil {
		return nil, err
	}
	// make tx
	minerFee := int64(len(MsgTxToBytes(tx))) * int64(minerFeeRate)
	return c.makeUnlockTx(txid, vout, inAmt, secret, minerFee)
}

func (c *HtlcCovenant) MakeRefundTx(
	txid []byte, vout uint32, inAmt int64, // input info
	minerFeeRate uint64,
) (*wire.MsgTx, error) {
	// estimate miner fee
	tx, err := c.makeRefundTx(txid, vout, inAmt, 1000)
	if err != nil {
		return nil, err
	}
	// make tx
	minerFee := int64(len(MsgTxToBytes(tx))) * int64(minerFeeRate)
	return c.makeRefundTx(txid, vout, inAmt, minerFee)
}

func (c *HtlcCovenant) makeUnlockTx(
	txid []byte, vout uint32, inAmt int64, // input info
	secret []byte,
	minerFee int64,
) (*wire.MsgTx, error) {

	if len(secret) != 32 {
		return nil, fmt.Errorf("secret is not 32 bytes")
	}

	seq := uint32(0)

	sigScript, err := c.BuildUnlockSigScript(secret)
	if err != nil {
		return nil, err
	}

	toAddr, err := bchutil.NewAddressPubKeyHash(c.recipientPkh, c.net)
	if err != nil {
		return nil, err
	}

	return newMsgTxBuilder().
		addInput(txid, vout, seq, sigScript).
		addOutput(toAddr, inAmt-minerFee).
		build()
}

func (c *HtlcCovenant) makeRefundTx(
	txid []byte, vout uint32, inAmt int64, // input info
	minerFee int64,
) (*wire.MsgTx, error) {

	seq := uint32(c.expiration)

	sigScript, err := c.BuildRefundSigScript()
	if err != nil {
		return nil, err
	}

	senderAddr, err := bchutil.NewAddressPubKeyHash(c.senderPkh, c.net)
	if err != nil {
		return nil, err
	}

	// no penalty
	if c.penaltyBPS == 0 {
		return newMsgTxBuilder().
			addInput(txid, vout, seq, sigScript).
			addOutput(senderAddr, inAmt-minerFee).
			build()
	}

	// consider penalty

	recipientAddr, err := bchutil.NewAddressPubKeyHash(c.recipientPkh, c.net)
	if err != nil {
		return nil, err
	}

	penaltyVal := inAmt * int64(c.penaltyBPS) / 10000
	if penaltyVal < 546 {
		penaltyVal = 546
	}

	return newMsgTxBuilder().
		addInput(txid, vout, seq, sigScript).
		addOutput(senderAddr, inAmt-penaltyVal-minerFee).
		addOutput(recipientAddr, penaltyVal).
		build()
}

func (c *HtlcCovenant) MakeLockTx(
	fromKey *bchec.PrivateKey,
	inputs []InputInfo, // inputs info
	outAmt int64, // output info
	minerFeeRate uint64,
) (*wire.MsgTx, error) {
	// estimate miner fee
	tx, err := c.makeLockTx(fromKey, inputs, outAmt, 1000)
	if err != nil {
		return nil, err
	}
	// make tx
	minerFee := int64(len(MsgTxToBytes(tx))) * int64(minerFeeRate)
	return c.makeLockTx(fromKey, inputs, outAmt, minerFee)
}

func (c *HtlcCovenant) makeLockTx(
	fromKey *bchec.PrivateKey,
	inputs []InputInfo, // inputs info
	outAmt int64, // output info
	minerFee int64,
) (*wire.MsgTx, error) {
	fromPk := fromKey.PubKey().SerializeCompressed()
	fromPkh := bchutil.Hash160(fromPk)

	script, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, fmt.Errorf("failed to build full redeem script: %d", err)
	}

	toAddr, err := bchutil.NewAddressScriptHash(script, c.net)
	if err != nil {
		return nil, fmt.Errorf("failed to calc p2sh address: %d", err)
	}

	changeAddr, err := bchutil.NewAddressPubKeyHash(fromPkh, c.net)
	if err != nil {
		return nil, fmt.Errorf("failed to calc p2pkh address: %w", err)
	}

	prevPkScript, err := payToPubKeyHashPkScript(fromPkh)
	if err != nil {
		return nil, fmt.Errorf("failed to creatte pkScript: %w", err)
	}

	opRetScript, err := c.BuildOpRetPkScript(make([]byte, 20), 1e8)
	if err != nil {
		return nil, fmt.Errorf("failed to build OP_RETURN: %w", err)
	}

	sigScriptFn := func(sig []byte) ([]byte, error) {
		return payToPubKeyHashSigScript(sig, fromPk)
	}

	builder := newMsgTxBuilder()
	var totalInAmt int64
	for _, input := range inputs {
		builder.addInput(input.TxID, input.Vout, 0, nil)
		totalInAmt += input.Amount
	}
	changeAmt := totalInAmt - outAmt - minerFee
	if changeAmt < 0 {
		return nil, fmt.Errorf("insufficient input value: %d < %d", totalInAmt, outAmt+minerFee)
	}
	builder.addOutput(toAddr, outAmt)
	builder.addOpRet(opRetScript)
	builder.addChange(changeAddr, changeAmt)
	for i, utxo := range inputs {
		builder.sign(i, utxo.Amount, prevPkScript, fromKey, sigScriptFn)
	}
	return builder.build()
}

func (c *HtlcCovenant) BuildFullRedeemScript() ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddInt64(int64(c.penaltyBPS)).
		AddInt64(int64(c.expiration)).
		AddData(c.hashLock).
		AddData(c.recipientPkh).
		AddData(c.senderPkh).
		AddOps(redeemScriptWithoutConstructorArgs).
		Script()
}

func (c *HtlcCovenant) BuildUnlockSigScript(secret []byte) ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	return txscript.NewScriptBuilder().
		AddData(secret).
		AddInt64(0). // selector
		AddData(redeemScript).
		Script()
}

func (c *HtlcCovenant) BuildRefundSigScript() ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	return txscript.NewScriptBuilder().
		AddInt64(1). // selector
		AddData(redeemScript).
		Script()
}

// OP_RETURN "SBAS" <recipient pkh> <sender pkh> <hash lock> <expiration> <penalty bps> <sbch user address> <expected price>
func (c *HtlcCovenant) BuildOpRetPkScript(sbchUserAddr []byte,
	expectedPrice uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte(protoID)).
		AddData(c.recipientPkh).
		AddData(c.senderPkh).
		AddData(c.hashLock).
		AddData(encodeBE16(c.expiration)).
		AddData(encodeBE16(c.penaltyBPS)).
		AddData(sbchUserAddr).
		AddData(encodeBE64(expectedPrice)).
		Script()
}

func encodeBE16(n uint16) []byte {
	buf := [2]byte{}
	binary.BigEndian.PutUint16(buf[:], n)
	return buf[:]
}
func encodeBE64(n uint64) []byte {
	buf := [8]byte{}
	binary.BigEndian.PutUint64(buf[:], n)
	return buf[:]
}

func payToPubKeyHashSigScript(sig, pk []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().AddData(sig).AddData(pk).Script()
}

func payToPubKeyHashPkScript(pubKeyHash []byte) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_DUP).
		AddOp(txscript.OP_HASH160).
		AddData(pubKeyHash).
		AddOp(txscript.OP_EQUALVERIFY).
		AddOp(txscript.OP_CHECKSIG).
		Script()
}

//func payToScriptHashPkScript(scriptHash []byte) ([]byte, error) {
//	return txscript.NewScriptBuilder().
//		AddOp(txscript.OP_HASH160).
//		AddData(scriptHash).
//		AddOp(txscript.OP_EQUAL).
//		Script()
//}
