package bot

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testDbFile = "test.db"
)

func TestLastHeights(t *testing.T) {
	db := initDB(t, 123, 456)

	h1, err := db.getLastBchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(123), h1)

	h2, err := db.getLastSbchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(456), h2)

	require.NoError(t, db.setLastBchHeight(321))
	h3, err := db.getLastBchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(321), h3)

	require.NoError(t, db.setLastSbchHeight(654))
	h4, err := db.getLastSbchHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(654), h4)
}

func TestAddBch2SbchRecord(t *testing.T) {
	//defer os.Remove(testDbFile)
	db := initDB(t, 123, 456)

	record := &Bch2SbchRecord{
		BchLockHeight:  11,
		BchLockTxHash:  "22",
		Value:          44,
		RecipientPkh:   "55",
		SenderPkh:      "66",
		HashLock:       "77",
		TimeLock:       88,
		HtlcScriptHash: "99",
		SenderEvmAddr:  "aa",
	}

	testCases := []func(*Bch2SbchRecord){
		func(r *Bch2SbchRecord) { r.BchLockHeight = 0 },
		func(r *Bch2SbchRecord) { r.BchLockTxHash = "" },
		func(r *Bch2SbchRecord) { r.Value = 0 },
		func(r *Bch2SbchRecord) { r.RecipientPkh = "" },
		func(r *Bch2SbchRecord) { r.SenderPkh = "" },
		func(r *Bch2SbchRecord) { r.HashLock = "" },
		func(r *Bch2SbchRecord) { r.BchLockHeight = 0 },
		func(r *Bch2SbchRecord) { r.HtlcScriptHash = "" },
		func(r *Bch2SbchRecord) { r.SenderEvmAddr = "" },
	}
	for _, fn := range testCases {
		record2 := cloneBch2SbchRecord(record)
		fn(record2)
		require.ErrorContains(t, db.addBch2SbchRecord(record2),
			"missing required fields")
	}

	require.NoError(t, db.addBch2SbchRecord(record))

	record2 := cloneBch2SbchRecord(record)
	record2.ID = 0
	record2.BchLockTxHash = "xx"
	require.ErrorContains(t, db.addBch2SbchRecord(record2),
		"UNIQUE constraint failed")

	record2.BchLockTxHash = "22"
	record2.HashLock = "yy"
	require.ErrorContains(t, db.addBch2SbchRecord(record2),
		"UNIQUE constraint failed")

	record2.BchLockTxHash = "xx"
	record2.HashLock = "yy"
	require.NoError(t, db.addBch2SbchRecord(record2))

	records, err := db.GetAllBch2SbchRecords()
	require.NoError(t, err)
	require.Len(t, records, 2)
}

func TestAddSbch2BchRecord(t *testing.T) {
	//defer os.Remove(testDbFile)
	db := initDB(t, 123, 456)

	record := &Sbch2BchRecord{
		SbchLockTime:    11,
		SbchLockTxHash:  "22",
		Value:           44,
		SbchSenderAddr:  "55",
		BchRecipientPkh: "66",
		HashLock:        "77",
		TimeLock:        88,
		HtlcScriptHash:  "99",
	}

	testCases := []func(*Sbch2BchRecord){
		func(r *Sbch2BchRecord) { r.SbchLockTime = 0 },
		func(r *Sbch2BchRecord) { r.SbchLockTxHash = "" },
		func(r *Sbch2BchRecord) { r.Value = 0 },
		func(r *Sbch2BchRecord) { r.SbchSenderAddr = "" },
		func(r *Sbch2BchRecord) { r.BchRecipientPkh = "" },
		func(r *Sbch2BchRecord) { r.HashLock = "" },
		func(r *Sbch2BchRecord) { r.TimeLock = 0 },
		func(r *Sbch2BchRecord) { r.HtlcScriptHash = "" },
	}
	for _, fn := range testCases {
		record2 := cloneSbch2BchRecord(record)
		fn(record2)
		require.ErrorContains(t, db.addSbch2BchRecord(record2),
			"missing required fields")
	}

	require.NoError(t, db.addSbch2BchRecord(record))

	record2 := cloneSbch2BchRecord(record)
	record2.ID = 0
	record2.SbchLockTxHash = "xx"
	require.ErrorContains(t, db.addSbch2BchRecord(record2),
		"UNIQUE constraint failed")

	record2.SbchLockTxHash = "22"
	record2.HashLock = "yy"
	require.ErrorContains(t, db.addSbch2BchRecord(record2),
		"UNIQUE constraint failed")

	record2.SbchLockTxHash = "xx"
	record2.HashLock = "yy"
	require.NoError(t, db.addSbch2BchRecord(record2))

	records, err := db.GetAllSbch2BchRecords()
	require.NoError(t, err)
	require.Len(t, records, 2)
}

func TestUpdatedAt(t *testing.T) {
	//defer os.Remove(testDbFile)
	db := initDB(t, 123, 456)

	record := &Sbch2BchRecord{
		SbchLockTime:    11,
		SbchLockTxHash:  "22",
		Value:           44,
		SbchSenderAddr:  "55",
		BchRecipientPkh: "66",
		HashLock:        "77",
		TimeLock:        88,
		HtlcScriptHash:  "99",
	}

	err := db.addSbch2BchRecord(record)
	require.NoError(t, err)

	record.Status = Sbch2BchStatusBchLocked
	record.BchLockTxHash = "hh"
	err = db.updateSbch2BchRecord(record)
	require.NoError(t, err)

	records, err := db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 100)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.True(t, time.Now().Sub(records[0].UpdatedAt).Seconds() < 2)
}

func initDB(t *testing.T, lastBchHeight, lastSbchHeight uint64) DB {
	_ = os.Remove(testDbFile)
	db, err := OpenDB(testDbFile)
	require.NoError(t, err)
	require.NoError(t, db.syncSchemas())
	require.NoError(t, db.initLastHeights(lastBchHeight, lastSbchHeight))
	return db
}

func cloneBch2SbchRecord(record *Bch2SbchRecord) *Bch2SbchRecord {
	var record2 Bch2SbchRecord
	bz, _ := json.Marshal(record)
	_ = json.Unmarshal(bz, &record2)
	return &record2
}
func cloneSbch2BchRecord(record *Sbch2BchRecord) *Sbch2BchRecord {
	var record2 Sbch2BchRecord
	bz, _ := json.Marshal(record)
	_ = json.Unmarshal(bz, &record2)
	return &record2
}
