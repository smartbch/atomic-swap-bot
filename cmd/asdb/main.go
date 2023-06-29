package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"

	"github.com/smartbch/atomic-swap-bot/bot"
)

func main() {
	dbFile := "./bot.db"
	format := "text"

	if len(os.Args) > 1 {
		dbFile = os.Args[1]
	}
	if len(os.Args) > 2 {
		if os.Args[2] == "table" {
			format = "table"
		}
	}

	db, err := bot.OpenDB(dbFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	allBch2Sbch, err := db.GetAllBch2SbchRecords()
	if err != nil {
		fmt.Println(err)
		return
	}
	allSbch2Bch, err := db.GetAllSbch2BchRecords()
	if err != nil {
		fmt.Println(err)
		return
	}

	if format == "table" {
		printBch2SbchRecordsTable(allBch2Sbch)
		printSbch2BchRecordsTable(allSbch2Bch)
	} else {
		printBch2SbchRecords(allBch2Sbch)
		printSbch2BchRecords(allSbch2Bch)
	}
}

func printBch2SbchRecords(records []*bot.Bch2SbchRecord) {
	fmt.Println("BCH2SBCH records:")
	j, _ := json.MarshalIndent(records, "", "  ")
	fmt.Println(string(j))
}

func printSbch2BchRecords(records []*bot.Sbch2BchRecord) {
	fmt.Println("SBCH2BCH records:")
	j, _ := json.MarshalIndent(records, "", "  ")
	fmt.Println(string(j))
}

func printBch2SbchRecordsTable(records []*bot.Bch2SbchRecord) {
	fmt.Println("BCH2SBCH records:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"ID",
		"BchLockH",
		"BchLockTx",
		"Value",
		"Recipient",
		"Sender",
		"HashLock",
		"TimeLock",
		"Penalty",
		"SenderEvmAddr",
		"HtlcHash",
		"SbchLockTxTime",
		"SbchLockTxHash",
		"SbchUnlockTx",
		"Secret",
		"BchUnlockTx",
		"SbchRefundTx",
		"Status",
	})
	for _, record := range records {
		table.Append([]string{
			intToStr(record.ID),
			intToStr(record.BchLockHeight),
			subStr12(record.BchLockTxHash),
			intToStr(record.Value),
			subStr12(record.RecipientPkh),
			subStr12(record.SenderPkh),
			subStr12(record.HashLock),
			intToStr(record.TimeLock),
			intToStr(record.PenaltyBPS),
			subStr12(record.SenderEvmAddr),
			subStr12(record.HtlcScriptHash),
			intToStr(record.SbchLockTxTime),
			subStr12(record.SbchLockTxHash),
			subStr12(record.SbchUnlockTxHash),
			subStr12(record.Secret),
			subStr12(record.BchUnlockTxHash),
			subStr12(record.SbchRefundTxHash),
			intToStr(record.Status),
		})
	}
	table.Render() // Send output
}

func printSbch2BchRecordsTable(records []*bot.Sbch2BchRecord) {
	fmt.Println("SBCH2BCH records:")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"ID",
		"SbchLockTime",
		"SbchLockTx",
		"Value",
		"SbchSender",
		"BchRecipient",
		"HashLock",
		"TimeLock",
		"Penalty",
		"HtlcHash",
		"BchLockBlock",
		"BchLockTx",
		"BchUnlockTx",
		"Secret",
		"SbchUnlockTx",
		"BchRefundTx",
		"Status",
	})
	for _, record := range records {
		table.Append([]string{
			intToStr(record.ID),
			intToStr(record.SbchLockTime),
			subStr12(record.SbchLockTxHash),
			intToStr(record.Value),
			subStr12(record.SbchSenderAddr),
			subStr12(record.BchRecipientPkh),
			subStr12(record.HashLock),
			intToStr(record.TimeLock),
			intToStr(record.PenaltyBPS),
			subStr12(record.HtlcScriptHash),
			intToStr(record.BchLockBlockNum),
			subStr12(record.BchLockTxHash),
			subStr12(record.BchUnlockTxHash),
			subStr12(record.Secret),
			subStr12(record.SbchUnlockTxHash),
			subStr12(record.BchRefundTxHash),
			intToStr(record.Status),
		})
	}
	table.Render() // Send output
}

func intToStr(n any) string {
	return intToStr(n)
}
func subStr12(s string) string {
	if len(s) < 12 {
		return s
	}
	return s[:12]
}
