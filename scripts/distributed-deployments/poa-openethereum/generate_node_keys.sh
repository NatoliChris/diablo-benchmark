#!/bin/bash

/home/natc/dev/openethereum/target/release/openethereum account new \
  --chain diablo-spec.json \
  -d node_directories/$1 \
