package bot

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type MockSbchClient struct {
	ts      uint64
	hFrom   uint64
	hTo     uint64
	logs    map[uint64][]types.Log
	txTimes map[string]uint64
}

func newMockSbchClient(hFrom, hTo, ts uint64) *MockSbchClient {
	cli := &MockSbchClient{
		ts:      ts,
		hFrom:   hFrom,
		hTo:     hTo,
		logs:    map[uint64][]types.Log{},
		txTimes: map[string]uint64{},
	}
	return cli
}

func (c *MockSbchClient) getBlockNumber() (uint64, error) {
	return c.hTo, nil
}

func (c *MockSbchClient) getBlockTimeLatest() (uint64, error) {
	return c.ts, nil
}

func (c *MockSbchClient) getTxTime(txHash string) (uint64, error) {
	return c.txTimes[txHash], nil
}

func (c *MockSbchClient) getHtlcLogs(fromBlock, toBlock uint64) ([]types.Log, error) {
	if fromBlock < c.hFrom || toBlock > c.hTo {
		return nil, fmt.Errorf("invalid block range")
	}

	var logs []types.Log
	for i := fromBlock; i <= toBlock; i++ {
		logs = append(logs, c.logs[i]...)
	}
	return logs, nil
}

func (c *MockSbchClient) lockSbchToHtlc(
	userEvmAddr common.Address,
	hashLock common.Hash,
	timeLock uint32,
	amt *big.Int,
) (*common.Hash, error) {
	txHash := common.BytesToHash(reverseBytes(hashLock[:]))
	return &txHash, nil
}

func (c *MockSbchClient) unlockSbchFromHtlc(
	hashLock common.Hash,
	secret common.Hash,
) (*common.Hash, error) {
	txHash := common.BytesToHash(reverseBytes(hashLock[:]))
	return &txHash, nil
}

func (c *MockSbchClient) refundSbchFromHtlc(
	hashLock common.Hash,
) (*common.Hash, error) {
	txHash := common.BytesToHash(reverseBytes(hashLock[:]))
	return &txHash, nil
}
