package bot

import (
	"encoding/json"
	"fmt"
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

func TestGetBch2SbchRecordsByStatus_orderBy(t *testing.T) {
	db := initDB(t, 123, 456)

	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(100)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(111)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(222)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(333)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(444)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(555)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(666)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(777)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(888)))
	require.NoError(t, db.addBch2SbchRecord(createFakeBch2SbchRecord(999)))

	records, err := db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 6)
	require.NoError(t, err)
	require.Equal(t, []uint64{100, 111, 222, 333, 444, 555}, getBch2SbchRecordValues(records))

	records, err = db.getBch2SbchRecordsByStatus(Bch2SbchStatusNew, 100)
	require.NoError(t, err)
	require.NoError(t, db.updateBch2SbchRecord(records[9].UpdateStatusToSbchLocked("txhash", 1234)))
	require.NoError(t, db.updateBch2SbchRecord(records[8].UpdateStatusToSbchLocked("txhash", 1234)))
	require.NoError(t, db.updateBch2SbchRecord(records[7].UpdateStatusToSbchLocked("txhash", 1234)))
	require.NoError(t, db.updateBch2SbchRecord(records[5].UpdateStatusToSbchLocked("txhash", 1234)))
	require.NoError(t, db.updateBch2SbchRecord(records[3].UpdateStatusToSbchLocked("txhash", 1234)))
	records, err = db.getBch2SbchRecordsByStatus(Bch2SbchStatusSbchLocked, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{999, 888, 777, 555, 333}, getBch2SbchRecordValues(records))

	require.NoError(t, db.updateBch2SbchRecord(records[3].UpdateStatusToSecretRevealed("secret", "txhash")))
	require.NoError(t, db.updateBch2SbchRecord(records[2].UpdateStatusToSecretRevealed("secret", "txhash")))
	require.NoError(t, db.updateBch2SbchRecord(records[1].UpdateStatusToSecretRevealed("secret", "txhash")))
	records, err = db.getBch2SbchRecordsByStatus(Bch2SbchStatusSecretRevealed, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{555, 777, 888}, getBch2SbchRecordValues(records))
}

func TestGetSbch2BchRecordsByStatus_orderBy(t *testing.T) {
	db := initDB(t, 123, 456)

	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(100)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(111)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(222)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(333)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(444)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(555)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(666)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(777)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(888)))
	require.NoError(t, db.addSbch2BchRecord(createFakeSbch2BchRecord(999)))

	records, err := db.getSbch2BchRecordsByStatus(Sbch2BchStatusNew, 6)
	require.NoError(t, err)
	require.Equal(t, []uint64{100, 111, 222, 333, 444, 555}, getSbch2BchRecordValues(records))

	records, err = db.getSbch2BchRecordsByStatus(Sbch2BchStatusNew, 100)
	require.NoError(t, err)
	require.NoError(t, db.updateSbch2BchRecord(records[9].UpdateStatusToBchLocked("txhash", 1)))
	require.NoError(t, db.updateSbch2BchRecord(records[8].UpdateStatusToBchLocked("txhash", 2)))
	require.NoError(t, db.updateSbch2BchRecord(records[7].UpdateStatusToBchLocked("txhash", 3)))
	require.NoError(t, db.updateSbch2BchRecord(records[5].UpdateStatusToBchLocked("txhash", 4)))
	require.NoError(t, db.updateSbch2BchRecord(records[3].UpdateStatusToBchLocked("txhash", 5)))
	records, err = db.getSbch2BchRecordsByStatus(Sbch2BchStatusBchLocked, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{999, 888, 777, 555, 333}, getSbch2BchRecordValues(records))

	require.NoError(t, db.updateSbch2BchRecord(records[3].UpdateStatusToSecretRevealed("secret", "txhash")))
	require.NoError(t, db.updateSbch2BchRecord(records[2].UpdateStatusToSecretRevealed("secret", "txhash")))
	require.NoError(t, db.updateSbch2BchRecord(records[1].UpdateStatusToSecretRevealed("secret", "txhash")))
	records, err = db.getSbch2BchRecordsByStatus(Sbch2BchStatusSecretRevealed, 10)
	require.NoError(t, err)
	require.Equal(t, []uint64{555, 777, 888}, getSbch2BchRecordValues(records))
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

func createFakeBch2SbchRecord(fakeN uint) *Bch2SbchRecord {
	return &Bch2SbchRecord{
		BchLockHeight:  uint64(fakeN),
		BchLockTxHash:  fmt.Sprintf("%d", fakeN),
		Value:          uint64(fakeN),
		RecipientPkh:   fmt.Sprintf("%d", fakeN),
		SenderPkh:      fmt.Sprintf("%d", fakeN),
		HashLock:       fmt.Sprintf("%d", fakeN),
		TimeLock:       uint32(fakeN),
		HtlcScriptHash: fmt.Sprintf("%d", fakeN),
		SenderEvmAddr:  fmt.Sprintf("%d", fakeN),
	}
}
func createFakeSbch2BchRecord(fakeN uint) *Sbch2BchRecord {
	return &Sbch2BchRecord{
		SbchLockTime:    uint64(fakeN),
		SbchLockTxHash:  fmt.Sprintf("%d", fakeN),
		Value:           uint64(fakeN),
		SbchSenderAddr:  fmt.Sprintf("%d", fakeN),
		BchRecipientPkh: fmt.Sprintf("%d", fakeN),
		HashLock:        fmt.Sprintf("%d", fakeN),
		TimeLock:        uint32(fakeN),
		HtlcScriptHash:  fmt.Sprintf("%d", fakeN),
	}
}

func getBch2SbchRecordValues(records []*Bch2SbchRecord) []uint64 {
	vals := make([]uint64, len(records))
	for i, record := range records {
		vals[i] = record.Value
	}
	return vals
}
func getSbch2BchRecordValues(records []*Sbch2BchRecord) []uint64 {
	vals := make([]uint64, len(records))
	for i, record := range records {
		vals[i] = record.Value
	}
	return vals
}
