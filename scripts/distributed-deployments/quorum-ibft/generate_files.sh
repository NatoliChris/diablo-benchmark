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
    "config": {
        "chainId": 10,
        "homesteadBlock": 0,
        "eip150Block": 0,
        "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "eip155Block": 0,
        "eip158Block": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "istanbul": {
            "epoch": 30000,
            "policy": 0,
            "ceil2Nby3Block": 0
        },
        "txnSizeLimit": 64,
        "maxCodeSizeConfig": [
          {
            "block": 0,
            "size": 35
          }
        ],
        "isQuorum": true
    },
    "nonce": "0x0",
    "timestamp": "0x602080f3",
    "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000f89af8549447d0548432502e9e98556018bfcffd4787d991eb94bbb28fb9166a89c5648c2cf3568055dcfe998ae294717fe5040b73836514c069ee057181db13e5cbe7944947112c7ac6da4978c8c3d29de17f6abafb40cab8410000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0",
    "gasLimit": "0xe0000000",
    "difficulty": "0x1",
    "mixHash": "0x63746963616c2062797a616e74696e65206661756c7420746f6c6572616e6365",
    "coinbase": "0x0000000000000000000000000000000000000000",
    "alloc": {
        "47d0548432502e9e98556018bfcffd4787d991eb": {
            "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
        },
        "4947112c7ac6da4978c8c3d29de17f6abafb40ca": {
            "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
        },
        "717fe5040b73836514c069ee057181db13e5cbe7": {
            "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
        },
        "bbb28fb9166a89c5648c2cf3568055dcfe998ae2": {
            "balance": "0x446c3b15f9926687d2c40534fdb564000000000000"
        },
	%s
    },
    "number": "0x0",
    "gasUsed": "0x0",
    "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
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
