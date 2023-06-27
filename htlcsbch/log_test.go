package htlcsbch

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestParseHtlcOpenLog(t *testing.T) {
	logJson := `{
  "address" :"0xa98881b7c5f31d277c09bdfac2096436538bb31c",
  "topics" :[
    "0x7c669c5d09f55af8b2e3b6e432f8bd140dd3a4811451b4864833bcee54f7df67",
    "0x000000000000000000000000f29c9ef6496a482b94bdb45aba93d661f082922c",
    "0x00000000000000000000000060d8666337c854686f2cf8a49b777c223b72fe34"
  ],
  "data": "0x064fc464aa4a83786e72727b4be4790176600ab1c29f63be8a4333d13bc4da3300000000000000000000000000000000000000000000000000000000644261f400000000000000000000000000000000000000000000000000005af3107a40006a47edfbcbeeb8e61d20351dd3b0305454395ab500000000000000000000000000000000000000000000000000000000000000000000000000000000644261a40000000000000000000000000000000000000000000000000000000000000345",
  "blockNumber": "0x9328a6",
  "transactionHash": "0x1147850cdd823782a205f800695c11d7dfd0a93da07e3efae78b576cf9b22fce",
  "transactionIndex": "0x0",
  "blockHash": "0xd2bc148d47f783da842057933c6b0e4c410077f414d037ff3e3a72336aaa8eb6",
  "logIndex": "0x0",
  "removed": false
} `

	var log types.Log
	err := json.Unmarshal([]byte(logJson), &log)
	require.NoError(t, err)

	openLog := ParseHtlcOpenLog(log)
	require.NotNil(t, openLog)
	require.Equal(t, "0xf29c9eF6496A482b94BDB45ABA93d661F082922C",
		openLog.LockerAddr.String())
	require.Equal(t, "0x60d8666337C854686F2CF8A49B777c223b72fe34",
		openLog.UnlockerAddr.String())
	require.Equal(t, "0x064fc464aa4a83786e72727b4be4790176600ab1c29f63be8a4333d13bc4da33",
		openLog.HashLock.String())
	require.Equal(t, uint64(1682072052), openLog.UnlockTime)
	require.Equal(t, uint64(100000000000000), openLog.Value.Uint64())
	require.Equal(t, "0x6a47EDfbcBEeB8e61d20351dD3b0305454395AB5",
		openLog.BchRecipientPkh.String())
	require.Equal(t, uint64(1682071972), openLog.CreatedTime)
	require.Equal(t, uint16(0x345), openLog.PenaltyBPS)
}

func TestParseHtlcCloseLog(t *testing.T) {
	logJson := `{
  "address": "0xc03a886b25cabc20db49170981ef118693e807d9",
  "topics": [
	"0x842eb23b01edb198a935f6cf1ead8ec295651395574206ce5787d42293c5b430",
	"0x3bd34fe3485138a7be6f1be4a1d3c23661090d2c95af969c5c73fee04089ab06",
	"0x3163666434353566623035326435363964633361363337636263373065390000"
  ],
  "data": "0x",
  "blockNumber": "0x966838",
  "transactionHash": "0x576788bdc7c221a4cf6c2670f7aa54062599f45a1806afe60e46d34a5cee8ae8",
  "transactionIndex": "0x0",
  "blockHash": "0x3ce5eacccbe3b67c88690854fc209b9af0810eec34a867746be5e3dd4df13f65",
  "logIndex": "0x0",
  "removed": false
}`

	var log types.Log
	err := json.Unmarshal([]byte(logJson), &log)
	require.NoError(t, err)

	closeLog := ParseHtlcCloseLog(log)
	require.NotNil(t, closeLog)
	require.Equal(t, "0x3bd34fe3485138a7be6f1be4a1d3c23661090d2c95af969c5c73fee04089ab06",
		closeLog.HashLock.String())
	require.Equal(t, "0x3163666434353566623035326435363964633361363337636263373065390000",
		closeLog.Secret.String())
}
