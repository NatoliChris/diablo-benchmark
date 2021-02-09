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

AVIATIONACC=100000
TWITTERACC=600000

# ------------------------------------------------------------------------------
#                                 Template
# ------------------------------------------------------------------------------

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
        "size" : 32
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
echo -e "${GREEN}Welcome to the Quorum Diablo Docker System $RESET"

if [[ $# -lt 1 ]]; then
  echo -e "\033[31m[ERR]\033[0m INVALID ARGS\n"
  echo "Usage: ./pre-generation.sh <aviation|twitter>"
  echo "OR"
  echo "Usage: ./pre-generation.sh naccounts <number of accounts>"
  exit 1
fi

echo "[*] Setting Up Quorum"

case $1 in
  aviation)
    echo -e "${GOK} Aviation Workload";
    echo -e "[*] Setting Up ${AVIATIONACC} accounts for quorum"
    accs=$(head -n $AVIATIONACC accounts_raw)
    echo -e "    ${GOK} Found $(echo "$accs" | wc -l)"
    echo -e "[*] Formatting to accounts file: ${GREEN}keys.json${RESET}"
    genesislines=()
    accountslines=()
    while IFS= read -r accrow; do
      addr=$(echo "$accrow" | cut -d ':' -f 1 | xargs)
      priv=$(echo "$accrow" | cut -d ':' -f 2 | xargs)

      # Add to the accounts file
      accountsline+=("{\"address\":\"${addr}\",\"private\":\"${priv}\"}")

      # Add to genesis
      genesislines+=("\"${addr}\": {\"balance\":\"10000100000000000000\"}")
    done <<< "$accs"

    # Print to the files
    echo "[" > keys.json
    echo $(join_arr , "${accountsline[@]}") >> keys.json
    echo "]" >> keys.json
    ## Genesis
    joinedgenesis=$(join_arr , "${genesislines[@]}")

    echo -e "${GOK} DONE"
    echo -e "${GOK} Genesis DONE"
    printf "$template" "$joinedgenesis" >> generated_genesis.json

    mv generated_genesis.json nodedata/genesis.json
    echo -e "[*] Spawning Docker"
    # sudo docker-compose up -d
    exit 1;;
  twitter)
    echo -e "${GOK} Twitter Workload";
    echo -e "[*] Setting Up ${TWITTERACC} accounts for quorum"
    accs=$(head -n $TWITTERACC accounts_raw)
    echo -e "    ${GOK} Found $(echo "$accs" | wc -l)"
    echo -e "[*] Formatting to accounts file: ${GREEN}keys.json${RESET}"
    genesislines=()
    accountslines=()
    while IFS= read -r accrow; do
      addr=$(echo "$accrow" | cut -d ':' -f 1 | xargs)
      priv=$(echo "$accrow" | cut -d ':' -f 2 | xargs)

      # Add to the accounts file
      accountsline+=("{\"address\":\"${addr}\",\"private\":\"${priv}\"}")

      # Add to genesis
      genesislines+=("\"${addr}\": {\"balance\":\"10000100000000000000\"}")
    done <<< "$accs"

    # Print to the files
    echo "[" > keys.json
    echo $(join_arr , "${accountsline[@]}") >> keys.json
    echo "]" >> keys.json
    ## Genesis
    joinedgenesis=$(join_arr , "${genesislines[@]}")

    echo -e "${GOK} DONE"
    echo -e "${GOK} Genesis DONE"
    printf "$template" "$joinedgenesis" >> generated_genesis.json

    mv generated_genesis.json nodedata/genesis.json
    echo -e "[*] Spawning Docker"
    # sudo docker-compose up -d
    exit 1;;
  naccounts)

    if [[ $# -lt 2 ]]; then
      echo -e "${RERR} Requires number of accounts for \"naccounts\""
      echo -e "\nUsage: ./pre-generation.sh naccounts <number of accounts>"
      exit 1
    fi

    echo -e "[${GREEN}OK${RESET}] Accounts ${2}";


    if [[ $2 -gt $(cat accounts_raw | wc -l) ]]; then
      echo -e "${RERR} Too many accounts, requires more generation"
      exit 1
    fi

    accs=$(head -n $2 accounts_raw)
    echo -e "    ${GOK} Found $(echo "$accs" | wc -l)"
    echo -e "[*] Formatting to accounts file: ${GREEN}keys.json${RESET}"
    genesislines=()
    accountslines=()
    while IFS= read -r accrow; do
      addr=$(echo "$accrow" | cut -d ':' -f 1 | xargs)
      priv=$(echo "$accrow" | cut -d ':' -f 2 | xargs)

      # Add to the accounts file
      accountsline+=("{\"address\":\"${addr}\",\"private\":\"${priv}\"}")

      # Add to genesis
      genesislines+=("\"${addr}\": {\"balance\":\"10000100000000000000\"}")
    done <<< "$accs"

    # Print to the files
    echo "[" > keys.json
    echo $(join_arr , "${accountsline[@]}") >> keys.json
    echo "]" >> keys.json
    ## Genesis
    joinedgenesis=$(join_arr , "${genesislines[@]}")
    echo "${joinedgenesis}"

    echo -e "${GOK} DONE"
    echo -e "${GOK} Genesis DONE"
    printf "$template" "$joinedgenesis" >> generated_genesis.json

    mv generated_genesis.json nodedata/genesis.json
    echo -e "[*] Spawning Docker"
    # sudo docker-compose up -d
    exit 1;;
  * ) echo -e "[${RED}ERR${RED}] Invalid Argument";
    exit 1;;
esac

