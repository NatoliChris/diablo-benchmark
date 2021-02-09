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

rm -f permissioned-nodes.json

read -d '' template << _EOF_
[
  "enode://ac6b1096ca56b9f6d004b779ae3728bf83f8e22453404cc3cef16a3d9b96608bc67c4b30db88e0a5a6c6390213f7acbe1153ff6d23ce57380104288ae19373ef@%s:30305?discport=0&raftport=50400",
  "enode://0ba6b9f606a43a95edc6247cdb1c1e105145817be7bcafd6b2c0ba15d58145f0dc1a194f70ba73cd6f4cdd6864edc7687f311254c7555cc32e4d45aeb1b80416@%s:30305?discport=0&raftport=50400",
  "enode://579f786d4e2830bbcc02815a27e8a9bacccc9605df4dc6f20bcc1a6eb391e7225fff7cb83e5b4ecd1f3a94d8b733803f2f66b7e871961e7b029e22c155c3a778@%s:30305?discport=0&raftport=50400"
]
_EOF_

printf "$template" "$1" "$2" "$3" >> permissioned-nodes.json

echo -e "${GOK} Permissioned Nodes Generated in ${BLUE}permissioned-nodes.json${RESET}"
