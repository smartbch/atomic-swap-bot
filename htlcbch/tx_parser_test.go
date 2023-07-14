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
	recipientPkh := gethcmn.FromHex("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	senderPkh := gethcmn.FromHex("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	hashLock := gethcmn.FromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	expiration := gethcmn.FromHex("1234")
	penaltyBPS := gethcmn.FromHex("5555")
	sbchAddr := gethcmn.FromHex("ffffffffffffffffffffffffffffffffffffffff")
	pkScript, _ := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).
		AddData([]byte(protoID)).
		AddData(recipientPkh).
		AddData(senderPkh).
		AddData(hashLock).
		AddData(expiration).
		AddData(penaltyBPS).
		AddData(sbchAddr).
		Script()

	c, err := NewTestnet3Covenant(senderPkh, recipientPkh, hashLock, 0x1234, 0x5555)
	require.NoError(t, err)

	pkScript2, err := c.BuildOpRetPkScript(sbchAddr)
	require.NoError(t, err)
	require.Equal(t, pkScript, pkScript2)

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

func TestIsHtlcDepositTx(t *testing.T) {
	txHex := "0200000001fcd86b80ba62f6e28737fd0ffbf360bac9c2ee264b7c266240e0ed2e39439a98020000006441cf3435c8ed23d6e5cb2078594e84e1ec398d398ce00b711dd24848bd3fad5a7052c0a7603cf03fccb2d322ead31b63df1ade87b5cc9dff034a7cde276c2d0e53412102ae6769e5255703c1ce3077b7d4b5b8f53bae40eb3e3be3b9348dc368f67fa6830000000003881300000000000017a914a8afaf6b99a5d5dfd359aa1bc0ef9a0bef0886c88700000000000000006c6a04534241531492a9a3f7f0bbd5b6a66b95db86957de6277bc491148b79ea99e6c418776a9c9d2c5dc074b4404c8a5720ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf30200020201f414621e0b041d19b6472b1e991fe53d78af3c264fa89a9f9800000000001976a9148b79ea99e6c418776a9c9d2c5dc074b4404c8a5788ac00000000"

	tx, err := MsgTxFromBytes(gethcmn.FromHex(txHex))
	require.NoError(t, err)

	//recipientPkh := gethcmn.FromHex("0x104f3f29055f1b2b6debeb6e69a6f0d534f01585")
	result := isHtlcDepositTx(tx)
	require.NotNil(t, result)
	require.Equal(t, "7e6343c8ccdc0ef7504931fb80b61414c1eee4bab287879cbf1f3deb63222b4f", result.TxHash)
	require.Equal(t, "92a9a3f7f0bbd5b6a66b95db86957de6277bc491", hex.EncodeToString(result.RecipientPkh))
	require.Equal(t, "8b79ea99e6c418776a9c9d2c5dc074b4404c8a57", hex.EncodeToString(result.SenderPkh))
	require.Equal(t, "ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3", hex.EncodeToString(result.HashLock))
	require.Equal(t, uint16(2), result.Expiration)
	require.Equal(t, "621e0b041d19b6472b1e991fe53d78af3c264fa8", hex.EncodeToString(result.SenderEvmAddr))
	require.Equal(t, "a8afaf6b99a5d5dfd359aa1bc0ef9a0bef0886c8", hex.EncodeToString(result.ScriptHash))
	require.Equal(t, uint64(5000), result.Value)
}

func TestGetHtlcReceiptInfo(t *testing.T) {
	sigScript := gethcmn.FromHex("203132330000000000000000000000000000000000000000000000000000000000004cd102f401012420ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf31492a9a3f7f0bbd5b6a66b95db86957de6277bc491148b79ea99e6c418776a9c9d2c5dc074b4404c8a575579009c63c0009d567aa8537a880376a9147b7e0288ac7e00cd8800cc00c602d00794a2696d6d5167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975680376a914547a7e0288ac7e00cd8800cc7b02d00794a2696d6d755168")
	receiptInfo := getHtlcReceiptInfo(sigScript)
	require.NotNil(t, receiptInfo)
	require.Equal(t, "3132330000000000000000000000000000000000000000000000000000000000", receiptInfo.Secret)
}

func TestIsReceiptTx(t *testing.T) {
	txHex := "0200000001bc28f853454cae2c597fa0aeb0cdc885df32eb8ac30a07d5c8cb7e90ce4fce4400000000f5203132330000000000000000000000000000000000000000000000000000000000004cd102f401012420ed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf31492a9a3f7f0bbd5b6a66b95db86957de6277bc491148b79ea99e6c418776a9c9d2c5dc074b4404c8a575579009c63c0009d567aa8537a880376a9147b7e0288ac7e00cd8800cc00c602d00794a2696d6d5167557a519dc0009d537ab27500c67600567900a06352795779950210279677527978947b757c0376a91455797e0288ac7e51cd788851cc5279a26975680376a914547a7e0288ac7e00cd8800cc7b02d00794a2696d6d755168feffffff01a00f0000000000001976a91492a9a3f7f0bbd5b6a66b95db86957de6277bc49188ac60630200"

	tx, err := MsgTxFromBytes(gethcmn.FromHex(txHex))
	require.NoError(t, err)

	result := isHtlcReceiptTx(tx)
	require.NotNil(t, result)
	require.Equal(t, "44ce4fce907ecbc8d5070ac38aeb32df85c8cdb0aea07f592cae4c4553f828bc", result.PrevTxHash)
	require.Equal(t, "c748992bb1d40087c6976099e70c4fbf7124ab17359e5337baeb8e96589db15f", result.TxHash)
	require.Equal(t, "3132330000000000000000000000000000000000000000000000000000000000", result.Secret)
}
