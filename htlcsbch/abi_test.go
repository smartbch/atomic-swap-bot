package htlcsbch

import (
	"encoding/hex"
	"math/big"
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

func TestUnpackGetSwapState(t *testing.T) {
	data := common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000003")
	n, err := UnpackGetSwapState(data)
	require.NoError(t, err)
	require.Equal(t, uint8(3), n)
}

func TestPackGetMarketMaker(t *testing.T) {
	addr := common.Address{'b', 'o', 't', '1'}
	data, err := PackGetMarketMaker(addr)
	require.NoError(t, err)
	require.Equal(t, strings.ReplaceAll(`f60559eb
000000000000000000000000626f743100000000000000000000000000000000
`, "\n", ""), hex.EncodeToString(data))
}

func TestUnpackGetMarketMaker(t *testing.T) {
	data := common.FromHex("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c80000000000000000000000000000000000000000000000000000000000000000626f7431000000000000000000000000000000000000000000000000000000004d027fdd0585302264922bed58b8a84d38776ccb0000000000000000000000000000000000000000000000000000000000000000000000000000000000000048000000000000000000000000000000000000000000000000000000000000a8c000000000000000000000000000000000000000000000000000000000000001f40000000000000000000000000000000000000000000000000000000000000064000000000000000000000000000000000000000000000000016345785d8a00000000000000000000000000000000000000000000000000000de0b6b3a764000000000000000000000000000000000000000000000000000000000000499602d30000000000000000000000009965507d1a55bcc2695c58ba16fb37d819b0a4dc0000000000000000000000000000000000000000000000000000000000000000")
	mm, err := UnpackGetMarketMaker(data)
	require.NoError(t, err)
	require.Equal(t, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", mm.Addr.String())
	require.Equal(t, uint64(0x0), mm.RetiredAt)
	require.Equal(t, "626f743100000000000000000000000000000000000000000000000000000000", hex.EncodeToString(mm.Intro[:]))
	require.Equal(t, "4d027fdd0585302264922bed58b8a84d38776ccb", hex.EncodeToString(mm.BchPkh[:]))
	require.Equal(t, uint16(72), mm.BchLockTime)
	require.Equal(t, uint32(43200), mm.SbchLockTime)
	require.Equal(t, uint16(500), mm.PenaltyBPS)
	require.Equal(t, uint16(100), mm.FeeBPS)
	require.Equal(t, big.NewInt(100000000000000000), mm.MinSwapAmt)
	require.Equal(t, big.NewInt(1000000000000000000), mm.MaxSwapAmt)
	require.Equal(t, big.NewInt(1234567891), mm.StakedValue)
	require.Equal(t, "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc", mm.Checker.String())
	require.Equal(t, false, mm.Unavailable)
}
