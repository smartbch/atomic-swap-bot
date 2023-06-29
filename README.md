# Market Maker Bot

Market maker bot written in Golang.


## Prepare

```bash

git clone https://github.com/smartbch/atomic-swap-bot.git
cd atomic-swap-bot
```


## bot cmd

Start bot on BCH testnet:

```bash
go run github.com/smartbch/atomic-swap-bot/cmd/asbot \
	--db-file=bot.db \
	--bch-key= \
	--sbch-key= \
	--bch-rpc-url= \
	--sbch-rpc-url= \
	--sbch-htlc-addr= \
	--sbch-gas-price= \
	--bch-timelock= \
	--sbch-timelock= \
	--penalty-ratio= \
	--fee-ratio= \
	--min-swap-val= \
	--max-swap-val= \
	--confirmations= \
	--bch-send-fee-rate= \
	--bch-receive-fee-rate= \
	--bch-refund-fee-rate= \
	--sbch-open-gas= \
	--sbch-close-gas= \
	--sbch-expire-gas=

```



## htlc cmd

You can use `htlc` cmd to test BCH HTLC covenant using Golang on BCH testnets.

Locking example:

```bash
go run github.com/smartbch/atomic-swap-bot/cmd/htlc lock \
	--wif=cUR6VdPBVn3VQWzJZ9Pr7owhWg3u4Tzoy1w5rstrNKouycpDLUdb \
	--to-addr=bchtest:qpxsyl7aqkznqgnyjg476k9c4pxnsamvevm9fv3cfh \
	--secret=123 \
	--expiration=36 \
	--penalty-bps=500 \
	--utxo=bba060e1756a596ca99c45b4c5628a52eb7c28e53fe6f7009512a65d41c3fbf5:2:70000 \
	--amt=5000 \
	--miner-fee-rate=2 \
	--dry-run=true
```

Unlocking example:

```bash
go run github.com/smartbch/atomic-swap-bot/cmd/htlc unlock \
	--wif=cSuHicBzB3NHUMQgNUniGXvXvLwZYArg2AUt4G6NgEqsc1CZ2yRd \
	--from-addr=bchtest:qzj8ze00ga7fnffum6uydfmg0grf6l0j0s5t4h0xhn \
	--secret=123 \
	--expiration=36 \
	--penalty-bps=500 \
	--utxo=bba060e1756a596ca99c45b4c5628a52eb7c28e53fe6f7009512a65d41c3fbf5:2:70000 \
	--miner-fee-rate=2 \
	--dry-run=true
```

Refunding example:

```bash
go run github.com/smartbch/atomic-swap-bot/cmd/htlc refund \
	--wif=cUR6VdPBVn3VQWzJZ9Pr7owhWg3u4Tzoy1w5rstrNKouycpDLUdb \
	--to-addr=bchtest:qpxsyl7aqkznqgnyjg476k9c4pxnsamvevm9fv3cfh \
	--secret=123 \
	--expiration=36 \
	--penalty-bps=500 \
	--utxo=bba060e1756a596ca99c45b4c5628a52eb7c28e53fe6f7009512a65d41c3fbf5:2:70000 \
	--miner-fee-rate=2 \
	--dry-run=true
```

