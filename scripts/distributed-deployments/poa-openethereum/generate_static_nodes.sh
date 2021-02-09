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

# ------------------------------------------------------------------------------
#                                 Template
# ------------------------------------------------------------------------------

rm peers.txt

read -d '' template << _EOF_
enode://74d881f8e7c90d9b2aabf9135c8f1868295f5b22a49c07febaf21424ace75d0c093f6ecf3d4c34783f50cbaa04f967ecccabb964e32262b795cab1e6af98c76c@%s:30303
enode://6e33d25028b9a1adb93d40f7b8dbd11397c9bf86a1d6fd064d455db74dee98f13e67c20d706d73c2b6c6f5bc632a6333e84845c59e2063bb6d4a3416ec8df3e9@%s:30303
enode://61d1ea07a01e730dc3a746d161bd9e8eeb986671884fc1823686771167226ddc6c08f652443c050c0a7d0f3f5d226e5dd5c7a4c7c628f8f03e870db0f44aa119@%s:30303
enode://8a89847856c98d17607c52499bd1982a7c81eeb7b5e7f44e735ddb542bcb716974e8e58458bb63005555ddc0cb87d0f2057c393681cbcadf2ba8c68bcf979cfe@%s:30303

_EOF_

printf "$template" "$1" "$2" "$3" "$4" >> peers.txt

echo -e "${GOK} Peer list: ${BLUE}peers.txt${RESET}"
