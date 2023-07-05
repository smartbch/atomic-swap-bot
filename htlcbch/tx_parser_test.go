package htlcbch

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/txscript"
)

func TestIsP2SH(t *testing.T) {
	require.Nil(t, getP2SHash(gethcmn.FromHex("a914748284390f9e263a4b766a75d0633c50426eb87587ff"))) // wrong length
	require.Nil(t, getP2SHash(gethcmn.FromHex("a814748284390f9e263a4b766a75d0633c50426eb87587")))   // invalid head
	require.Nil(t, getP2SHash(gethcmn.FromHex("a915748284390f9e263a4b766a75d0633c50426eb87587")))   // invalid push
	require.Nil(t, getP2SHash(gethcmn.FromHex("a914748284390f9e263a4b766a75d0633c50426eb87588")))   // invalid tail
	require.Equal(t, gethcmn.FromHex("748284390f9e263a4b766a75d0633c50426eb875"),
		getP2SHash(gethcmn.FromHex("a914748284390f9e263a4b766a75d0633c50426eb87587")))
}

func TestGetHtlcDepositInfo(t *testing.T) {
	pkScript, _ := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte(protoID)).
		AddData(gethcmn.FromHex("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")).
		AddData(gethcmn.FromHex("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")).
		AddData(gethcmn.FromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")).
		AddData(gethcmn.FromHex("1234")).
		AddData(gethcmn.FromHex("5555")).
		AddData(gethcmn.FromHex("ffffffffffffffffffffffffffffffffffffffff")).Script()
	//fmt.Println(hex.EncodeToString(s))
	depositInfo := getHtlcDepositInfo(pkScript)
	require.NotNil(t, depositInfo)
	require.Equal(t, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", hex.EncodeToString(depositInfo.RecipientPkh))
	require.Equal(t, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", hex.EncodeToString(depositInfo.SenderPkh))
	require.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", hex.EncodeToString(depositInfo.HashLock))
	require.Equal(t, uint16(0x1234), depositInfo.Expiration)
	require.Equal(t, uint16(0x5555), depositInfo.PenaltyBPS)
	require.Equal(t, "ffffffffffffffffffffffffffffffffffffffff", hex.EncodeToString(depositInfo.SenderEvmAddr))
}

func TestGetHtlcDepositInfo2(t *testing.T) {
	pkScript := "0x6a0453424153144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df27c20ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf302002402050014765fd1f0e3d125b36de29b5f88295a247814276e"
	depositInfo := getHtlcDepositInfo(gethcmn.FromHex(pkScript))
	require.NotNil(t, depositInfo)
	require.Equal(t, "4d027fdd0585302264922bed58b8a84d38776ccb", hex.EncodeToString(depositInfo.RecipientPkh))
	require.Equal(t, "a47165ef477c99a53cdeb846a7687a069d7df27c", hex.EncodeToString(depositInfo.SenderPkh))
	require.Equal(t, "ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3", hex.EncodeToString(depositInfo.HashLock))
	require.Equal(t, uint16(0x0024), depositInfo.Expiration)
	require.Equal(t, uint16(0x0500), depositInfo.PenaltyBPS)
	require.Equal(t, "765fd1f0e3d125b36de29b5f88295a247814276e", hex.EncodeToString(depositInfo.SenderEvmAddr))
}

func TestGetHtlcReceiptInfo(t *testing.T) {
	sigScript := gethcmn.FromHex("207365637265740000000000000000000000000000000000000000000000000000210364bb904687b930a61a2eed3bf90eb230ab71f098148086857d4119b92272f4e303736967004cfb02f401012420497a39b618484855ebb5a2cabf6ee52ff092e7c17f8bfe79313529f9774f83a2144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df2ab5579009c63c0009d587aa8537a885579827700a0635679a952798855795779ad670376a91452797e0288ac7e00cd788800cc00c6a26975686d6d6d755167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975685779827700a0635879a954798857795979ad670376a91454797e0288ac7e00cd788800cc5379a26975686d6d6d6d755168")
	receiptInfo := getHtlcReceiptInfo(sigScript)
	require.NotNil(t, receiptInfo)
	require.Equal(t, "7365637265740000000000000000000000000000000000000000000000000000", receiptInfo.Secret)
}

/*
func TestIsHtlcDepositTx(t *testing.T) {
	txHex := "0200000001f96d09453d59c691d5d3f50e80d797c88785534d5396a05bdec77a9d9bbd51b80000000064410b6856319797109f8d69bc4defe2074387bc3239fa7b2f21f4308ab2f132cc902f5ac63a7825134211e94f002d2fcc92efcd50ff38e020e213eccb6ce137b6cd412103bfca3bfe0d213cad8c7e88573fdae7948381582af35c7af8e51c8d1c9e7a8d68000000000380841e000000000017a9143c7cd004400d5e1609ddb062cd327c39e44bd4718700000000000000006c6a045342415314104f3f29055f1b2b6debeb6e69a6f0d534f01585142976d1e8430b664cb903e05187a4277d111a3b5620511806571595259ff92bc5bdd13530b8cda3aec235ac6f31aa3e29574cb5455e0200060201f41412b8b81ffcf5af1ce29f0d5bb9a6347601d17b786e437a00000000001976a9142976d1e8430b664cb903e05187a4277d111a3b5688ac00000000"

	tx, err := MsgTxFromBytes(gethcmn.FromHex(txHex))
	require.NoError(t, err)

	recipientPkh := gethcmn.FromHex("0x104f3f29055f1b2b6debeb6e69a6f0d534f01585")
	result := isHtlcDepositTx(tx, recipientPkh)
	require.NotNil(t, result)
	require.Equal(t, "a4eb7578a853e99acc9d721a204f47dbf15f25a0768eb50f243c1a8337edb26b", result.TxHash)
	require.Equal(t, "104f3f29055f1b2b6debeb6e69a6f0d534f01585", hex.EncodeToString(result.RecipientPkh))
	require.Equal(t, "2976d1e8430b664cb903e05187a4277d111a3b56", hex.EncodeToString(result.SenderPkh))
	require.Equal(t, "511806571595259ff92bc5bdd13530b8cda3aec235ac6f31aa3e29574cb5455e", hex.EncodeToString(result.HashLock))
	require.Equal(t, uint16(6), result.Expiration)
	require.Equal(t, "12b8b81ffcf5af1ce29f0d5bb9a6347601d17b78", hex.EncodeToString(result.SenderEvmAddr))
	require.Equal(t, "3c7cd004400d5e1609ddb062cd327c39e44bd471", hex.EncodeToString(result.ScriptHash))
	require.Equal(t, uint64(2000000), result.Value)
}
*/
