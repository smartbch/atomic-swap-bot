package bot

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/atomic-swap-bot/htlcsbch"
)

const (
	getReceiptRetryCount = 30
	getReceiptWaitTime   = 2 * time.Second
)

const (
	SwapInvalid = iota
	SwapOpen
	SwapClosed
	SwapExpired
)

var _ ISbchClient = (*SbchClient)(nil)

type ISbchClient interface {
	getBlockNumber() (uint64, error)
	getBlockTimeLatest() (uint64, error)
	getTxTime(txHash string) (uint64, error)
	getHtlcLogs(fromBlock, toBlock uint64) ([]types.Log, error)
	lockSbchToHtlc(userEvmAddr common.Address, hashLock common.Hash, timeLock uint32, amt *big.Int) (*common.Hash, error)
	unlockSbchFromHtlc(hashLock common.Hash, secret common.Hash) (*common.Hash, error)
	refundSbchFromHtlc(hashLock common.Hash) (*common.Hash, error)
	getSwapState(hashLock common.Hash) (uint8, error)
}

type SbchClient struct {
	client         *ethclient.Client
	timeout        time.Duration
	privKey        *ecdsa.PrivateKey
	botAddr        common.Address
	htlcAddr       common.Address
	chainId        *big.Int
	gasPrice       *big.Int
	openGasLimit   uint64
	closeGasLimit  uint64
	expireGasLimit uint64
}

func newSbchClient(
	rawUrl string, timeout time.Duration,
	privKey *ecdsa.PrivateKey, botAddr common.Address,
	htlcAddr common.Address,
	gasPrice *big.Int,
	openGasLimit, closeGasLimit, expireGasLimit uint64,
) (*SbchClient, error) {

	client, err := ethclient.Dial(rawUrl)
	if err != nil {
		return nil, err
	}
	return &SbchClient{
		client:         client,
		timeout:        timeout,
		privKey:        privKey,
		botAddr:        botAddr,
		htlcAddr:       htlcAddr,
		gasPrice:       gasPrice,
		openGasLimit:   openGasLimit,
		closeGasLimit:  closeGasLimit,
		expireGasLimit: expireGasLimit,
	}, nil
}

func (c *SbchClient) getBlockNumber() (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.BlockNumber(ctx)
}

func (c *SbchClient) getBlockTimeLatest() (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	header, err := c.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	return header.Time, nil
}

func (c *SbchClient) getTxTime(txHash string) (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()

	tr, err := c.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return 0, err
	}

	header, err := c.client.HeaderByHash(ctx, tr.BlockHash)
	if err != nil {
		return 0, err
	}
	return header.Time, nil
}

func (c *SbchClient) getHtlcLogs(fromBlock, toBlock uint64) ([]types.Log, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(toBlock)),
		Addresses: []common.Address{c.htlcAddr},
	})
}

func (c *SbchClient) getSwapState(hashLock common.Hash) (uint8, error) {
	callData, err := htlcsbch.PackGetSwapState(hashLock)
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		From: c.botAddr,
		To:   &c.htlcAddr,
		Gas:  c.openGasLimit,
		Data: callData,
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return 0, err
	}

	// TODO: unpack result
	return result[0], nil
}

// call open()
func (c *SbchClient) lockSbchToHtlc(
	userEvmAddr common.Address,
	hashLock common.Hash,
	timeLock uint32,
	amt *big.Int,
) (*common.Hash, error) {
	bchAddr := common.Address{}
	log.Info("lock sBCH to HTLC",
		", userEvmAddr: ", userEvmAddr.String(),
		", hashLock: ", hashLock.String(),
		", timeLock: ", timeLock,
		", amt :", amt.String(),
		", bchAddr: ", bchAddr.String())

	data, err := htlcsbch.PackOpen(userEvmAddr, hashLock, timeLock, bchAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to pack calldata: %w", err)
	}
	return c.callHtlc(amt, data, c.openGasLimit)
}

// call close()
func (c *SbchClient) unlockSbchFromHtlc(
	hashLock common.Hash,
	secret common.Hash,
) (*common.Hash, error) {
	log.Info("unlock sBCH from HTLC",
		", hashLock: ", hashLock.String(),
		", secret: ", secret.String())

	data, err := htlcsbch.PackClose(hashLock, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to pack calldata: %w", err)
	}
	return c.callHtlc(big.NewInt(0), data, c.closeGasLimit)
}

// call expire()
func (c *SbchClient) refundSbchFromHtlc(hashLock common.Hash) (*common.Hash, error) {
	log.Info("refund sBCH from HTLC",
		", hashLock: ", hashLock.String())

	data, err := htlcsbch.PackExpire(hashLock)
	if err != nil {
		return nil, fmt.Errorf("failed to pack calldata: %w", err)
	}
	return c.callHtlc(big.NewInt(0), data, c.expireGasLimit)
}

func (c *SbchClient) callHtlc(val *big.Int, data []byte, gasLimit uint64) (*common.Hash, error) {
	chainID, err := c.getChainId()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}
	nonce, err := c.getNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	signer := types.NewEIP155Signer(chainID)
	tx, err := types.SignNewTx(c.privKey, signer, &types.LegacyTx{
		Nonce:    nonce,
		To:       &c.htlcAddr,
		Value:    val,
		Gas:      gasLimit,
		GasPrice: c.gasPrice,
		Data:     data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	err = c.sendTx(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send tx: %w", err)
	}

	txHash := tx.Hash()
	log.Info("tx sent, hash: ", txHash.String())

	receipt, err := c.waitTxReceipt(txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("tx failed! tx hash: %s", txHash.String())
	}

	return &txHash, nil
}

func (c *SbchClient) getChainId() (*big.Int, error) {
	if c.chainId != nil {
		return c.chainId, nil
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.ChainID(ctx)
}

func (c *SbchClient) getNonce() (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.NonceAt(ctx, c.botAddr, nil)
}

func (c *SbchClient) sendTx(tx *types.Transaction) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.SendTransaction(ctx, tx)
}

func (c *SbchClient) getTxReceipt(txHash common.Hash) (*types.Receipt, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.TransactionReceipt(ctx, txHash)
}

func (c *SbchClient) waitTxReceipt(txHash common.Hash) (receipt *types.Receipt, err error) {
	log.Info("get tx receipt, hash: ", txHash.String())
	for i := 0; i < getReceiptRetryCount; i++ {
		receipt, err = c.getTxReceipt(txHash)
		if err == ethereum.NotFound {
			log.Info("tx receipt not ready, wait 2 seconds ...")
			time.Sleep(getReceiptWaitTime)
			continue
		}
		return
	}
	return
}
