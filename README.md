# Market Maker Bot

Atomic Swap market maker bot written in Golang.




## Prepare

```bash
git clone https://github.com/smartbch/atomic-swap-bot.git
cd atomic-swap-bot
go test ./...
```



## Start bot on BCH/SBCH testnets

Step1, start your own [BCHN](https://bitcoincashnode.org/en/) testnet node.



Step2, generate a new BCH private key:

```bash
git clone https://github.com/smartbch/atomic-swap-covenants.git
cd atomic-swap-covenant
npm i

ts-node scripts/htlc.ts gen-user
# the output looks like this:
wif : cVPTD8swmUJrYNDkTm1EwhcZ5V5UdsmrTBmYdb9bH6NMxn47H5Yz
addr: bchtest:qqgy70efq403k2mda04ku6dx7r2nfuq4s5u6xh83hw

ts-node scripts/htlc.ts user-info \
  --wif=cVPTD8swmUJrYNDkTm1EwhcZ5V5UdsmrTBmYdb9bH6NMxn47H5Yz
# the output looks like this:
addr : bchtest:qqgy70efq403k2mda04ku6dx7r2nfuq4s5u6xh83hw
pbk  : 020661e43f6a0be81057f6e77157aec13782894dd6faf6fc217bfd9d89cc03e5a7
pkh  : 104f3f29055f1b2b6debeb6e69a6f0d534f01585
```

Send this new address some test tBCH through [faucet](https://tbch.googol.cash/). And import it to you BCHN testnet node:

```bash
curl --user user:pass --data-binary '{"jsonrpc": "1.0", "id":"curltest", "method": "importaddress", "params": ["bchtest:qqgy70efq403k2mda04ku6dx7r2nfuq4s5u6xh83hw", "testbot", true] }' -H 'content-type: text/plain;' http://127.0.0.1:48334/
```



Step3, generate a new SmartBCH private key:

```bash
git clone https://github.com/smartbch/smartbch.git
cd smartbch
go run github.com/smartbch/smartbch/cmd/smartbchd gen-test-keys -n 1 --show-address
# the output looks like this:
3f122c95922493442b9a358d851ab42771efdc73f6e7fd2f6af8091f2cfca491 0x3Aad4164ee396E8d4dAa36b97c60A734D49CC946
```

Send this new address some test sBCH through [faucet](http://13.214.162.63:8080/faucet).



Step4, register the bot:

```bash
git clone https://github.com/smartbch/atomic-swap-contracts.git
cd atomic-swap-contracts
npm i

KEY1=3f122c95922493442b9a358d851ab42771efdc73f6e7fd2f6af8091f2cfca491 \
HARDHAT_NETWORK=sbch_testnet node ./scripts/htlc.js register-bot \
	--htlc-addr=0x3246D84c930794cDFAABBab954BAc58A7c08b4cd \
	--intro=TestBot \
	--pkh=0x104f3f29055f1b2b6debeb6e69a6f0d534f01585 \
	--bch-lock-time=6 \
	--sbch-lock-time=3600 \
	--penalty-bps=500 \
	--fee-bps=100 \
	--min-swap-amt=0.01 \
	--max-swap-amt=10.0 \
	--status-checker=0x3Aad4164ee396E8d4dAa36b97c60A734D49CC946
```

Note the `--htlc-addr` option. Follow [this doc](https://github.com/smartbch/atomic-swap-contracts/blob/main/README.md) to deploy you own HTLC smart contract on SmartBCH testnet.



Step5, start the bot:

```bash
cd atomic-swap-bot

go run github.com/smartbch/atomic-swap-bot/cmd/asbot \
	--db-file=bot.db \
	--bch-key=cVPTD8swmUJrYNDkTm1EwhcZ5V5UdsmrTBmYdb9bH6NMxn47H5Yz \
	--bch-rpc-url=http://user:pass@127.0.0.1:48334 \
	--sbch-rpc-url=http://127.0.0.1:8545 \
	--sbch-key=3f122c95922493442b9a358d851ab42771efdc73f6e7fd2f6af8091f2cfca491 \
	--sbch-htlc-addr=0x3246D84c930794cDFAABBab954BAc58A7c08b4cd \
	--sbch-gas-price=1.05 \
	--bch-timelock=6 \
	--sbch-timelock=3600 \
	--penalty-ratio=500 \
	--fee-ratio=100 \
	--min-swap-val=0.01 \
	--max-swap-val=10.0 \
	--bch-confirmations=0 \
	--bch-send-fee-rate=2 \
	--bch-receive-fee-rate=2 \
	--bch-refund-fee-rate=2 \
	--sbch-open-gas=500000 \
	--sbch-close-gas=500000 \
	--sbch-expire-gas=500000
```



Or start bot in enclave using [EGo](https://www.edgeless.systems/products/ego/):

```bash
git clone https://github.com/smartbch/atomic-swap-bot.git
cd atomic-swap-bot

ego-go build github.com/smartbch/atomic-swap-bot/cmd/asbot
ego sign asbot
mkdir data
ego run asbot \
	--db-file=bot.db \
	--bch-rpc-url=http://user:pass@127.0.0.1:48334 \
	--sbch-rpc-url=http://127.0.0.1:8545 \
	--sbch-htlc-addr=0x3246D84c930794cDFAABBab954BAc58A7c08b4cd \
	--sbch-gas-price=1.05 \
	--bch-timelock=6 \
	--sbch-timelock=3600 \
	--penalty-ratio=500 \
	--fee-ratio=100 \
	--min-swap-val=0.01 \
	--max-swap-val=10.0 \
	--bch-confirmations=0 \
	--bch-send-fee-rate=2 \
	--bch-receive-fee-rate=2 \
	--bch-refund-fee-rate=2 \
	--sbch-open-gas=500000 \
	--sbch-close-gas=500000 \
	--sbch-expire-gas=500000
```

The above cmd prints something like this and wait inputs:
```
EGo v1.3.0 (360a6a40836461465fdbd0742dfb0f980b68c638)
[erthost] loading enclave ...
[erthost] entering enclave ...
[ego] starting application ...
The ecies pubkey: 03052743d278846f90cedb64282a3ea3db20a8414b627e4aff3dc5408110073eed
Enter the encrypted BCH WIF (ASIC): 
```

You can encrypt your BCH/sBCH keys using this cmd (in another shell window):

```bash
go run github.com/smartbch/atomic-swap-bot/cmd/encrypt
```

Then feed the encrypted keys into `ego run` cmd and the bot will be started.



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

