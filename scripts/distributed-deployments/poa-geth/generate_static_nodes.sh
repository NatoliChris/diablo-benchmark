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
rm -f static-nodes.json

read -d '' template << _EOF_
[
	"enode://6ca7523c3ba2f82d2fbb6eff3caf2df770cbc107d7e8a71d2d953366a245c86bd85e9abcc60eae688151d148c7d7128cb67d65fb55c674aa630aef9243a1a80f@%s:30303?discport=0",
	"enode://fed128a47681df058c1ac8d91b8e159acead842d65c0daaee214c4ca238c06b8fe86ad7a280ba863c2c9cd7b15f0b90ab2ba12f948ae02421021f9b16f1e51e9@%s:30303?discport=0",
	"enode://6cc4da743d04277a4416594b9a1ee871f544854a0cf9c4a0eb16f600df0121d8def216905205433eec65b9489e74477bf578aa0ef2120956eca9d96e004194e6@%s:30303?discport=0",
	"enode://1160f948f3c9fb041efab2dedc8f55ef0d688956bd4898cb3bc9b6ddf9b391c92c1e199f4cfe1d0b4dbd683487b9ab905e29f353d202947d230e7bd51fee84a4@%s:30303?discport=0"
]
_EOF_

printf "$template" "$1" "$2" "$3" "$4" >> static-nodes.json

echo -e "${GOK} Static Nodes Generated in ${BLUE}static-nodes.json${RESET}"
