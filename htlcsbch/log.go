package htlcsbch

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type LockLog struct {
	LockerAddr      common.Address
	UnlockerAddr    common.Address
	HashLock        common.Hash
	UnlockTime      uint64
	Value           *big.Int
	BchRecipientPkh common.Address
	CreatedTime     uint64
	PenaltyBPS      uint16
	ExpectedPrice   *big.Int
}

type UnlockLog struct {
	TxHash   common.Hash
	HashLock common.Hash
	Secret   common.Hash
}

type RefundLog struct {
	TxHash   common.Hash
	HashLock common.Hash
}

func ParseHtlcLockLog(log types.Log) *LockLog {
	if len(log.Topics) != 3 ||
		log.Topics[0] != LockEventId ||
		len(log.Data) != 32*7 {
		//log.Info("invalid topics or data")
		return nil
	}

	return &LockLog{
		LockerAddr:      common.BytesToAddress(log.Topics[1][12:]),
		UnlockerAddr:    common.BytesToAddress(log.Topics[2][12:]),
		HashLock:        common.BytesToHash(log.Data[0:32]),
		UnlockTime:      bytesToBI(log.Data[32:64]).Uint64(),
		Value:           bytesToBI(log.Data[64:96]),
		BchRecipientPkh: common.BytesToAddress(log.Data[96:128][:20]),
		CreatedTime:     bytesToBI(log.Data[128:160]).Uint64(),
		PenaltyBPS:      uint16(bytesToBI(log.Data[160:192]).Uint64()),
		ExpectedPrice:   bytesToBI(log.Data[192:224]),
	}
}

func ParseHtlcUnlockLog(log types.Log) *UnlockLog {
	if len(log.Topics) != 3 ||
		log.Topics[0] != UnlockEventId {
		return nil
	}
	return &UnlockLog{
		TxHash:   log.TxHash,
		HashLock: log.Topics[1],
		Secret:   log.Topics[2],
	}
}

func ParseHtlcRefundLog(log types.Log) *RefundLog {
	if len(log.Topics) != 2 ||
		log.Topics[0] != RefundEventId {
		return nil
	}
	return &RefundLog{
		TxHash:   log.TxHash,
		HashLock: log.Topics[1],
	}
}

func bytesToBI(bs []byte) *big.Int {
	return big.NewInt(0).SetBytes(bs)
}
