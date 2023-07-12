package bot

import (
	"crypto/sha256"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"

	"github.com/smartbch/atomic-swap-bot/htlcbch"
	"github.com/smartbch/atomic-swap-bot/htlcsbch"
)

var (
	testBchPrivKeyHex = "771a1a3d28e7c001bc85906ec0c592133f33f552bf464005d2f50fb558442f91"
	testBchPrivKey, _ = bchec.PrivKeyFromBytes(bchec.S256(), gethcmn.FromHex(testBchPrivKeyHex))
	testBchPubKey     = testBchPrivKey.PubKey().SerializeCompressed()
	testBchPkh        = bchutil.Hash160(testBchPubKey)
	testBchAddr, _    = bchutil.NewAddressPubKeyHash(testBchPkh, &chaincfg.MainNetParams)
	testEvmAddr       = gethcmn.Address{'b', 'o', 't'}
)

func TestUtxoAmtToSats(t *testing.T) {
	amt, err := strconv.ParseFloat("0.000520", 64)
	require.NoError(t, err)
	require.Equal(t, int64(52000), utxoAmtToSats(amt))
}

func TestBch2Sbch_userLockBch(t *testing.T) {
	_botPkh := testBchPkh
	_userPkh := gethAddrBytes("user")
	_hashLock := gethHash32Bytes("hash")
	_hashLock2 := gethHash32Bytes("hash2")
	_timeLock := uint16(100)
	_penaltyBPS := uint16(500)
	_evmAddr := gethAddrBytes("evm")

	covenant, err := htlcbch.NewMainnetCovenant(_userPkh, _botPkh, _hashLock, _timeLock, _penaltyBPS)
	require.NoError(t, err)
	scriptHash, err := covenant.GetRedeemScriptHash()
	require.NoError(t, err)

	covenant2, err := htlcbch.NewMainnetCovenant(_userPkh, _userPkh, _hashLock2, _timeLock, _penaltyBPS)
	require.NoError(t, err)
	scriptHash2, err := covenant2.GetRedeemScriptHash()
	require.NoError(t, err)

	_db := initDB(t, 123, 456)
	_bchCli := newMockBchClient(124, 128)
	_bchCli.blocks[126] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{
					{
						Value:    12345678,
						PkScript: newP2SHPkScript(scriptHash),
					},
					{
						PkScript: newHtlcDepositOpRet(_botPkh, _userPkh, _hashLock, _timeLock, _penaltyBPS, _evmAddr),
					},
				},
			},
		},
	}
	_bchCli.blocks[127] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{
					{
						Value:    12345678,
						PkScript: newP2SHPkScript(scriptHash2),
					},
					{
						PkScript: newHtlcDepositOpRet(_userPkh, _userPkh, _hashLock2, _timeLock, _penaltyBPS, _evmAddr),
					},
				},
			},
		},
	}

	_bot := &MarketMakerBot{
		db:           _db,
		bchCli:       _bchCli,
		bchPrivKey:   testBchPrivKey,
		bchPkh:       _botPkh,
		bchTimeLock:  _timeLock,
		penaltyRatio: _penaltyBPS,
	}
	_bot.scanBchBlocks()

	newH, err := _db.getLastBchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(128), newH)

	records, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, uint64(126), record0.BchLockHeight)
	require.Equal(t, _bchCli.blocks[126].Transactions[0].TxHash().String(), record0.BchLockTxHash)
	require.Equal(t, uint64(_bchCli.blocks[126].Transactions[0].TxOut[0].Value), record0.Value)
	require.Equal(t, toHex(_botPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, uint32(_timeLock), record0.TimeLock)
	require.Equal(t, _penaltyBPS, record0.PenaltyBPS)
	require.Equal(t, toHex(_evmAddr), record0.SenderEvmAddr)
	require.Equal(t, toHex(scriptHash), record0.HtlcScriptHash)
	require.Equal(t, "", record0.SbchLockTxHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusNew, record0.Status)
}

func TestBch2Sbch_userLockBch_invalidParams(t *testing.T) {
	_userPkh := gethAddrBytes("user")
	_hashLock := gethHash32Bytes("hash")
	_timeLock := uint16(100)
	_penaltyBPS := uint16(500)
	_minSwapVal := uint64(100000)
	_maxSwapVal := uint64(999999)
	_evmAddr := gethAddrBytes("evm")

	_db := initDB(t, 123, 456)
	_bchCli := newMockBchClient(124, 128)
	_bchCli.blocks[126] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{ // invalid timeLock
					{
						Value:    int64(_maxSwapVal - 1),
						PkScript: getHtlcP2shPkScript(_userPkh, testBchPkh, _hashLock, _timeLock/2, _penaltyBPS),
					},
					{
						PkScript: newHtlcDepositOpRet(testBchPkh, _userPkh, _hashLock, _timeLock/2, _penaltyBPS, _evmAddr),
					},
				},
			},
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{ // invalid penaltyBPS
					{
						Value:    int64(_maxSwapVal - 1),
						PkScript: getHtlcP2shPkScript(_userPkh, testBchPkh, _hashLock, _timeLock, _penaltyBPS+1),
					},
					{
						PkScript: newHtlcDepositOpRet(testBchPkh, _userPkh, _hashLock, _timeLock, _penaltyBPS+1, _evmAddr),
					},
				},
			},
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{ // value too large
					{
						Value:    int64(_maxSwapVal + 1),
						PkScript: getHtlcP2shPkScript(_userPkh, testBchPkh, _hashLock, _timeLock, _penaltyBPS),
					},
					{
						PkScript: newHtlcDepositOpRet(testBchPkh, _userPkh, _hashLock, _timeLock, _penaltyBPS, _evmAddr),
					},
				},
			},
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{ // value too small
					{
						Value:    int64(_minSwapVal - 1),
						PkScript: getHtlcP2shPkScript(_userPkh, testBchPkh, _hashLock, _timeLock, _penaltyBPS),
					},
					{
						PkScript: newHtlcDepositOpRet(testBchPkh, _userPkh, _hashLock, _timeLock, _penaltyBPS, _evmAddr),
					},
				},
			},
		},
	}

	_bot := &MarketMakerBot{
		db:           _db,
		bchCli:       _bchCli,
		bchPrivKey:   testBchPrivKey,
		bchPkh:       testBchPkh,
		bchTimeLock:  _timeLock,
		penaltyRatio: _penaltyBPS,
		maxSwapVal:   _maxSwapVal,
		minSwapVal:   _minSwapVal,
	}
	_bot.scanBchBlocks()

	newH, err := _db.getLastBchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(128), newH)

	records, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, records, 0)
}

func TestBch2Sbch_botLockSbch(t *testing.T) {
	_val := uint64(12345678)
	_txHash := gethHash32Bytes("bchlock")
	_botPkh := gethAddrBytes("bot")
	_userPkh := gethAddrBytes("user")
	_hashLock := gethHash32Bytes("hash")
	_timeLock := uint32(100)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
		Status:         Bch2SbchStatusNew,
	}))

	_bchCli := newMockBchClient(124, 125)
	_sbchCli := newMockSbchClient(457, 999, 0)
	_bot := &MarketMakerBot{
		db:              _db,
		bchCli:          _bchCli,
		sbchCli:         _sbchCli,
		bchPrivKey:      testBchPrivKey,
		bchPkh:          _botPkh,
		bchTimeLock:     72,
		serviceFeeRatio: 100,
	}
	_bot.handleBchUserDeposits()

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 0)

	bchLocked, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 100)
	require.NoError(t, err)
	require.Len(t, bchLocked, 1)

	record0 := bchLocked[0]
	require.Equal(t, toHex(_txHash), record0.BchLockTxHash)
	require.Equal(t, _val, record0.Value)
	require.Equal(t, toHex(_botPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_evmAddr), record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusSbchLocked, record0.Status)
}

func TestBch2Sbch_botLockSbch_notConfirmed(t *testing.T) {
	_val := uint64(12345678)
	_txHash := gethHash32Bytes("bchlock")
	_botPkh := gethAddrBytes("bot")
	_userPkh := gethAddrBytes("user")
	_hashLock := gethHash32Bytes("hash")
	_timeLock := uint32(72)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
	}))

	_bchCli := newMockBchClient(124, 130)
	_bot := &MarketMakerBot{
		db:               _db,
		bchCli:           _bchCli,
		bchPrivKey:       testBchPrivKey,
		bchPkh:           _botPkh,
		bchTimeLock:      72,
		bchConfirmations: 10,
	}
	_bot.scanBchBlocks()

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 1)
}

func TestBch2Sbch_botLockSbch_tooLate(t *testing.T) {
	_val := uint64(12345678)
	_txHash := gethHash32Bytes("bchlock")
	_botPkh := gethAddrBytes("bot")
	_userPkh := gethAddrBytes("user")
	_hashLock := gethHash32Bytes("hash")
	_timeLock := uint32(72)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
	}))

	_bchCli := newMockBchClient(134, 160)
	_bchCli.confirmations[toHex(_txHash)] = 100
	_bot := &MarketMakerBot{
		db:          _db,
		bchCli:      _bchCli,
		bchPrivKey:  testBchPrivKey,
		bchPkh:      _botPkh,
		bchTimeLock: 72,
	}
	_bot.handleBchUserDeposits()

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 0)

	tooLate, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusTooLateToLockSbch, 100)
	require.NoError(t, err)
	require.Len(t, tooLate, 1)
}

func TestBch2Sbch_userUnlockSbch(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethHash32("secret")
	_bchLockTxHash := gethHash32("bchlock")
	_userPkh := gethAddrBytes("user")
	_hashLock := sha256.Sum256(_secret[:])
	_timeLock := uint32(100)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")
	_sbchLockTxHash := gethHash32Bytes("sbchlock")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  122,
		BchLockTxHash:  toHex(_bchLockTxHash.Bytes()),
		Value:          _val,
		RecipientPkh:   toHex(testBchPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock[:]),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
		SbchLockTxHash: toHex(_sbchLockTxHash),
		Status:         Bch2SbchStatusSbchLocked,
	}))

	_sbchCli := newMockSbchClient(457, 999, 0)
	_sbchCli.logs[458] = []gethtypes.Log{
		{
			Topics: []gethcmn.Hash{
				htlcsbch.CloseEventId,
				_hashLock,
				_secret,
			},
		},
	}

	_bot := &MarketMakerBot{
		db:      _db,
		sbchCli: _sbchCli,
		bchPkh:  testBchPkh,
	}

	_bot.scanSbchEvents()

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 0)

	bchLocked, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 100)
	require.NoError(t, err)
	require.Len(t, bchLocked, 0)

	secretRevealed, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSecretRevealed, 100)
	require.NoError(t, err)
	require.Len(t, secretRevealed, 1)
	record0 := secretRevealed[0]
	require.Equal(t, toHex(_bchLockTxHash.Bytes()), record0.BchLockTxHash)
	require.Equal(t, _val, record0.Value)
	require.Equal(t, toHex(testBchPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock[:]), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_evmAddr), record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, toHex(_sbchLockTxHash), record0.SbchLockTxHash)
	require.Equal(t, toHex(_secret[:]), record0.Secret)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusSecretRevealed, record0.Status)
}

func TestBch2Sbch_botUnlockBch(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethHash32Bytes("secret")
	_bchLockTxHash := gethHash32Bytes("bchlock")
	_userPkh := gethAddrBytes("user")
	_hashLock := sha256.Sum256(_secret)
	_timeLock := uint32(100)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")
	_sbchLockTxHash := gethHash32Bytes("sbchlock")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  122,
		BchLockTxHash:  toHex(_bchLockTxHash),
		Value:          _val,
		RecipientPkh:   toHex(testBchPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock[:]),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
		SbchLockTxHash: toHex(_sbchLockTxHash),
		Secret:         toHex(_secret),
		Status:         Bch2SbchStatusSecretRevealed,
	}))

	_bot := &MarketMakerBot{
		db:         _db,
		bchCli:     &MockBchClient{},
		bchPrivKey: testBchPrivKey,
		bchPkh:     testBchPkh,
		bchAddr:    testBchAddr,
	}
	_bot.unlockBchUserDeposits()

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 0)

	bchLocked, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 100)
	require.NoError(t, err)
	require.Len(t, bchLocked, 0)

	secretRevealed, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSecretRevealed, 100)
	require.NoError(t, err)
	require.Len(t, secretRevealed, 0)

	bchUnlocked, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusBchUnlocked, 100)
	require.NoError(t, err)
	require.Len(t, bchUnlocked, 1)
	record0 := bchUnlocked[0]
	require.Equal(t, toHex(_bchLockTxHash), record0.BchLockTxHash)
	require.Equal(t, _val, record0.Value)
	require.Equal(t, toHex(testBchPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock[:]), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_evmAddr), record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, toHex(_sbchLockTxHash), record0.SbchLockTxHash)
	require.Equal(t, toHex(_secret), record0.Secret)
	require.Equal(t, "5447476b1799d9744a78a641ab1e313615f96fe06fcee59ce3b56f36d30630f4",
		record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusBchUnlocked, record0.Status)
}

func TestBch2Sbch_botRefundSbch(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethHash32("secret")
	_bchLockTxHash := gethHash32("bchlock")
	_userPkh := gethAddrBytes("user")
	_hashLock := sha256.Sum256(_secret[:])
	_timeLock := uint32(72)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")
	_sbchLockTxHash := gethHash32Bytes("sbchlock")
	_sbchNow := uint64(time.Now().Unix())
	_sbchLockTxTime := _sbchNow - 22000

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  122,
		BchLockTxHash:  toHex(_bchLockTxHash.Bytes()),
		Value:          _val,
		RecipientPkh:   toHex(testBchPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock[:]),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
		SbchLockTxTime: _sbchLockTxTime,
		SbchLockTxHash: toHex(_sbchLockTxHash),
		Status:         Bch2SbchStatusSbchLocked,
	}))

	_sbchCli := newMockSbchClient(457, 999, _sbchNow)
	_sbchCli.txTimes[toHex(_sbchLockTxHash)] = _sbchLockTxTime
	_bot := &MarketMakerBot{
		db:      _db,
		sbchCli: _sbchCli,
		bchPkh:  testBchPkh,
	}

	_bot.refundLockedSbch()

	secretRevealed, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchRefunded, 100)
	require.NoError(t, err)
	require.Len(t, secretRevealed, 1)
	record0 := secretRevealed[0]
	require.Equal(t, toHex(_bchLockTxHash.Bytes()), record0.BchLockTxHash)
	require.Equal(t, _val, record0.Value)
	require.Equal(t, toHex(testBchPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock[:]), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_evmAddr), record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, toHex(_sbchLockTxHash), record0.SbchLockTxHash)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, "a2834f77f929353179fe8b7fc1e792f02fe56ebfcaa2b5eb55484818b6397a49",
		record0.SbchRefundTxHash)
	require.Equal(t, Bch2SbchStatusSbchRefunded, record0.Status)
}

func TestBch2Sbch_handleSbchOpenEvent_slaveMode(t *testing.T) {
	_val := uint64(12345678)
	_txHash := gethHash32Bytes("bchlock")
	_botPkh := gethAddrBytes("bot")
	_userPkh := gethAddrBytes("user")
	_hashLock := gethHash32Bytes("hash")
	_timeLock := uint32(100)
	_userEvmAddr := gethAddr("uevm")
	_scriptHash := gethAddrBytes("htlc")

	_sbchLockTxHash := gethHash32("sbchlocktx")
	_botEvmAddr := gethAddr("botevm")
	_userBchPkh := gethAddrBytes("ubch")
	_createdAt := int64ToBytes32(987600000)
	_penaltyBPS := int64ToBytes32(500)

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_userEvmAddr[:]),
		HtlcScriptHash: toHex(_scriptHash),
		Status:         Bch2SbchStatusNew,
	}))

	_sbchCli := newMockSbchClient(457, 999, 0)
	_sbchCli.logs[459] = []gethtypes.Log{
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethAddrToHash32(_botEvmAddr),
				gethAddrToHash32(_userEvmAddr),
			},
			Data: joinBytes(
				_hashLock,
				int64ToBytes32(int64(_timeLock)),
				satsToWeiBytes32(_val),
				rightPad0(_userBchPkh, 12),
				_createdAt,
				_penaltyBPS,
			),
		},
	}

	_bot := &MarketMakerBot{
		db:          _db,
		sbchCli:     _sbchCli,
		sbchAddr:    _botEvmAddr,
		isSlaveMode: true,
	}
	_bot.handleSbchEvents(457, 500)

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 0)

	bchLocked, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 100)
	require.NoError(t, err)
	require.Len(t, bchLocked, 1)

	record0 := bchLocked[0]
	require.Equal(t, toHex(_txHash), record0.BchLockTxHash)
	require.Equal(t, _val, record0.Value)
	require.Equal(t, toHex(_botPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_userEvmAddr[:]), record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusSbchLocked, record0.Status)
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Greater(t, record0.SbchLockTxTime, uint64(0))
}

func TestBch2Sbch_handleBchReceiptTxs(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethHash32Bytes("secret")
	_bchLockTxHash := bchHash32("bchlocktx")
	_userPkh := gethAddrBytes("user")
	_hashLock := sha256.Sum256(_secret)
	_timeLock := uint32(100)
	_evmAddr := gethAddrBytes("evm")
	_sbchLockTxHash := gethHash32Bytes("sbchlock")
	_userBchPkh := gethAddrBytes("ubch")

	c, err := htlcbch.NewMainnetCovenant(
		testBchPkh,
		_userBchPkh,
		_hashLock[:],
		uint16(_timeLock),
		0,
	)
	require.NoError(t, err)
	_scriptHash, err := c.GetRedeemScriptHash()
	require.NoError(t, err)
	_sigScript, err := c.BuildReceiveSigScript([]byte{'s', 'i', 'g'}, testBchPubKey, _secret)
	require.NoError(t, err)

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  122,
		BchLockTxHash:  _bchLockTxHash.String(),
		Value:          _val,
		RecipientPkh:   toHex(testBchPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock[:]),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
		SbchLockTxHash: toHex(_sbchLockTxHash),
		Secret:         toHex(_secret),
		Status:         Bch2SbchStatusSecretRevealed,
	}))

	_bchCli := newMockBchClient(122, 129)
	_bchCli.blocks[127] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Hash: _bchLockTxHash,
						},
						SignatureScript: _sigScript,
					},
				},
				TxOut: []*wire.TxOut{},
			},
		},
	}

	_bot := &MarketMakerBot{
		db:     _db,
		bchCli: _bchCli,
		bchPkh: testBchPkh,
	}

	_bot.scanBchBlocks()

	bchUnlocked, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusBchUnlocked, 100)
	require.NoError(t, err)
	require.Len(t, bchUnlocked, 1)
	record0 := bchUnlocked[0]
	require.Equal(t, "93e1119f2af3efda8bce6077be24e15c583ffc501621f2914335f507179199bc",
		record0.BchUnlockTxHash)
}

func TestBch2Sbch_handleSbchExpireEvent(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethHash32("secret")
	_bchLockTxHash := gethHash32("bchlock")
	_userPkh := gethAddrBytes("user")
	_hashLock := sha256.Sum256(_secret[:])
	_timeLock := uint32(72)
	_evmAddr := gethAddrBytes("evm")
	_scriptHash := gethAddrBytes("htlc")
	_sbchLockTxHash := gethHash32("sbchlock")
	_sbchRefundTxHash := gethHash32("sbchrefund")
	_sbchNow := uint64(time.Now().Unix())
	_sbchLockTxTime := _sbchNow - 22000

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  122,
		BchLockTxHash:  toHex(_bchLockTxHash.Bytes()),
		Value:          _val,
		RecipientPkh:   toHex(testBchPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock[:]),
		TimeLock:       _timeLock,
		SenderEvmAddr:  toHex(_evmAddr),
		HtlcScriptHash: toHex(_scriptHash),
		SbchLockTxTime: _sbchLockTxTime,
		SbchLockTxHash: toHex(_sbchLockTxHash[:]),
		Status:         Bch2SbchStatusSbchLocked,
	}))

	_sbchCli := newMockSbchClient(457, 999, _sbchNow)
	_sbchCli.logs[458] = []gethtypes.Log{
		{
			BlockNumber: 458,
			TxHash:      _sbchRefundTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.ExpireEventId,
				_hashLock,
			},
		},
	}
	_bot := &MarketMakerBot{
		db:      _db,
		sbchCli: _sbchCli,
		bchPkh:  testBchPkh,
	}

	_bot.handleSbchEvents(457, 500)

	secretRevealed, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchRefunded, 100)
	require.NoError(t, err)
	require.Len(t, secretRevealed, 1)
	record0 := secretRevealed[0]
	require.Equal(t, toHex(_bchLockTxHash.Bytes()), record0.BchLockTxHash)
	require.Equal(t, _val, record0.Value)
	require.Equal(t, toHex(testBchPkh), record0.RecipientPkh)
	require.Equal(t, toHex(_userPkh), record0.SenderPkh)
	require.Equal(t, toHex(_hashLock[:]), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_evmAddr), record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, "73626368726566756e6400000000000000000000000000000000000000000000",
		record0.SbchRefundTxHash)
	require.Equal(t, Bch2SbchStatusSbchRefunded, record0.Status)
}

func TestSbch2Bch_userLockSbch(t *testing.T) {
	_sbchLockTxHash := gethHash32("sbchlocktx")
	_userEvmAddr := gethAddr("uevm")
	_hashLock := gethHash32("hashlock")
	_val := satsToWeiBytes32(12345678)
	_userBchPkh := gethAddrBytes("ubch")
	_createdAt := int64ToBytes32(987600000)
	_timeLock := int64ToBytes32(987600000 + 12*3600)
	_penaltyBPS := int64ToBytes32(500)

	_db := initDB(t, 123, 456)
	_sbchCli := newMockSbchClient(457, 999, 0)
	_sbchCli.logs[459] = []gethtypes.Log{
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethAddrToHash32(_userEvmAddr),
				gethAddrToHash32(testEvmAddr),
			},
			Data: joinBytes(_hashLock.Bytes(), _timeLock, _val, rightPad0(_userBchPkh, 12),
				_createdAt, _penaltyBPS),
		},
	}
	_bot := &MarketMakerBot{
		db:           _db,
		sbchCli:      _sbchCli,
		sbchAddr:     testEvmAddr,
		bchPkh:       testBchPkh,
		sbchTimeLock: 12 * 3600,
		penaltyRatio: 500,
	}
	_bot.scanSbchEvents()

	newH, err := _db.getLastSbchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(999), newH)

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, uint64(987600000), record0.SbchLockTime)
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Equal(t, uint64(12345678), record0.Value)
	require.Equal(t, toHex(_userEvmAddr[:]), record0.SbchSenderAddr)
	require.Equal(t, toHex(_userBchPkh), record0.BchRecipientPkh)
	require.Equal(t, toHex(_hashLock[:]), record0.HashLock)
	require.Equal(t, uint32(12*3600), record0.TimeLock)
	require.Equal(t, "7e7dcbfcc8c9ce21ee2d5f0a9e73e64469cba144",
		record0.HtlcScriptHash)
	require.Equal(t, "", record0.BchLockTxHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusNew, record0.Status)
}

func TestSbch2Bch_userLockSbch_invalidParams(t *testing.T) {
	_sbchLockTxHash := gethHash32("sbchlocktx")
	_userEvmAddr := gethAddr("uevm")
	_hashLock := gethHash32("hashlock")
	_userBchPkh := gethAddrBytes("ubch")
	_createdAt := time.Now().Unix()
	_penaltyBPS := uint16(500)
	_sbchTimeLock := uint32(12 * 3600)
	_minSwapVal := uint64(100000)
	_maxSwapVal := uint64(999999)

	_db := initDB(t, 123, 456)
	_sbchCli := newMockSbchClient(457, 999, 0)
	_sbchCli.logs[459] = []gethtypes.Log{
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethAddrToHash32(_userEvmAddr),
				gethAddrToHash32(testEvmAddr),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				int64ToBytes32(_createdAt+int64(_sbchTimeLock)),
				satsToWeiBytes32(_minSwapVal+1),
				rightPad0(_userBchPkh, 12),
				int64ToBytes32(_createdAt),
				int64ToBytes32(int64(_penaltyBPS/2)), // invalid penaltyBPS
			),
		},
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethAddrToHash32(_userEvmAddr),
				gethAddrToHash32(testEvmAddr),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				int64ToBytes32(_createdAt+int64(_sbchTimeLock/2)), // invalid sbchTimeLock
				satsToWeiBytes32(_minSwapVal+1),
				rightPad0(_userBchPkh, 12),
				int64ToBytes32(_createdAt),
				int64ToBytes32(int64(_penaltyBPS)),
			),
		},
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethAddrToHash32(_userEvmAddr),
				gethAddrToHash32(testEvmAddr),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				int64ToBytes32(_createdAt+int64(_sbchTimeLock)),
				satsToWeiBytes32(_minSwapVal-1), // minSwapVal too small
				rightPad0(_userBchPkh, 12),
				int64ToBytes32(_createdAt),
				int64ToBytes32(int64(_penaltyBPS)),
			),
		},
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethAddrToHash32(_userEvmAddr),
				gethAddrToHash32(testEvmAddr),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				int64ToBytes32(_createdAt+int64(_sbchTimeLock)),
				satsToWeiBytes32(_maxSwapVal+1), // maxSwapVal too large
				rightPad0(_userBchPkh, 12),
				int64ToBytes32(_createdAt),
				int64ToBytes32(int64(_penaltyBPS)),
			),
		},
	}
	_bot := &MarketMakerBot{
		db:           _db,
		sbchCli:      _sbchCli,
		sbchAddr:     testEvmAddr,
		bchPkh:       testBchPkh,
		sbchTimeLock: _sbchTimeLock,
		penaltyRatio: _penaltyBPS,
		minSwapVal:   _minSwapVal,
		maxSwapVal:   _maxSwapVal,
	}
	_bot.scanSbchEvents()

	newH, err := _db.getLastSbchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(999), newH)

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, records, 0)
}

func TestSbch2Bch_botLockBch(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_hashLock := gethHash32Bytes("hashlock")
	_lockTime := uint64(1683248875) // time.Now().Unix()
	_timeLock := uint32(36000)
	_userBchPkh := gethAddrBytes("ubch")
	_scriptHash := gethAddrBytes("htlc")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     _lockTime,
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         _timeLock,
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    "",
		Secret:           "",
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusNew,
	}))

	_bchCli := &MockBchClient{}
	_sbchCli := newMockSbchClient(457, 500, _lockTime+60)
	_bot := &MarketMakerBot{
		db:           _db,
		bchCli:       _bchCli,
		bchPrivKey:   testBchPrivKey,
		bchPkh:       testBchPkh,
		sbchCli:      _sbchCli,
		sbchAddr:     testEvmAddr,
		sbchTimeLock: _timeLock,
	}

	_bot.handleSbchUserDeposits()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Equal(t, uint64(12345678), record0.Value)
	require.Equal(t, _userEvmAddr.String(), record0.SbchSenderAddr)
	require.Equal(t, toHex(_userBchPkh), record0.BchRecipientPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, uint32(36000), record0.TimeLock)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, "a19d7ad2f4013aebb8d4360478d242992fad16ce0aaf230a240a2fc7836a1257",
		record0.BchLockTxHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusBchLocked, record0.Status)
}

func TestSbch2Bch_botLockBch_tooLate(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_hashLock := gethHash32Bytes("hashlock")
	_lockTime := uint64(time.Now().Unix())
	_timeLock := uint32(36000)
	_userBchPkh := gethAddrBytes("ubch")
	_scriptHash := gethAddrBytes("htlc")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     _lockTime,
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         _timeLock,
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    "",
		Secret:           "",
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusNew,
	}))

	_bchCli := &MockBchClient{}
	_sbchCli := newMockSbchClient(457, 500, _lockTime+uint64(_timeLock/3)+1)
	_bot := &MarketMakerBot{
		db:           _db,
		bchCli:       _bchCli,
		bchPrivKey:   testBchPrivKey,
		bchPkh:       testBchPkh,
		sbchCli:      _sbchCli,
		sbchAddr:     testEvmAddr,
		sbchTimeLock: _timeLock,
	}

	_bot.handleSbchUserDeposits()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 100)
	require.NoError(t, err)
	require.Len(t, records, 0)

	toLate, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusTooLateToLockSbch, 100)
	require.NoError(t, err)
	require.Len(t, toLate, 1)
}

func TestSbch2Bch_userUnlockBch(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_secret := gethHash32Bytes("secret")
	_hashLock := gethcmn.FromHex(secretToHashLock(_secret))
	_timeLock := uint16(888)
	_userBchPkh := gethAddrBytes("ubch")
	_bchLockTxHash := bchHash32("bchlocktx")

	c, err := htlcbch.NewMainnetCovenant(
		testBchPkh,
		_userBchPkh,
		_hashLock,
		_timeLock,
		0,
	)
	require.NoError(t, err)
	_scriptHash, err := c.GetRedeemScriptHash()
	require.NoError(t, err)
	_sigScript, err := c.BuildReceiveSigScript([]byte{'s', 'i', 'g'}, testBchPubKey, _secret)
	require.NoError(t, err)

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     uint64(time.Now().Unix()),
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         uint32(_timeLock),
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    _bchLockTxHash.String(),
		Secret:           "",
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusBchLocked,
	}))

	_bchCli := newMockBchClient(122, 129)
	_bchCli.blocks[127] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Hash: _bchLockTxHash,
						},
						SignatureScript: _sigScript,
					},
				},
				TxOut: []*wire.TxOut{},
			},
		},
	}

	_bot := &MarketMakerBot{
		db:     _db,
		bchCli: _bchCli,
		bchPkh: testBchPkh,
	}

	_bot.scanBchBlocks()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusSecretRevealed, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Equal(t, uint64(12345678), record0.Value)
	require.Equal(t, _userEvmAddr.String(), record0.SbchSenderAddr)
	require.Equal(t, toHex(_userBchPkh), record0.BchRecipientPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, uint32(888), record0.TimeLock)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, _bchLockTxHash.String(), record0.BchLockTxHash)
	require.Equal(t, "a9c6d5a3d9a35b22738c24d47ea374f116cfdbff9209979435c5f459787b0d91",
		record0.BchUnlockTxHash)
	require.Equal(t, toHex(_secret), record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusSecretRevealed, record0.Status)
}

func TestSbch2Bch_botUnlockSbch(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_secret := gethHash32Bytes("secret")
	_hashLock := gethHash32Bytes("hashlock")
	_timeLock := uint32(888)
	_scriptHash := gethAddrBytes("htlc")
	_userBchPkh := gethAddrBytes("ubch")
	_bchLockTxHash := bchHash32("bchlocktx")
	_bchUnlockTxHash := bchHash32("bchunlocktx")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     uint64(time.Now().Unix()),
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         _timeLock,
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    _bchLockTxHash.String(),
		BchUnlockTxHash:  _bchUnlockTxHash.String(),
		Secret:           toHex(_secret),
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusSecretRevealed,
	}))

	_sbchCli := &MockSbchClient{}
	_bot := &MarketMakerBot{
		db:      _db,
		sbchCli: _sbchCli,
	}

	_bot.unlockSbchUserDeposits()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusSbchUnlocked, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Equal(t, uint64(12345678), record0.Value)
	require.Equal(t, _userEvmAddr.String(), record0.SbchSenderAddr)
	require.Equal(t, toHex(_userBchPkh), record0.BchRecipientPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, uint32(888), record0.TimeLock)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, _bchLockTxHash.String(), record0.BchLockTxHash)
	require.Equal(t, _bchUnlockTxHash.String(), record0.BchUnlockTxHash)
	require.Equal(t, toHex(_secret), record0.Secret)
	require.Equal(t, "0000000000000000000000000000000000000000000000006b636f6c68736168",
		record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusSbchUnlocked, record0.Status)
}

func TestSbch2Bch_botRefundBch(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_hashLock := gethHash32Bytes("hashlock")
	_timeLock := uint32(72000)
	_userBchPkh := gethAddrBytes("ubch")
	_bchLockTxHash := bchHash32("bchlocktx")

	c, err := htlcbch.NewMainnetCovenant(
		testBchPkh,
		_userBchPkh,
		_hashLock,
		uint16(_timeLock/600),
		0,
	)
	require.NoError(t, err)
	_scriptHash, err := c.GetRedeemScriptHash()
	require.NoError(t, err)

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     uint64(time.Now().Unix()),
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         _timeLock,
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    _bchLockTxHash.String(),
		Secret:           "",
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusBchLocked,
	}))

	_bchCli := newMockBchClient(122, 129)
	_bchCli.confirmations[_bchLockTxHash.String()] = 61

	_bot := &MarketMakerBot{
		db:         _db,
		bchCli:     _bchCli,
		bchPrivKey: testBchPrivKey,
		bchPkh:     testBchPkh,
		bchAddr:    testBchAddr,
	}

	_bot.refundLockedBCH(true)

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchRefunded, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, toHex(_sbchLockTxHash[:]), record0.SbchLockTxHash)
	require.Equal(t, uint64(12345678), record0.Value)
	require.Equal(t, _userEvmAddr.String(), record0.SbchSenderAddr)
	require.Equal(t, toHex(_userBchPkh), record0.BchRecipientPkh)
	require.Equal(t, toHex(_hashLock), record0.HashLock)
	require.Equal(t, _timeLock, record0.TimeLock)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, _bchLockTxHash.String(), record0.BchLockTxHash)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, "5b65aa3b16069ebd2ff6ab0cec804ad052f85824e600d9fb44614d39acb03ff2",
		record0.BchRefundTxHash)
	require.Equal(t, Sbch2BchStatusBchRefunded, record0.Status)
}

// TODO
/*
func TestSbch2Bch_handleBchDepositTxS2B(t *testing.T) {
	_botPkh := testBchPkh
	_userPkh := gethAddrBytes("user")
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_hashLock := gethHash32Bytes("hashlock")
	_lockTime := uint64(time.Now().Unix())
	_sbchTimeLock := uint32(36000)
	_bchTimeLock := uint16(60)
	_userBchPkh := gethAddrBytes("ubch")
	_scriptHash := gethAddrBytes("htlc")
	_penaltyBPS := uint16(500)

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     _lockTime,
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         _sbchTimeLock,
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    "",
		Secret:           "",
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusNew,
	}))

	covenant, err := htlcbch.NewMainnetCovenant(_botPkh, _userPkh, _hashLock, _bchTimeLock, _penaltyBPS)
	require.NoError(t, err)
	scriptHash, err := covenant.GetRedeemScriptHash()
	require.NoError(t, err)

	_bchCli := &MockBchClient{}
	_bchCli.blocks[126] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{},
				TxOut: []*wire.TxOut{
					{
						Value:    12345678,
						PkScript: newP2SHPkScript(scriptHash),
					},
					{
						PkScript: newHtlcDepositOpRet(_botPkh, _userPkh, _hashLock, _bchTimeLock, _penaltyBPS, _userEvmAddr[:]),
					},
				},
			},
		},
	}

	_bot := &MarketMakerBot{
		db:           _db,
		bchCli:       _bchCli,
		bchPrivKey:   testBchPrivKey,
		bchPkh:       testBchPkh,
		sbchAddr:     testEvmAddr,
		sbchTimeLock: _sbchTimeLock,
	}

	_bot.scanBchBlocks()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 100)
	require.NoError(t, err)
	require.Len(t, records, 0)

	toLate, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusTooLateToLockSbch, 100)
	require.NoError(t, err)
	require.Len(t, toLate, 1)
}
*/

func TestSbch2Bch_handleSbchCloseEventS2B(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_secret := gethHash32("secret")
	_bchLockTxHash := gethHash32("bchlock")
	_hashLock := sha256.Sum256(_secret[:])
	_timeLock := uint32(888)
	_scriptHash := gethAddrBytes("htlc")
	_userBchPkh := gethAddrBytes("ubch")
	_bchUnlockTxHash := bchHash32("bchunlocktx")
	_sbchCloseTxHash := gethHash32("close")

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     uint64(time.Now().Unix()),
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock[:]),
		TimeLock:         _timeLock,
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    _bchLockTxHash.String(),
		BchUnlockTxHash:  _bchUnlockTxHash.String(),
		Secret:           toHex(_secret[:]),
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusSecretRevealed,
	}))

	_sbchCli := newMockSbchClient(457, 999, 0)
	_sbchCli.logs[458] = []gethtypes.Log{
		{
			TxHash: _sbchCloseTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.CloseEventId,
				_hashLock,
				_secret,
			},
		},
	}

	_bot := &MarketMakerBot{
		db:      _db,
		sbchCli: _sbchCli,
	}

	_bot.scanSbchEvents()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusSbchUnlocked, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)

	record0 := records[0]
	require.Equal(t, Sbch2BchStatusSbchUnlocked, record0.Status)
	require.Equal(t, toHex(_sbchCloseTxHash[:]), record0.SbchUnlockTxHash)
}

func TestSbch2Bch_handleBchRefundTxs(t *testing.T) {
	_sbchLockTxHash := gethHash32Bytes("sbchlocktx")
	_val := uint64(12345678)
	_userEvmAddr := gethAddr("uevm")
	_secret := gethHash32Bytes("secret")
	_hashLock := gethcmn.FromHex(secretToHashLock(_secret))
	_timeLock := uint16(888)
	_userBchPkh := gethAddrBytes("ubch")
	_bchLockTxHash := bchHash32("bchlocktx")

	c, err := htlcbch.NewMainnetCovenant(
		testBchPkh,
		_userBchPkh,
		_hashLock,
		_timeLock,
		0,
	)
	require.NoError(t, err)
	_scriptHash, err := c.GetRedeemScriptHash()
	require.NoError(t, err)
	_sigScript, err := c.BuildRefundSigScript([]byte{'s', 'i', 'g'}, testBchPubKey)
	//fmt.Println(toHex(_sigScript))
	require.NoError(t, err)

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addSbch2BchRecord(&Sbch2BchRecord{
		SbchLockTime:     uint64(time.Now().Unix()),
		SbchLockTxHash:   toHex(_sbchLockTxHash),
		Value:            _val,
		SbchSenderAddr:   _userEvmAddr.String(),
		BchRecipientPkh:  toHex(_userBchPkh),
		HashLock:         toHex(_hashLock),
		TimeLock:         uint32(_timeLock),
		HtlcScriptHash:   toHex(_scriptHash),
		BchLockTxHash:    _bchLockTxHash.String(),
		Secret:           "",
		SbchUnlockTxHash: "",
		Status:           Sbch2BchStatusBchLocked,
	}))

	_bchCli := newMockBchClient(122, 129)
	_bchCli.blocks[127] = &wire.MsgBlock{
		Transactions: []*wire.MsgTx{
			{
				TxIn: []*wire.TxIn{
					{
						PreviousOutPoint: wire.OutPoint{
							Hash: _bchLockTxHash,
						},
						SignatureScript: _sigScript,
					},
				},
				TxOut: []*wire.TxOut{},
			},
		},
	}

	_bot := &MarketMakerBot{
		db:     _db,
		bchCli: _bchCli,
		bchPkh: testBchPkh,
	}

	_bot.scanBchBlocks()

	records, err := _db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchRefunded, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)
}
