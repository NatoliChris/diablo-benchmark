#!/bin/bash

ADDRS=("0xd2ffab60f7ed4b53f10155b21cee106c429ab506" "0x0ec2aa16564d3ca9df603721815c8c521e858f48" "0x333d24a961070e7002323e1373c77190e1016bb5" "0x2f45bf71dde5d6a1132b737006b14a1573448f16")

openethereum \
  --chain diablo-spec.json \
  -d data \
  --mode active \
  --keys-path data/keys \
  --unlock "${ADDRS[$1]}" \
  --password=/home/ubuntu/passwords.txt \
  --no-discovery \
  --reserved-only \
  --port 30303 \
  --reserved-peers=/home/ubuntu/peers.txt \
  --reseal-on-txs=all \
  --reseal-max-period=2000 \
  --jsonrpc-port 8545 \
  --jsonrpc-interface all \
  --jsonrpc-apis all \
  --jsonrpc-hosts all \
  --jsonrpc-cors all \
  --ws-port 8546 \
  --ws-interface all \
  --ws-apis all \
  --ws-origins all \
  --ws-hosts all \
  --tx-queue-mem-limit 0 \
  --engine-signer="${ADDRS[$1]}" \
  --author="${ADDRS[$1]}"
