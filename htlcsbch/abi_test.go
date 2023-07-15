package htlcsbch

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestABI(t *testing.T) {
	require.Equal(t, "0x7c669c5d09f55af8b2e3b6e432f8bd140dd3a4811451b4864833bcee54f7df67",
		OpenEventId.String())
	require.Equal(t, "0x842eb23b01edb198a935f6cf1ead8ec295651395574206ce5787d42293c5b430",
		CloseEventId.String())
	require.Equal(t, "0xbddd9b693ea862fad6ecf78fd51c065be26fda94d1f3cad3a7d691453a38a735",
		ExpireEventId.String())
	require.Equal(t, "f4fa2653", hex.EncodeToString(htlcAbi.Methods["open"].ID))
	require.Equal(t, "f10ca95a", hex.EncodeToString(htlcAbi.Methods["close"].ID))
	require.Equal(t, "c6441798", hex.EncodeToString(htlcAbi.Methods["expire"].ID))
}

func TestPackOpen(t *testing.T) {
	recipient := common.Address{'b', 'o', 't', 0xbb}
	hashLock := common.Hash{'s', 'e', 'c', 'r', 'e', 't', 0xcc}
	timeLock := uint32(0x12345)
	bchAddr := common.Address{'u', 's', 'e', 'r', 0xdd}

	data, err := PackOpen(recipient, hashLock, timeLock, bchAddr)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(`f4fa2653
000000000000000000000000626f74bb00000000000000000000000000000000
736563726574cc00000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000012345
75736572dd000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000000
`, "\n", ""), hex.EncodeToString(data))
}

func TestPackClose(t *testing.T) {
	hashLock := common.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k', 0xaa}
	secret := common.Hash{'s', 'e', 'c', 'r', 'e', 't', 0xbb}
	data, err := PackClose(hashLock, secret)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(`f10ca95a
686173686c6f636baa0000000000000000000000000000000000000000000000
736563726574bb00000000000000000000000000000000000000000000000000
`, "\n", ""), hex.EncodeToString(data))
}

func TestPackExpire(t *testing.T) {
	hashLock := common.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k', 0xaa}
	data, err := PackExpire(hashLock)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(`c6441798
686173686c6f636baa0000000000000000000000000000000000000000000000
`, "\n", ""), hex.EncodeToString(data))
}

func TestPackGetSwapState(t *testing.T) {
	hashLock := common.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k', 0xaa}
	data, err := PackGetSwapState(hashLock)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(`db9b6d06
686173686c6f636baa0000000000000000000000000000000000000000000000
`, "\n", ""), hex.EncodeToString(data))
}
