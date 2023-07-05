package htlcbch

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
)

const (
	testExpiration = 36
	testPenaltyBPS = 500
)

var (
	testSenderWIF, _    = bchutil.DecodeWIF("cUR6VdPBVn3VQWzJZ9Pr7owhWg3u4Tzoy1w5rstrNKouycpDLUdb")
	testSenderPbk       = gethcmn.FromHex("0x0209476c86262ab494e258f4a0b5eded53a9593458061064525fe804b9d699a6fb")
	testSenderPkh       = bchutil.Hash160(testSenderPbk)
	testSenderAddr, _   = bchutil.NewAddressPubKeyHash(testSenderPkh, &chaincfg.TestNet3Params)
	testRecipientWIF, _ = bchutil.DecodeWIF("cSuHicBzB3NHUMQgNUniGXvXvLwZYArg2AUt4G6NgEqsc1CZ2yRd")
	testRecipientPbk    = gethcmn.FromHex("0x0364bb904687b930a61a2eed3bf90eb230ab71f098148086857d4119b92272f4e3")
	testRecipientPkh    = bchutil.Hash160(testRecipientPbk)
	testSecretKey       = gethcmn.Hash{'1', '2', '3'}.Bytes()
	testSecretHash      = gethcmn.Hash(sha256.Sum256(testSecretKey)).Bytes()
)

func TestP2SHAddr(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		testExpiration,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)

	p2sh, err := c.GetRedeemScriptHash()
	require.NoError(t, err)
	require.Equal(t, "edd54dea3f17975c6ffce95a091320cda6347766", hex.EncodeToString(p2sh))

	addr, err := c.GetP2SHAddress()
	require.NoError(t, err)
	require.Equal(t, "bchtest:prka2n028utewhr0ln545zgnyrx6vdrhvcjjtaa3w0", addr)
}

func TestBuildFullRedeemScript(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		0x1234,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)

	script, err := c.BuildFullRedeemScript()
	require.NoError(t, err)
	require.Equal(t, "02f40102341220ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df27c5579009c63c0009d587aa8537a885579827700a0635679a952798855795779ad670376a91452797e0288ac7e00cd788800cc00c6a26975686d6d6d755167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975685779827700a0635879a954798857795979ad670376a91454797e0288ac7e00cd788800cc5379a26975686d6d6d6d755168",
		hex.EncodeToString(script))
}

func TestBuildReceiveUnlockingScript(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		testExpiration,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)

	recipientSig := []byte{'s', 'i', 'g'}
	script, err := c.BuildReceiveSigScript(recipientSig, testRecipientPbk, testSecretKey)
	require.NoError(t, err)
	require.Equal(t, "203132330000000000000000000000000000000000000000000000000000000000210364bb904687b930a61a2eed3bf90eb230ab71f098148086857d4119b92272f4e303736967004cfb02f401012420ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df27c5579009c63c0009d587aa8537a885579827700a0635679a952798855795779ad670376a91452797e0288ac7e00cd788800cc00c6a26975686d6d6d755167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975685779827700a0635879a954798857795979ad670376a91454797e0288ac7e00cd788800cc5379a26975686d6d6d6d755168",
		hex.EncodeToString(script))
}

func TestBuildRefundUnlockingScript(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		testExpiration,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)

	senderSig := []byte{'s', 'i', 'g'}
	script, err := c.BuildRefundSigScript(senderSig, testSenderPbk)
	require.NoError(t, err)
	require.Equal(t, "210209476c86262ab494e258f4a0b5eded53a9593458061064525fe804b9d699a6fb03736967514cfb02f401012420ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df27c5579009c63c0009d587aa8537a885579827700a0635679a952798855795779ad670376a91452797e0288ac7e00cd788800cc00c6a26975686d6d6d755167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975685779827700a0635879a954798857795979ad670376a91454797e0288ac7e00cd788800cc5379a26975686d6d6d6d755168",
		hex.EncodeToString(script))
}

func TestMakeReceiveTx(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		testExpiration,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)
	tx, err := c.MakeReceiveTx(
		gethcmn.Hash{'u', 't', 'x', 'o'}.Bytes(),
		1,
		100000000,
		testSenderAddr,
		2,
		testSecretKey,
		testRecipientWIF.PrivKey,
	)
	require.NoError(t, err)
	require.Equal(t, uint32(0xffffffff), tx.TxIn[0].Sequence)
	require.Len(t, MsgTxToBytes(tx), 480)
	//require.Equal(t, "?", MsgTxToHex(tx))
}

func TestMakeRefundTx(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		testExpiration,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)
	tx, err := c.MakeRefundTx(
		gethcmn.Hash{'u', 't', 'x', 'o'}.Bytes(),
		1,
		100000000,
		testSenderAddr,
		3,
		testSenderWIF.PrivKey,
	)
	require.NoError(t, err)
	require.Equal(t, uint32(testExpiration), tx.TxIn[0].Sequence)
	require.Len(t, MsgTxToBytes(tx), 447)
	//require.Equal(t, "?", MsgTxToHex(tx))
}

func TestMakeLockTx(t *testing.T) {
	c, err := NewCovenant(
		testSenderPkh,
		testRecipientPkh,
		testSecretHash,
		testExpiration,
		testPenaltyBPS,
		&chaincfg.TestNet3Params,
	)
	require.NoError(t, err)

	inputs := []InputInfo{
		{
			TxID:   gethcmn.Hash{'t', 'x', 'i', 'd'}.Bytes(),
			Vout:   uint32(1),
			Amount: int64(20000),
		},
	}

	outAmt := int64(10000)
	feeRate := uint64(2)
	tx, err := c.MakeLockTx(testSenderWIF.PrivKey, inputs, outAmt, feeRate)
	require.NoError(t, err)
	require.Len(t, MsgTxToBytes(tx), 223)
	//require.Equal(t, "?", MsgTxToHex(tx))
}
