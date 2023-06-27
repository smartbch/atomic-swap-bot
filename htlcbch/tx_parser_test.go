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
	sigScript := gethcmn.FromHex("207365637265740000000000000000000000000000000000000000000000000000210364bb904687b930a61a2eed3bf90eb230ab71f098148086857d4119b92272f4e303736967004cde02f401012420497a39b618484855ebb5a2cabf6ee52ff092e7c17f8bfe79313529f9774f83a2144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df27c5579009c635679827700a0635779a952798856795879ad670376a91452797e0288ac7e51c778887568c0009d587aa8537a886d6d6d755167557a519d5579827700a0635679a9788855795779ad670376a914787e0288ac7e51c778887568c0009d537ab275537900a06300c65479950210279600cc78a2690376a91453797e0288ac7e00cd78886d686d6d6d5168")
	receiptInfo := getHtlcReceiptInfo(sigScript)
	require.NotNil(t, receiptInfo)
	require.Equal(t, "7365637265740000000000000000000000000000000000000000000000000000", receiptInfo.Secret)
}

/*
func TestIsHtlcDepositTx(t *testing.T) {
	txHex := "02000000014063938cc2418fb0cbd855a3743f8807fe19e21ae724562aecf8f009a4d8905b000000006b48304502210081ac9e44c11d2ebb87e9cce90f85eaffbf257721867f957f54e9022f445d513802206efdf6a1b7f83e84a1e4f840427a0de932102bcbce3d294bcf0bb9b88806d77f41210209476c86262ab494e258f4a0b5eded53a9593458061064525fe804b9d699a6fbffffffff03881300000000000017a914ea613599b57cd5604b5e2e29f669cae3ebc212348700000000000000006b6a0453424153144d027fdd0585302264922bed58b8a84d38776ccb14a47165ef477c99a53cdeb846a7687a069d7df27c20ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3040000002414765fd1f0e3d125b36de29b5f88295a247814276e18730100000000001976a914a47165ef477c99a53cdeb846a7687a069d7df27c88ac00000000"

	tx, err := msgTxFromBytes(gethcmn.FromHex(txHex))
	require.NoError(t, err)

	recipientPkh := gethcmn.FromHex("0x4d027fdd0585302264922bed58b8a84d38776ccb")
	result := isHtlcDepositTx(tx, recipientPkh)
	require.NotNil(t, result)
	require.Equal(t, "191ab837849dbba53236f4344d30fd8567b43b04ed12007ca869b007a1f98630", result.TxHash)
	require.Equal(t, "4d027fdd0585302264922bed58b8a84d38776ccb", hex.EncodeToString(result.RecipientPkh))
	require.Equal(t, "a47165ef477c99a53cdeb846a7687a069d7df27c", hex.EncodeToString(result.SenderPkh))
	require.Equal(t, "ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3", hex.EncodeToString(result.HashLock))
	require.Equal(t, uint32(36), result.Expiration)
	require.Equal(t, "765fd1f0e3d125b36de29b5f88295a247814276e", hex.EncodeToString(result.SenderEvmAddr))
	require.Equal(t, "ea613599b57cd5604b5e2e29f669cae3ebc21234", hex.EncodeToString(result.ScriptHash))
	require.Equal(t, uint64(5000), result.Value)
}
*/

//func msgTxFromBytes(data []byte) (*wire.MsgTx, error) {
//	msg := &wire.MsgTx{}
//	err := msg.Deserialize(bytes.NewReader(data))
//	return msg, err
//}
