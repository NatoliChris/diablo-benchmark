#!/bin/bash

ACCS=("0xD45742eF839e1891A00cc91F4eC6BcB250A436ca" "0x07E86a424ABFE327b949Eee191aEb7062A407421" "0xCEBD31C47333747D96de8E8A93DBcB5954354AF5" "0xa9EFBeA3eEdFa675b6E2F1547eCE39eB5820B61c")

geth \
  --datadir data \
  --nodiscover \
  --syncmode full \
  --mine \
  --minerthreads 1 \
  --verbosity 5 \
  --networkid 21 \
  --http \
  --http.corsdomain "*" \
  --http.vhosts "*" \
  --http.addr 0.0.0.0 \
  --http.port 8545 \
  --rpcapi admin,eth,debug,miner,net,txpool,personal,web3\
  --port 30303 \
  --ws \
  --ws.addr 0.0.0.0 \
  --ws.port 8546 \
  --ws.api admin,eth,debug,miner,net,txpool,personal,web3\
  --ws.origins "*" \
  --unlock ${ACCS[$1]} \
  --allow-insecure-unlock \
  --password /home/ubuntu/passwords.txt
