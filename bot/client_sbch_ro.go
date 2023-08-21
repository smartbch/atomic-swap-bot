package bot

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type SbchClientRO struct {
	client  *ethclient.Client
	timeout time.Duration
	botAddr common.Address
}

func newSbchClientRO(
	rawUrl string, timeout time.Duration,
	botAddr common.Address,
) (*SbchClientRO, error) {

	client, err := ethclient.Dial(rawUrl)
	if err != nil {
		return nil, err
	}
	return &SbchClientRO{
		client:  client,
		timeout: timeout,
		botAddr: botAddr,
	}, nil
}

func (c *SbchClientRO) getBotBalance() (*big.Int, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.BalanceAt(ctx, c.botAddr, nil)
}
