package bot

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type (
	Sbch2BchStatus int
	Bch2SbchStatus int
)

const (
	Bch2SbchStatusNew Bch2SbchStatus = iota
	Bch2SbchStatusSbchLocked
	Bch2SbchStatusSecretRevealed
	Bch2SbchStatusBchUnlocked
	Bch2SbchStatusSbchRefunded
	Bch2SbchStatusTooLateToLockSbch
)

const (
	Sbch2BchStatusNew Sbch2BchStatus = iota
	Sbch2BchStatusBchLocked
	Sbch2BchStatusSecretRevealed
	Sbch2BchStatusSbchUnlocked
	Sbch2BchStatusBchRefunded
	Sbch2BchStatusTooLateToLockSbch
)

type LastHeights struct {
	gorm.Model
	LastBchHeight  uint64
	LastSbchHeight uint64
}

type Bch2SbchRecord struct {
	gorm.Model
	BchLockHeight    uint64         `gorm:"not null"` // got from tx
	BchLockTxHash    string         `gorm:"unique"`   // got from tx
	Value            uint64         `gorm:"not null"` // got from tx, in Sats
	RecipientPkh     string         `gorm:"not null"` // got from retData
	SenderPkh        string         `gorm:"not null"` // got from retData
	HashLock         string         `gorm:"unique"`   // got from retData, in Blocks
	TimeLock         uint32         `gorm:"not null"` // got from retData
	PenaltyBPS       uint16         `gorm:"not null"` // got from retData
	SenderEvmAddr    string         `gorm:"not null"` // got from retData
	HtlcScriptHash   string         `gorm:"not null"` // calculated
	SbchLockTxTime   uint64         ``                // set when status changed to Bch2SbchStatusSbchLocked
	SbchLockTxHash   string         ``                // set when status changed to Bch2SbchStatusSbchLocked
	SbchUnlockTxHash string         ``                // set when status changed to Bch2SbchStatusSecretRevealed
	Secret           string         ``                // set when status changed to Bch2SbchStatusSecretRevealed
	BchUnlockTxHash  string         ``                // set when status changed to Bch2SbchStatusBchUnlocked
	SbchRefundTxHash string         ``                // set when status changed to Bch2SbchStatusSbchRefunded
	Status           Bch2SbchStatus `gorm:"not null"` //
}

type Sbch2BchRecord struct {
	gorm.Model
	SbchLockTime     uint64         `gorm:"not null"` // got from event
	SbchLockTxHash   string         `gorm:"unique"`   // got from event
	Value            uint64         `gorm:"not null"` // got from txValue, in Sats
	SbchSenderAddr   string         `gorm:"not null"` // got from event
	BchRecipientPkh  string         `gorm:"not null"` // got from event
	HashLock         string         `gorm:"unique"`   // got from event
	TimeLock         uint32         `gorm:"not null"` // got from event, in Seconds
	PenaltyBPS       uint16         `gorm:"not null"` // got from event
	HtlcScriptHash   string         `gorm:"not null"` // calculated by bot
	BchLockTxHash    string         ``                // set when status changed to Sbch2BchStatusBchLocked
	BchUnlockTxHash  string         ``                // set when status changed to Sbch2BchStatusSecretRevealed
	Secret           string         ``                // set when status changed to Sbch2BchStatusSecretRevealed
	SbchUnlockTxHash string         ``                // set when status changed to Sbch2BchStatusSbchUnlocked
	BchRefundTxHash  string         ``                // set when status changed to Sbch2BchStatusBchRefunded
	Status           Sbch2BchStatus `gorm:"not null"` //
}

type DB struct {
	db *gorm.DB
}

func OpenDB(dbFile string) (DB, error) {
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		return DB{}, err
	}
	return DB{db}, nil
}

func (db DB) syncSchemas() error {
	return db.db.AutoMigrate(&Bch2SbchRecord{}, &Sbch2BchRecord{}, &LastHeights{})
}

func (db DB) initLastHeights(lastBchHeight, lastSbchHeight uint64) error {
	result := db.db.Create(&LastHeights{
		LastBchHeight:  lastBchHeight,
		LastSbchHeight: lastSbchHeight,
	})
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to init LastHeights, RowsAffected: %d", result.RowsAffected)
	}
	return nil
}

func (db DB) getLastBchHeight() (uint64, error) {
	heights, err := db.getLastHeights()
	return heights.LastBchHeight, err
}
func (db DB) getLastSbchHeight() (uint64, error) {
	heights, err := db.getLastHeights()
	return heights.LastSbchHeight, err
}
func (db DB) getLastHeights() (*LastHeights, error) {
	heights := &LastHeights{}
	result := db.db.First(heights)
	return heights, result.Error
}

func (db DB) setLastBchHeight(h uint64) error {
	heights, err := db.getLastHeights()
	if err != nil {
		return err
	}
	heights.LastBchHeight = h
	result := db.db.Save(heights)
	return result.Error
}
func (db DB) setLastSbchHeight(h uint64) error {
	heights, err := db.getLastHeights()
	if err != nil {
		return err
	}
	heights.LastSbchHeight = h
	result := db.db.Save(heights)
	return result.Error
}

func (db DB) addBch2SbchRecord(record *Bch2SbchRecord) error {
	if record.BchLockHeight == 0 ||
		record.BchLockTxHash == "" ||
		record.Value == 0 ||
		record.RecipientPkh == "" ||
		record.SenderPkh == "" ||
		record.HashLock == "" ||
		record.TimeLock == 0 ||
		record.SenderEvmAddr == "" ||
		record.HtlcScriptHash == "" {

		return fmt.Errorf("missing required fields")
	}

	result := db.db.Create(record)
	return result.Error
}

func (db DB) addSbch2BchRecord(record *Sbch2BchRecord) error {
	if record.SbchLockTime == 0 ||
		record.SbchLockTxHash == "" ||
		record.Value == 0 ||
		record.SbchSenderAddr == "" ||
		record.BchRecipientPkh == "" ||
		record.HashLock == "" ||
		record.TimeLock == 0 ||
		record.HtlcScriptHash == "" {

		return fmt.Errorf("missing required fields")
	}

	result := db.db.Create(record)
	return result.Error
}

func (db DB) getBch2SbchRecordsByStatus(status Bch2SbchStatus, limit int) (records []*Bch2SbchRecord, err error) {
	result := db.db.Where("status = ?", status).Limit(limit).Find(&records)
	err = result.Error
	return
}

func (db DB) getSbch2BchRecordsByStatus(status Sbch2BchStatus, limit int) (records []*Sbch2BchRecord, err error) {
	result := db.db.Where("status = ?", status).Limit(limit).Find(&records)
	err = result.Error
	return
}

func (db DB) getBch2SbchRecordByHashLock(hashLock string) (record *Bch2SbchRecord, err error) {
	record = &Bch2SbchRecord{}
	result := db.db.Where("hash_lock = ?", hashLock).First(record)
	return record, result.Error
}

func (db DB) getSbch2BchRecordByBchLockTxHash(txHashHex string) (record *Sbch2BchRecord, err error) {
	record = &Sbch2BchRecord{}
	result := db.db.Where("bch_lock_tx_hash = ?", txHashHex).First(record)
	return record, result.Error
}

func (db DB) updateBch2SbchRecord(record *Bch2SbchRecord) error {
	if record.Status == Bch2SbchStatusSbchLocked {
		if record.SbchLockTxHash == "" {
			return fmt.Errorf("SbchLockTxHash is empty")
		}
	} else if record.Status == Bch2SbchStatusSecretRevealed {
		if record.Secret == "" {
			return fmt.Errorf("secret is empty")
		}
		if record.SbchUnlockTxHash == "" {
			return fmt.Errorf("SbchUnlockTxHash is empty")
		}
	} else if record.Status == Bch2SbchStatusBchUnlocked {
		if record.BchUnlockTxHash == "" {
			return fmt.Errorf("BchUnlockTxHash is empty")
		}
	} //else if record.Status == Bch2SbchStatusTooLateToLockSbch {}
	result := db.db.Save(record)
	return result.Error
}

func (db DB) updateSbch2BchRecord(record *Sbch2BchRecord) error {
	if record.Status == Sbch2BchStatusBchLocked {
		if record.BchLockTxHash == "" {
			return fmt.Errorf("BchLockTxHash is empty")
		}
	} else if record.Status == Sbch2BchStatusSecretRevealed {
		if record.Secret == "" {
			return fmt.Errorf("secret is empty")
		}
		if record.BchUnlockTxHash == "" {
			return fmt.Errorf("BchUnlockTxHash is empty")
		}
	} else if record.Status == Sbch2BchStatusSbchUnlocked {
		if record.SbchUnlockTxHash == "" {
			return fmt.Errorf("BchUnlockTxHash is empty")
		}
	} //else if record.Status == Sbch2BchStatusTooLateToLockSbch {}
	result := db.db.Save(record)
	return result.Error
}

func (db DB) GetAllBch2SbchRecords() (records []*Bch2SbchRecord, err error) {
	result := db.db.Find(&records)
	err = result.Error
	return
}
func (db DB) GetAllSbch2BchRecords() (records []*Sbch2BchRecord, err error) {
	result := db.db.Find(&records)
	err = result.Error
	return
}
