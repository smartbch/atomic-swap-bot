package htlcbch

import (
	"bytes"
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
	// cashc --hex ../../atomic-swap-covenants/covenants/HTLC3.cash
	RedeemScriptWithoutConstructorArgsHex = "0x5579009c635679827700a0635779a952798856795879ad670376a91452797e0288ac7e51c778887568c0009d587aa8537a886d6d6d755167557a519d5579827700a0635679a9788855795779ad670376a914787e0288ac7e51c778887568c0009d537ab275537900a06300c65479950210279600cc78a2690376a91453797e0288ac7e00cd78886d686d6d6d5168"
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

func (c *HtlcCovenant) MakeReceiveTx(
	txid []byte, vout uint32, inAmt int64, // input info
	toAddr bchutil.Address, minerFeeRate uint64, // output info
	secret []byte,
	privKey *bchec.PrivateKey,
) (*wire.MsgTx, error) {
	// estimate miner fee
	tx, err := c.makeReceiveOrRefundTx(txid, vout, inAmt, toAddr, 1000, secret, privKey)
	if err != nil {
		return nil, err
	}
	// make tx
	minerFee := int64(len(MsgTxToBytes(tx))) * int64(minerFeeRate)
	return c.makeReceiveOrRefundTx(txid, vout, inAmt, toAddr, minerFee, secret, privKey)
}

func (c *HtlcCovenant) MakeRefundTx(
	txid []byte, vout uint32, inAmt int64, // input info
	toAddr bchutil.Address, minerFeeRate uint64, // output info
	privKey *bchec.PrivateKey,
) (*wire.MsgTx, error) {
	// estimate miner fee
	tx, err := c.makeReceiveOrRefundTx(txid, vout, inAmt, toAddr, 1000, nil, privKey)
	if err != nil {
		return nil, err
	}
	// make tx
	minerFee := int64(len(MsgTxToBytes(tx))) * int64(minerFeeRate)
	return c.makeReceiveOrRefundTx(txid, vout, inAmt, toAddr, minerFee, nil, privKey)
}

func (c *HtlcCovenant) makeReceiveOrRefundTx(
	txid []byte, vout uint32, inAmt int64, // input info
	toAddr bchutil.Address, minerFee int64, // output info
	secret []byte,
	privKey *bchec.PrivateKey,
) (*wire.MsgTx, error) {

	pbk := privKey.PubKey().SerializeCompressed()
	pkh := bchutil.Hash160(pbk)
	isReceive := bytes.Equal(pkh, c.recipientPkh)

	// check args
	if isReceive {
		if len(secret) != 32 {
			return nil, fmt.Errorf("secret is not 32 bytes")
		}
	} else { // refund
		if !bytes.Equal(pkh, c.senderPkh) {
			return nil, fmt.Errorf("wrong priv key")
		}
	}

	seq := uint32(0)
	if !isReceive {
		seq = uint32(c.expiration)
	}

	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	sigScriptFn := func(sig []byte) ([]byte, error) {
		if isReceive {
			return c.BuildReceiveSigScript(sig, pbk, secret)
		}
		return c.BuildRefundSigScript(sig, pbk)
	}

	return newMsgTxBuilder().
		addInput(txid, vout, seq).
		addOutput(toAddr, inAmt-minerFee).
		sign(0, inAmt, redeemScript, privKey, sigScriptFn).
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

	sigScriptFn := func(sig []byte) ([]byte, error) {
		return payToPubKeyHashSigScript(sig, fromPk)
	}

	builder := newMsgTxBuilder()
	var totalInAmt int64
	for _, input := range inputs {
		builder.addInput(input.TxID, input.Vout, 0)
		totalInAmt += input.Amount
	}
	changeAmt := totalInAmt - outAmt - minerFee
	if changeAmt < 0 {
		return nil, fmt.Errorf("insufficient input value: %d < %d", totalInAmt, outAmt+minerFee)
	}
	builder.addOutput(toAddr, outAmt)
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

func (c *HtlcCovenant) BuildReceiveSigScript(recipientSig, recipientPk, secret []byte) ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	return txscript.NewScriptBuilder().
		AddData(secret).
		AddData(recipientPk).
		AddData(recipientSig).
		AddInt64(0). // selector
		AddData(redeemScript).
		Script()
}

func (c *HtlcCovenant) BuildRefundSigScript(senderSig, senderPk []byte) ([]byte, error) {
	redeemScript, err := c.BuildFullRedeemScript()
	if err != nil {
		return nil, err
	}

	return txscript.NewScriptBuilder().
		AddData(senderPk).
		AddData(senderSig).
		AddInt64(1). // selector
		AddData(redeemScript).
		Script()
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
