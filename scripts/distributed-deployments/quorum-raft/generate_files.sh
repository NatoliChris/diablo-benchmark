#!/bin/bash

# ------------------------------------------------------------------------------
#                                 Global Vars
# ------------------------------------------------------------------------------

RED="\033[31m"
BLUE="\033[34m"
GREEN="\033[32m"
RESET="\033[0m"

GOK="[${GREEN}OK${RESET}]"
RERR="[${RED}ERR${RESET}]"

ACCOUNT_BASE="../../artifacts/ethereum_accounts"

# ------------------------------------------------------------------------------
#                                 Template
# ------------------------------------------------------------------------------

rm -f generated_genesis.json

read -d '' template << _EOF_
{
  "alloc": {
    "0xebf4298e6f75fbc97d16fb5f0dfe24727dd56ffb": {
      "balance": "10000100000000000000"
    },
    "0xed9d02e382b34818e88b88a309c7fe71e65f419d": {
      "balance": "10000100000000000000"
    },
    "0xca843569e3427144cead5e4d5999a3d0ccf92b8e": {
      "balance": "10000100000000000000"
    },
    "0x0fbdc686b912d7722dc86510934589e0aaf3b55a": {
      "balance": "10000100000000000000"
    },
    "0x9186eb3d20cbd1f5f992a950d808c4495153abd5": {
      "balance": "10000100000000000000"
    },
    "0x0638e1574728b6d862dd5d3a3e0942c3be47d996": {
      "balance": "10000100000000000000"
    },
    %s
  },
  "coinbase": "0x0000000000000000000000000000000000000000",
  "config": {
    "homesteadBlock": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "chainId": 10,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip158Block": 0,
    "isQuorum": true,
    "maxCodeSizeConfig" : [
      {
        "block" : 0,
        "size" : 35
      }
    ]
  },
  "difficulty": "0x0",
  "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": "0x00000000",
  "mixhash": "0x00000000000000000000000000000000000000647572616c65787365646c6578",
  "nonce": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp": "0x00"
}
_EOF_

# ------------------------------------------------------------------------------
#                                   HELPERS
# ------------------------------------------------------------------------------

function join_arr { local IFS="$1"; shift; echo "$*"; }

# ------------------------------------------------------------------------------
#                                   MAIN
# ------------------------------------------------------------------------------

echo -e "Setting up quorum templates."

if [[ $# -lt 1 ]]; then
  echo -e "${RERR} Invalid Arguments"
  echo -e "Usage: ./$0 number_of_accounts"
  exit 1
fi

accs=$(head -n 5000 $ACCOUNT_BASE)

genesislines=()
accountslines=()

an=0
while IFS= read -r accountrow; do
  addr=$(echo "$accountrow" | cut -d ':' -f 1 | xargs)
  priv=$(echo "$accountrow" | cut -d ':' -f 2 | xargs)


    # Add to the accounts file
    accountslines+=("{\"address\":\"${addr}\",\"private\":\"${priv}\"}")
    # Add to genesis
    genesislines+=("\"${addr}\": {\"balance\":\"10000100000000000000\"}")
    echo -ne "\\r Account: ${an} ${addr} ${priv}"
    an=$((an + 1))
done <<< "$accs"
echo ""

if [[ "${#accountslines[@]}" -ge 5000 ]]; then
	# Print to the files
	echo "[" > diablo_keys.json
	echo $(join_arr , "${accountslines[@]}") >> diablo_keys.json
	echo "]" >> diablo_keys.json
	## Genesis
	joinedgenesis=$(join_arr , "${genesislines[@]}")

	echo -e "${GOK} DONE"
	echo -e "${GOK} Genesis DONE"
	printf "$template" "$joinedgenesis" >> generated_genesis.json
else
	echo -e "${RERR} Failed to read accounts from file."
	exit 1
fi

echo -e "FIN"

exit 0
