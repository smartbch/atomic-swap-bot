package htlcsbch

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type OpenLog struct {
	LockerAddr      common.Address
	UnlockerAddr    common.Address
	HashLock        common.Hash
	UnlockTime      uint64
	Value           *big.Int
	BchRecipientPkh common.Address
	CreatedTime     uint64
	PenaltyBPS      uint16
}

type CloseLog struct {
	HashLock common.Hash
	Secret   common.Hash
}

type ExpireLog struct {
	HashLock common.Hash
}

func ParseHtlcOpenLog(log types.Log) *OpenLog {
	if len(log.Topics) != 3 ||
		log.Topics[0] != OpenEventId ||
		len(log.Data) != 32*6 {
		//log.Info("invalid topics or data")
		return nil
	}

	return &OpenLog{
		LockerAddr:      common.BytesToAddress(log.Topics[1][12:]),
		UnlockerAddr:    common.BytesToAddress(log.Topics[2][12:]),
		HashLock:        common.BytesToHash(log.Data[0:32]),
		UnlockTime:      bytesToBI(log.Data[32:64]).Uint64(),
		Value:           bytesToBI(log.Data[64:96]),
		BchRecipientPkh: common.BytesToAddress(log.Data[96:128][:20]),
		CreatedTime:     bytesToBI(log.Data[128:160]).Uint64(),
		PenaltyBPS:      uint16(bytesToBI(log.Data[160:192]).Uint64()),
	}
}

func ParseHtlcCloseLog(log types.Log) *CloseLog {
	if len(log.Topics) != 3 ||
		log.Topics[0] != CloseEventId {
		return nil
	}
	return &CloseLog{
		HashLock: log.Topics[1],
		Secret:   log.Topics[2],
	}
}

func ParseHtlcExpireLog(log types.Log) *ExpireLog {
	if len(log.Topics) != 2 ||
		log.Topics[0] != ExpireEventId {
		return nil
	}
	return &ExpireLog{
		HashLock: log.Topics[1],
	}
}

func bytesToBI(bs []byte) *big.Int {
	return big.NewInt(0).SetBytes(bs)
}
