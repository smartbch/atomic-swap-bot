package bot

import (
	"crypto/sha256"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
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
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h'}.Bytes()
	_timeLock := uint16(100)
	_penaltyBPS := uint16(500)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.Bytes()

	covenant, err := htlcbch.NewMainnetCovenant(_userPkh, testBchPkh, _hashLock, _timeLock, _penaltyBPS)
	require.NoError(t, err)
	scriptHash, err := covenant.GetRedeemScriptHash()
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
	require.Equal(t, toHex(testBchPkh), record0.RecipientPkh)
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
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h'}.Bytes()
	_timeLock := uint16(100)
	_penaltyBPS := uint16(500)
	_minSwapVal := uint64(100000)
	_maxSwapVal := uint64(999999)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.Bytes()

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
	_txHash := gethcmn.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_botPkh := gethcmn.Address{'b', 'o', 't'}.Bytes()
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h'}.Bytes()
	_timeLock := uint32(100)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.String()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  _evmAddr,
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
	_bot.scanBchBlocks()

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
	require.Equal(t, _evmAddr, record0.SenderEvmAddr)
	require.Equal(t, toHex(_scriptHash), record0.HtlcScriptHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusSbchLocked, record0.Status)
}

func TestBch2Sbch_botLockSbch_notConfirmed(t *testing.T) {
	_val := uint64(12345678)
	_txHash := gethcmn.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_botPkh := gethcmn.Address{'b', 'o', 't'}.Bytes()
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h'}.Bytes()
	_timeLock := uint32(72)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.String()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  _evmAddr,
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
	_txHash := gethcmn.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_botPkh := gethcmn.Address{'b', 'o', 't'}.Bytes()
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h'}.Bytes()
	_timeLock := uint32(72)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.String()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()

	_db := initDB(t, 123, 456)
	require.NoError(t, _db.addBch2SbchRecord(&Bch2SbchRecord{
		BchLockHeight:  123,
		BchLockTxHash:  toHex(_txHash),
		Value:          _val,
		RecipientPkh:   toHex(_botPkh),
		SenderPkh:      toHex(_userPkh),
		HashLock:       toHex(_hashLock),
		TimeLock:       _timeLock,
		SenderEvmAddr:  _evmAddr,
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
	_bot.scanBchBlocks()

	unhandled, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.Len(t, unhandled, 0)

	tooLate, err := _db.getBch2SbchRecordsByStatus(Bch2SbchStatusTooLateToLockSbch, 100)
	require.NoError(t, err)
	require.Len(t, tooLate, 1)
}

func TestBch2Sbch_userUnlockSbch(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethcmn.Hash{'s', 'e', 'c', 'r', 'e', 't'}
	_bchLockTxHash := gethcmn.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k'}
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := sha256.Sum256(_secret[:])
	_timeLock := uint32(100)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.Bytes()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()

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
	_secret := gethcmn.Hash{'s', 'e', 'c', 'r', 'e', 't'}.Bytes()
	_bchLockTxHash := gethcmn.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := sha256.Sum256(_secret)
	_timeLock := uint32(100)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.Bytes()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()

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
		bchPubKey:  testBchPubKey,
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
	require.Equal(t, "dad1c4460d8d617542dd5d2c77b10ce9d41becba759ecf13e55d81af6ecbf7ae", record0.BchUnlockTxHash)
	require.Equal(t, Bch2SbchStatusBchUnlocked, record0.Status)
}

func TestBch2Sbch_botRefundSbch(t *testing.T) {
	_val := uint64(12345678)
	_secret := gethcmn.Hash{'s', 'e', 'c', 'r', 'e', 't'}
	_bchLockTxHash := gethcmn.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k'}
	_userPkh := gethcmn.Address{'u', 's', 'e', 'r'}.Bytes()
	_hashLock := sha256.Sum256(_secret[:])
	_timeLock := uint32(72)
	_evmAddr := gethcmn.Address{'e', 'v', 'm'}.Bytes()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k'}.Bytes()
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

	_bot.handleSbchRefunds()

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
	require.Equal(t, "a2834f77f929353179fe8b7fc1e792f02fe56ebfcaa2b5eb55484818b6397a49", record0.SbchRefundTxHash)
	require.Equal(t, Bch2SbchStatusSbchRefunded, record0.Status)
}

func TestSbch2Bch_userLockSbch(t *testing.T) {
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k'}
	_val := satsToWei(12345678).FillBytes(make([]byte, 32))
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
	_createdAt := big.NewInt(987600000).FillBytes(make([]byte, 32))
	_timeLock := big.NewInt(987600000 + 12*3600).FillBytes(make([]byte, 32))
	_penaltyBPS := big.NewInt(500).FillBytes(make([]byte, 32))

	_db := initDB(t, 123, 456)
	_sbchCli := newMockSbchClient(457, 999, 0)
	_sbchCli.logs[459] = []gethtypes.Log{
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethcmn.BytesToHash(leftPad0(_userEvmAddr.Bytes(), 12)),
				gethcmn.BytesToHash(leftPad0(testEvmAddr.Bytes(), 12)),
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
	require.Equal(t, "d75827497e9df4ac13172fd0e47a07045a370b8f", record0.HtlcScriptHash)
	require.Equal(t, "", record0.BchLockTxHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusNew, record0.Status)
}

func TestSbch2Bch_userLockSbch_invalidParams(t *testing.T) {
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k'}
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
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
				gethcmn.BytesToHash(leftPad0(_userEvmAddr.Bytes(), 12)),
				gethcmn.BytesToHash(leftPad0(testEvmAddr.Bytes(), 12)),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				big.NewInt(_createdAt+int64(_sbchTimeLock)).FillBytes(make([]byte, 32)),
				satsToWei(_minSwapVal+1).FillBytes(make([]byte, 32)),
				rightPad0(_userBchPkh, 12),
				big.NewInt(_createdAt).FillBytes(make([]byte, 32)),
				big.NewInt(int64(_penaltyBPS/2)).FillBytes(make([]byte, 32)), // invalid penaltyBPS
			),
		},
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethcmn.BytesToHash(leftPad0(_userEvmAddr.Bytes(), 12)),
				gethcmn.BytesToHash(leftPad0(testEvmAddr.Bytes(), 12)),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				big.NewInt(_createdAt+int64(_sbchTimeLock/2)).FillBytes(make([]byte, 32)), // invalid sbchTimeLock
				satsToWei(_minSwapVal+1).FillBytes(make([]byte, 32)),
				rightPad0(_userBchPkh, 12),
				big.NewInt(_createdAt).FillBytes(make([]byte, 32)),
				big.NewInt(int64(_penaltyBPS)).FillBytes(make([]byte, 32)),
			),
		},
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethcmn.BytesToHash(leftPad0(_userEvmAddr.Bytes(), 12)),
				gethcmn.BytesToHash(leftPad0(testEvmAddr.Bytes(), 12)),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				big.NewInt(_createdAt+int64(_sbchTimeLock)).FillBytes(make([]byte, 32)),
				satsToWei(_minSwapVal-1).FillBytes(make([]byte, 32)), // minSwapVal too small
				rightPad0(_userBchPkh, 12),
				big.NewInt(_createdAt).FillBytes(make([]byte, 32)),
				big.NewInt(int64(_penaltyBPS)).FillBytes(make([]byte, 32)),
			),
		},
		{
			BlockNumber: 459,
			TxHash:      _sbchLockTxHash,
			Topics: []gethcmn.Hash{
				htlcsbch.OpenEventId,
				gethcmn.BytesToHash(leftPad0(_userEvmAddr.Bytes(), 12)),
				gethcmn.BytesToHash(leftPad0(testEvmAddr.Bytes(), 12)),
			},
			Data: joinBytes(
				_hashLock.Bytes(),
				big.NewInt(_createdAt+int64(_sbchTimeLock)).FillBytes(make([]byte, 32)),
				satsToWei(_maxSwapVal+1).FillBytes(make([]byte, 32)), // maxSwapVal too large
				rightPad0(_userBchPkh, 12),
				big.NewInt(_createdAt).FillBytes(make([]byte, 32)),
				big.NewInt(int64(_penaltyBPS)).FillBytes(make([]byte, 32)),
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
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}.Bytes()
	_val := uint64(12345678)
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_lockTime := uint64(1683248875) // time.Now().Unix()
	_timeLock := uint32(36000)
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()

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
	require.Equal(t, "f63b1a9b771f1991a85482f8158218388f198be483d842df12036ebc357bdadd", record0.BchLockTxHash)
	require.Equal(t, "", record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusBchLocked, record0.Status)
}

func TestSbch2Bch_botLockBch_tooLate(t *testing.T) {
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}.Bytes()
	_val := uint64(12345678)
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_lockTime := uint64(time.Now().Unix())
	_timeLock := uint32(36000)
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()

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
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}.Bytes()
	_val := uint64(12345678)
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_secret := gethcmn.Hash{'s', 'e', 'c', 'r', 'e', 't'}.Bytes()
	_hashLock := gethcmn.FromHex(secretToHashLock(_secret))
	_timeLock := uint16(888)
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
	_bchLockTxHash := chainhash.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}

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
	require.Equal(t, "fd51ff8a2386636abe7f1c5e33e1a4a8ea540ac181b97a57653e08207f893ad0", record0.BchUnlockTxHash)
	require.Equal(t, toHex(_secret), record0.Secret)
	require.Equal(t, "", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusSecretRevealed, record0.Status)
}

func TestSbch2Bch_botUnlockSbch(t *testing.T) {
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}.Bytes()
	_val := uint64(12345678)
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_secret := gethcmn.Hash{'s', 'e', 'c', 'r', 'e', 't'}.Bytes()
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_timeLock := uint32(888)
	_scriptHash := gethcmn.Address{'h', 't', 'l', 'c'}.Bytes()
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
	_bchLockTxHash := chainhash.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}
	_bchUnlockTxHash := chainhash.Hash{'b', 'c', 'h', 'u', 'n', 'l', 'o', 'c', 'k', 't', 'x'}

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
	require.Equal(t, "0000000000000000000000000000000000000000000000006b636f6c68736168", record0.SbchUnlockTxHash)
	require.Equal(t, Sbch2BchStatusSbchUnlocked, record0.Status)
}

func TestSbch2Bch_botRefundBch(t *testing.T) {
	_sbchLockTxHash := gethcmn.Hash{'s', 'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}.Bytes()
	_val := uint64(12345678)
	_userEvmAddr := gethcmn.Address{'u', 'e', 'v', 'm'}
	_hashLock := gethcmn.Hash{'h', 'a', 's', 'h', 'l', 'o', 'c', 'k'}.Bytes()
	_timeLock := uint32(72000)
	_userBchPkh := gethcmn.Address{'u', 'b', 'c', 'h'}.Bytes()
	_bchLockTxHash := chainhash.Hash{'b', 'c', 'h', 'l', 'o', 'c', 'k', 't', 'x'}

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

	_bot.scanBchBlocks()

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
	require.Equal(t, "c38a2ca5e0f3750c032f57483513fd3f35f43455cfe9031cf989165aa7c7ecf4", record0.BchRefundTxHash)
	require.Equal(t, Sbch2BchStatusBchRefunded, record0.Status)
}
