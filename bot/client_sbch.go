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
	"github.com/ethereum/go-ethereum/crypto"
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
	SwapLocked
	SwapUnlocked
	SwapRefunded
)

var _ ISbchClient = (*SbchClient)(nil)

type ISbchClient interface {
	getBlockNumber() (uint64, error)
	getBlockTimeLatest() (uint64, error)
	getTxTime(txHash common.Hash) (uint64, error)
	getHtlcLogs(fromBlock, toBlock uint64) ([]types.Log, error)
	lockSbchToHtlc(userEvmAddr common.Address, hashLock common.Hash, timeLock uint32, amt *big.Int) (*common.Hash, error)
	unlockSbchFromHtlc(senderAddr common.Address, hashLock common.Hash, secret common.Hash) (*common.Hash, error)
	refundSbchFromHtlc(senderAddr common.Address, hashLock common.Hash) (*common.Hash, error)
	getSwapState(senderAddr common.Address, hashLock common.Hash) (uint8, error)
	getMarketMakerInfo(addr common.Address) (*htlcsbch.MarketMakerInfo, error)
}

type SbchClient struct {
	client   *ethclient.Client
	timeout  time.Duration
	privKey  *ecdsa.PrivateKey
	botAddr  common.Address
	htlcAddr common.Address
	chainId  *big.Int
	gasPrice *big.Int
}

func newSbchClient(
	rawUrl string, timeout time.Duration,
	privKey *ecdsa.PrivateKey,
	htlcAddr common.Address,
	gasPrice *big.Int,
) (*SbchClient, error) {

	client, err := ethclient.Dial(rawUrl)
	if err != nil {
		return nil, err
	}

	botAddr := crypto.PubkeyToAddress(privKey.PublicKey)
	return &SbchClient{
		client:   client,
		timeout:  timeout,
		privKey:  privKey,
		botAddr:  botAddr,
		htlcAddr: htlcAddr,
		gasPrice: gasPrice,
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

func (c *SbchClient) getTxTime(txHash common.Hash) (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()

	tr, err := c.client.TransactionReceipt(ctx, txHash)
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

func (c *SbchClient) getSwapState(senderAddr common.Address, hashLock common.Hash) (uint8, error) {
	callData, err := htlcsbch.PackGetSwapState(senderAddr, hashLock)
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{
		From: c.botAddr,
		To:   &c.htlcAddr,
		Gas:  500_000,
		Data: callData,
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return 0, err
	}

	return htlcsbch.UnpackGetSwapState(result)
}

func (c *SbchClient) getMarketMakerInfo(addr common.Address) (*htlcsbch.MarketMakerInfo, error) {
	callData, err := htlcsbch.PackGetMarketMaker(addr)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		From: c.botAddr,
		To:   &c.htlcAddr,
		Gas:  500_000,
		Data: callData,
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	result, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	return htlcsbch.UnpackGetMarketMaker(result)
}

// call lock()
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

	data, err := htlcsbch.PackLock(userEvmAddr, hashLock, timeLock, bchAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to pack calldata: %w", err)
	}
	return c.callHtlc(amt, data)
}

// call unlock()
func (c *SbchClient) unlockSbchFromHtlc(
	senderAddr common.Address,
	hashLock common.Hash,
	secret common.Hash,
) (*common.Hash, error) {
	log.Info("unlock sBCH from HTLC",
		", hashLock: ", hashLock.String(),
		", secret: ", secret.String())

	data, err := htlcsbch.PackUnlock(senderAddr, hashLock, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to pack calldata: %w", err)
	}
	return c.callHtlc(big.NewInt(0), data)
}

// call refund()
func (c *SbchClient) refundSbchFromHtlc(
	senderAddr common.Address,
	hashLock common.Hash,
) (*common.Hash, error) {
	log.Info("refund sBCH from HTLC",
		", hashLock: ", hashLock.String())

	data, err := htlcsbch.PackRefund(senderAddr, hashLock)
	if err != nil {
		return nil, fmt.Errorf("failed to pack calldata: %w", err)
	}
	return c.callHtlc(big.NewInt(0), data)
}

func (c *SbchClient) callHtlc(val *big.Int, data []byte) (*common.Hash, error) {
	chainID, err := c.getChainId()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	nonce, err := c.getNonce(c.botAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasLimit, err := c.estimateGas(ethereum.CallMsg{
		From:  c.botAddr,
		To:    &c.htlcAddr,
		Value: val,
		Data:  data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}

	gasLimit = gasLimit * 120 / 100
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
	chainId, err := c.client.ChainID(ctx)
	if err == nil {
		c.chainId = chainId
	}
	return chainId, err
}

func (c *SbchClient) getNonce(addr common.Address) (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.NonceAt(ctx, addr, nil)
}

func (c *SbchClient) estimateGas(msg ethereum.CallMsg) (uint64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), c.timeout)
	defer cancelFn()
	return c.client.EstimateGas(ctx, msg)
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
