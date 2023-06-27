# Market Maker Bot

Market maker bot written in Golang.



## bot cmd

TODO



## htlc cmd

You can user `htlc` cmd to test BCH HTLC covenant using Golang on BCH Testnet3.

Locking example:

```bash
go run github.com/smartbch/atomic-swap/market-maker-bot/cmd/htlc lock \
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
go run github.com/smartbch/atomic-swap/market-maker-bot/cmd/htlc unlock \
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
go run github.com/smartbch/atomic-swap/market-maker-bot/cmd/htlc refund \
	--wif=cUR6VdPBVn3VQWzJZ9Pr7owhWg3u4Tzoy1w5rstrNKouycpDLUdb \
	--to-addr=bchtest:qpxsyl7aqkznqgnyjg476k9c4pxnsamvevm9fv3cfh \
	--secret=123 \
	--expiration=36 \
	--penalty-bps=500 \
	--utxo=bba060e1756a596ca99c45b4c5628a52eb7c28e53fe6f7009512a65d41c3fbf5:2:70000 \
	--miner-fee-rate=2 \
	--dry-run=true
```

