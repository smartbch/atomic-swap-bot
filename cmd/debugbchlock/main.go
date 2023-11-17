package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"

	"github.com/smartbch/atomic-swap-bot/bot"
	"github.com/smartbch/atomic-swap-bot/htlcbch"
)

var (
	bchRpcUrl = "https://user:pass@localhost:8333"
	bchAddr   = "bitcoincash:qzcnghz2vp4y245azzj89cxyraxz7mchlvnlz3dvzf"
	bchHeight = int64(819184)
)

func main() {
	flag.StringVar(&bchRpcUrl, "rpc-url", bchRpcUrl, "BCH RPC URL")
	flag.StringVar(&bchRpcUrl, "bot-addr", bchRpcUrl, "BCH Bot Address")
	flag.Int64Var(&bchHeight, "height", bchHeight, "BCH block number")
	flag.Parse()

	decodedBchAddr, err := bchutil.DecodeAddress(bchAddr, &chaincfg.MainNetParams)
	if err != nil {
		panic(fmt.Errorf("failed to decode Bot addr: %w", err))
	}

	bchCli, err := bot.NewBchClient(bchRpcUrl, decodedBchAddr)
	if err != nil {
		panic(fmt.Errorf("faield to create BCH RPC client: %w", err))
	}

	fmt.Println("get BCH block:", bchHeight, "...")
	block, err := bchCli.GetBlock(bchHeight)
	if err != nil {
		panic(fmt.Errorf("faield to get BCH block: %w", err))
	}

	// fmt.Println(block)
	deposits := htlcbch.GetHtlcLocksInfo(block)
	fmt.Println("HTLC deposits: ", len(deposits))
	for _, deposit := range deposits {
		fmt.Println("HTLC deposit: \n", toJSON(deposit))
	}
}

func toJSON(v interface{}) string {
	bs, _ := json.MarshalIndent(v, "", "  ")
	return string(bs)
}
