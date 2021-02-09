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

rm -f diablo-spec.json
read -d '' template << _EOF_
{
    "name": "DiabloPOA",
    "engine": {
        "authorityRound": {
            "params": {
                "stepDuration": "1",
                "validators" : {
                    "list": [
                        "0xd2ffab60f7ed4b53f10155b21cee106c429ab506",
                        "0x0ec2aa16564d3ca9df603721815c8c521e858f48",
                        "0x333d24a961070e7002323e1373c77190e1016bb5",
                        "0x2f45bf71dde5d6a1132b737006b14a1573448f16"
                    ]
                }
            }
        }
    },
    "params": {
        "gasLimitBoundDivisor": "0x400",
        "maximumExtraDataSize": "0x20",
        "minGasLimit": "0x1388",
        "networkID" : "0x2323",
        "eip155Transition": 0,
        "validateChainIdTransition": 0,
        "eip140Transition": 0,
        "eip211Transition": 0,
        "eip214Transition": 0,
        "eip658Transition": 0
    },
    "genesis": {
        "seal": {
            "authorityRound": {
                "step": "0x0",
                "signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
            }
        },
        "difficulty": "0x20000",
        "gasLimit": "0x5B8D80"
    },
    "accounts": {
        "0x0000000000000000000000000000000000000001": { "balance": "1", "builtin": { "name": "ecrecover", "pricing": { "linear": { "base": 3000, "word": 0 } } } },
        "0x0000000000000000000000000000000000000002": { "balance": "1", "builtin": { "name": "sha256", "pricing": { "linear": { "base": 60, "word": 12 } } } },
        "0x0000000000000000000000000000000000000003": { "balance": "1", "builtin": { "name": "ripemd160", "pricing": { "linear": { "base": 600, "word": 120 } } } },
        "0x0000000000000000000000000000000000000004": { "balance": "1", "builtin": { "name": "identity", "pricing": { "linear": { "base": 15, "word": 3 } } } },
        "0xd2ffab60f7ed4b53f10155b21cee106c429ab506": { "balance": "1000000000000" },
        "0x0ec2aa16564d3ca9df603721815c8c521e858f48": { "balance": "1000000000000" },
        "0x333d24a961070e7002323e1373c77190e1016bb5": { "balance": "1000000000000" },
        "0x2f45bf71dde5d6a1132b737006b14a1573448f16": { "balance": "1000000000000" },
        %s
    }
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
	printf "$template" "$joinedgenesis" >> diablo-spec.json
else
	echo -e "${RERR} Failed to read accounts from file."
	exit 1
fi

echo -e "FIN"

exit 0

