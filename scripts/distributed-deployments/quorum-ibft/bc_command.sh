#!/bin/bash

export PRIVATE_CONFIG=ignore

/home/ubuntu/quorum/build/bin/geth \
  --datadir data \
  --nodiscover \
  --istanbul.blockperiod 1 \
  --syncmode full \
  --mine \
  --minerthreads 1 \
  --verbosity 5 \
  --networkid 10 \
  --rpc \
  --rpccorsdomain "*" \
  --rpcvhosts "*" \
  --rpcaddr 0.0.0.0 \
  --rpcport 8545 \
  --rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,quorum,raft \
  --emitcheckpoints \
  --port 30305 \
  --ws \
  --wsaddr 0.0.0.0 \
  --wsport 8546 \
  --wsapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,quorum,raft \
  --wsorigins "*"
#  --unlock 0 \
#  --allow-insecure-unlock \
#  --password /home/ubuntu/passwords.txt
