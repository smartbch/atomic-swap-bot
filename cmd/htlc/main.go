package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	"github.com/urfave/cli/v2"

	"github.com/smartbch/atomic-swap-bot/bot"
	"github.com/smartbch/atomic-swap-bot/htlcbch"
)

const (
	flagNameRpcUrl       = "rpc-url"
	flagNameWIF          = "wif"
	flagNameFromAddr     = "from-addr"
	flagNameToAddr       = "to-addr"
	flagNameSecret       = "secret"
	flagNameExpiration   = "expiration"
	flagNamePenaltyBPS   = "penalty-bps"
	flagNameUTXO         = "utxo"
	flagNameAmt          = "amt"
	flagNameMinerFeeRate = "miner-fee-rate"
	flagNameDryRun       = "dry-run"
)

var (
	flagRpcUrl       = &cli.StringFlag{Name: flagNameRpcUrl, Required: false}
	flagWIF          = &cli.StringFlag{Name: flagNameWIF, Required: true}
	flagFromAddr     = &cli.StringFlag{Name: flagNameFromAddr, Required: false}
	flagToAddr       = &cli.StringFlag{Name: flagNameToAddr, Required: false}
	flagSecret       = &cli.StringFlag{Name: flagNameSecret, Required: false, DefaultText: "123"}
	flagExpiration   = &cli.Uint64Flag{Name: flagNameExpiration, Required: false, DefaultText: "36"}
	flagPenaltyBPS   = &cli.Uint64Flag{Name: flagNamePenaltyBPS, Required: false, DefaultText: "500"}
	flagUTXO         = &cli.StringFlag{Name: flagNameUTXO, Required: true, Usage: "txid:vout:val"}
	flagAmt          = &cli.Uint64Flag{Name: flagNameAmt, Required: true}
	flagMinerFeeRate = &cli.Uint64Flag{Name: flagNameMinerFeeRate, Required: false, DefaultText: "2"}
	flagDryRun       = &cli.BoolFlag{Name: flagNameDryRun, Required: false, DefaultText: "true"}
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			cmdLock(),
			cmdUnlock(),
			cmdRefund(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}

func cmdLock() *cli.Command {
	return &cli.Command{
		Name: "lock",
		Flags: []cli.Flag{
			flagWIF, flagToAddr, flagSecret, flagExpiration, flagPenaltyBPS,
			flagUTXO, flagAmt, flagMinerFeeRate, flagDryRun, flagRpcUrl,
		},
		Action: func(ctx *cli.Context) error {
			wif, pkh, addr, err := decodeWIF(ctx.String(flagNameWIF))
			if err != nil {
				return err
			}
			_, toPkh, err := decodeAddr(ctx.String(flagNameToAddr))
			if err != nil {
				return err
			}
			_, hashLock := secretToHashLock(ctx.String(flagNameSecret))

			c, err := htlcbch.NewTestnet3Covenant(
				pkh, toPkh, hashLock,
				uint16(ctx.Uint64(flagNameExpiration)),
				uint16(ctx.Uint64(flagNamePenaltyBPS)),
			)
			if err != nil {
				return err
			}

			cP2SH, err := c.GetRedeemScriptHash()
			if err != nil {
				return err
			}

			txid, vout, val, err := parseUTXO(ctx.String(flagNameUTXO))
			if err != nil {
				return err
			}

			//amt := int64(math.Round(ctx.Float64(flagNameAmt) * 1e8))
			amt := int64(ctx.Uint64(flagNameAmt))
			minerFeeRate := ctx.Uint64(flagNameMinerFeeRate)

			fmt.Println("from pkh :", hex.EncodeToString(pkh))
			fmt.Println("to pkh   :", hex.EncodeToString(toPkh))
			fmt.Println("hash lock:", hex.EncodeToString(hashLock))
			fmt.Println("htlc p2sh:", hex.EncodeToString(cP2SH))

			inputs := []htlcbch.InputInfo{{TxID: txid, Vout: uint32(vout), Amount: int64(val)}}
			tx, err := c.MakeLockTx(wif.PrivKey, inputs, amt, minerFeeRate)
			if err != nil {
				return err
			}

			fmt.Println("locking tx:", htlcbch.MsgTxToHex(tx))

			rpcRrl := ctx.String(flagNameRpcUrl)
			if ctx.Bool(flagNameDryRun) || rpcRrl == "" {
				return nil
			}

			bchCli, err := bot.NewBchClient(rpcRrl, addr)
			txHash, err := bchCli.SendTx(tx)
			if err != nil {
				return err
			}

			fmt.Println("tx hash:", txHash.String())
			return nil
		},
	}
}

func cmdUnlock() *cli.Command {
	return &cli.Command{
		Name: "unlock",
		Flags: []cli.Flag{
			flagWIF, flagFromAddr, flagSecret, flagExpiration, flagPenaltyBPS,
			flagUTXO, flagMinerFeeRate, flagDryRun, flagRpcUrl,
		},
		Action: func(ctx *cli.Context) error {
			_, pkh, addr, err := decodeWIF(ctx.String(flagNameWIF))
			if err != nil {
				return err
			}
			_, fromPkh, err := decodeAddr(ctx.String(flagNameFromAddr))
			if err != nil {
				return err
			}
			secret, hashLock := secretToHashLock(ctx.String(flagNameSecret))

			c, err := htlcbch.NewTestnet3Covenant(
				fromPkh, pkh, hashLock,
				uint16(ctx.Uint64(flagNameExpiration)),
				uint16(ctx.Uint64(flagNamePenaltyBPS)),
			)
			if err != nil {
				return err
			}

			cP2SH, err := c.GetRedeemScriptHash()
			if err != nil {
				return err
			}

			txid, vout, val, err := parseUTXO(ctx.String(flagNameUTXO))
			if err != nil {
				return err
			}

			//amt := int64(math.Round(ctx.Float64(flagNameAmt) * 1e8))
			minerFeeRate := ctx.Uint64(flagNameMinerFeeRate)

			fmt.Println("from pkh :", hex.EncodeToString(fromPkh))
			fmt.Println("to pkh   :", hex.EncodeToString(pkh))
			fmt.Println("hash lock:", hex.EncodeToString(hashLock))
			fmt.Println("htlc p2sh:", hex.EncodeToString(cP2SH))

			tx, err := c.MakeReceiveTx(txid, uint32(vout), int64(val), minerFeeRate, secret)
			if err != nil {
				return err
			}

			fmt.Println("unlocking tx:", htlcbch.MsgTxToHex(tx))

			rpcRrl := ctx.String(flagNameRpcUrl)
			if ctx.Bool(flagNameDryRun) || rpcRrl == "" {
				return nil
			}

			bchCli, err := bot.NewBchClient(rpcRrl, addr)
			txHash, err := bchCli.SendTx(tx)
			if err != nil {
				return err
			}

			fmt.Println("tx hash:", txHash.String())
			return nil
		},
	}
}

func cmdRefund() *cli.Command {
	return &cli.Command{
		Name: "refund",
		Flags: []cli.Flag{
			flagWIF, flagToAddr, flagSecret, flagExpiration, flagPenaltyBPS,
			flagUTXO, flagMinerFeeRate, flagDryRun, flagRpcUrl,
		},
		Action: func(ctx *cli.Context) error {
			_, pkh, addr, err := decodeWIF(ctx.String(flagNameWIF))
			if err != nil {
				return err
			}
			_, toPkh, err := decodeAddr(ctx.String(flagNameToAddr))
			if err != nil {
				return err
			}
			_, hashLock := secretToHashLock(ctx.String(flagNameSecret))

			c, err := htlcbch.NewTestnet3Covenant(
				pkh, toPkh, hashLock,
				uint16(ctx.Uint64(flagNameExpiration)),
				uint16(ctx.Uint64(flagNamePenaltyBPS)),
			)
			if err != nil {
				return err
			}

			cP2SH, err := c.GetRedeemScriptHash()
			if err != nil {
				return err
			}

			txid, vout, val, err := parseUTXO(ctx.String(flagNameUTXO))
			if err != nil {
				return err
			}

			//amt := int64(math.Round(ctx.Float64(flagNameAmt) * 1e8))
			minerFeeRate := ctx.Uint64(flagNameMinerFeeRate)

			fmt.Println("from pkh :", hex.EncodeToString(pkh))
			fmt.Println("to pkh   :", hex.EncodeToString(toPkh))
			fmt.Println("hash lock:", hex.EncodeToString(hashLock))
			fmt.Println("htlc p2sh:", hex.EncodeToString(cP2SH))

			tx, err := c.MakeRefundTx(txid, uint32(vout), int64(val), minerFeeRate)
			if err != nil {
				return err
			}

			fmt.Println("unlocking tx:", htlcbch.MsgTxToHex(tx))

			rpcRrl := ctx.String(flagNameRpcUrl)
			if ctx.Bool(flagNameDryRun) || rpcRrl == "" {
				return nil
			}

			bchCli, err := bot.NewBchClient(rpcRrl, addr)
			txHash, err := bchCli.SendTx(tx)
			if err != nil {
				return err
			}

			fmt.Println("tx hash:", txHash.String())
			return nil
		},
	}
}

func decodeWIF(wifStr string) (wif *bchutil.WIF, pkh []byte, addr *bchutil.AddressPubKeyHash, err error) {
	wif, err = bchutil.DecodeWIF(wifStr)
	if err != nil {
		return
	}

	wif.CompressPubKey = true
	pkh = bchutil.Hash160(wif.SerializePubKey())
	addr, err = bchutil.NewAddressPubKeyHash(pkh, &chaincfg.TestNet3Params)
	return
}

func decodeAddr(addrStr string) (bchutil.Address, []byte, error) {
	addr, err := bchutil.DecodeAddress(addrStr, &chaincfg.TestNet3Params)
	if err != nil {
		return nil, nil, err
	}
	addrPkh, ok := addr.(*bchutil.AddressPubKeyHash)
	if !ok {
		return nil, nil, fmt.Errorf("not P2PKH addr: %s", addrStr)
	}
	pkh := addrPkh.Hash160()
	return addr, (*pkh)[:], nil
}

func secretToHashLock(secret string) ([]byte, []byte) {
	var secret32 [32]byte
	copy(secret32[:], secret)
	hashLock := sha256.Sum256(secret32[:])
	return secret32[:], hashLock[:]
}

func parseUTXO(utxo string) (txid []byte, vout uint64, val uint64, err error) {
	ss := strings.Split(utxo, ":")
	if len(ss) != 3 {
		err = fmt.Errorf("invalid utxo")
		return
	}
	txid = gethcmn.FromHex(ss[0])
	vout, err = strconv.ParseUint(ss[1], 10, 64)
	if err != nil {
		return
	}
	val, err = strconv.ParseUint(ss[2], 10, 64)
	return
}
