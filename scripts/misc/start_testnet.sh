#!/bin/bash

# This script uses ganache-cli for testing purposes

ganache-cli --mnemonic "nice charge tank ivory warfare spin deposit ecology beauty unusual comic melt" \
  -h "0.0.0.0" \
  --defaultBalanceEther 10000000000000000 \
  --acctKeys "accounts" \
  -a 2000
  # -b 5
  # --verbose
