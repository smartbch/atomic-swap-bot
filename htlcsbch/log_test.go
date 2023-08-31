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
    "0x910da5a424b2c73336aaacc2dfb830def3c3a743a913c0c8f40b65412c867037",
    "0x000000000000000000000000f29c9ef6496a482b94bdb45aba93d661f082922c",
    "0x00000000000000000000000060d8666337c854686f2cf8a49b777c223b72fe34"
  ],
  "data": "0x064fc464aa4a83786e72727b4be4790176600ab1c29f63be8a4333d13bc4da3300000000000000000000000000000000000000000000000000000000644261f400000000000000000000000000000000000000000000000000005af3107a40006a47edfbcbeeb8e61d20351dd3b0305454395ab500000000000000000000000000000000000000000000000000000000000000000000000000000000644261a400000000000000000000000000000000000000000000000000000000000003450000000000000000000000000000000000000000000000000de0b6b3a7640000",
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

	lockLog := ParseHtlcLockLog(log)
	require.NotNil(t, lockLog)
	require.Equal(t, "0xf29c9eF6496A482b94BDB45ABA93d661F082922C",
		lockLog.LockerAddr.String())
	require.Equal(t, "0x60d8666337C854686F2CF8A49B777c223b72fe34",
		lockLog.UnlockerAddr.String())
	require.Equal(t, "0x064fc464aa4a83786e72727b4be4790176600ab1c29f63be8a4333d13bc4da33",
		lockLog.HashLock.String())
	require.Equal(t, uint64(1682072052), lockLog.UnlockTime)
	require.Equal(t, uint64(100000000000000), lockLog.Value.Uint64())
	require.Equal(t, "0x6a47EDfbcBEeB8e61d20351dD3b0305454395AB5",
		lockLog.BchRecipientPkh.String())
	require.Equal(t, uint64(1682071972), lockLog.CreatedTime)
	require.Equal(t, uint16(0x345), lockLog.PenaltyBPS)
}

func TestParseHtlcUnlockLog(t *testing.T) {
	logJson := `{
  "address": "0xc03a886b25cabc20db49170981ef118693e807d9",
  "topics": [
	  "0x3175e1e0b41583586838d3f2db12a22ab1b97413989a1e14f52bc748396ee957",
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

	unlockLog := ParseHtlcUnlockLog(log)
	require.NotNil(t, unlockLog)
	require.Equal(t, "0x576788bdc7c221a4cf6c2670f7aa54062599f45a1806afe60e46d34a5cee8ae8",
		unlockLog.TxHash.String())
	require.Equal(t, "0x3bd34fe3485138a7be6f1be4a1d3c23661090d2c95af969c5c73fee04089ab06",
		unlockLog.HashLock.String())
	require.Equal(t, "0x3163666434353566623035326435363964633361363337636263373065390000",
		unlockLog.Secret.String())
}

func TestParseRefundLog(t *testing.T) {
	logJson := `{
  "address": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
  "topics": [
    "0x3fbd469ec3a5ce074f975f76ce27e727ba21c99176917b97ae2e713695582a12",
    "0xed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3"
  ],
  "data": "0x",
  "blockNumber": "0x4",
  "transactionHash": "0xda0ae40abf70d204a1bdcc012ea97dd06f85842c9b36e08d66c16a23c5aab027",
  "transactionIndex": "0x0",
  "blockHash": "0x80f2a0785c21778ddc28896448756b93d38c751a2b360ddd9e660019d7411304",
  "logIndex": "0x0",
  "removed": false
}`

	var log types.Log
	err := json.Unmarshal([]byte(logJson), &log)
	require.NoError(t, err)

	refundLog := ParseHtlcRefundLog(log)
	require.NotNil(t, refundLog)
	require.Equal(t, "0xda0ae40abf70d204a1bdcc012ea97dd06f85842c9b36e08d66c16a23c5aab027",
		refundLog.TxHash.String())
	require.Equal(t, "0xed88bb4d5991f2f91939d37277c0f988bbf461c889cafbdd5384ecb881ce6bf3",
		refundLog.HashLock.String())
}
