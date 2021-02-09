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

read -d '' template << _EOF_
[
	"enode://0fea913e782360802c3dca962f82d8b7741bb3c05b8acd4021e594201617baae91998b96af4ec0f121320cd86ea70d500d980e976e4016aa7691db4c7f621c98@%s:30305?discport=0",
	"enode://431544d4e02122432d44417b054c9f41eb72ce4b53c9af713a4ae5b14fcd7bbe26a84eec28ede5d468206ce2936adf5d65c22e2d4ae0fd84e92a0040838a044c@%s:30305?discport=0",
	"enode://d0e539036df701e8fa099802b33c0c07e4d095b6901eadf101d4067f9d902d862a037703b9c210fe376bcbdf2e3a6e653e20931325b44fa0c163718eaa18e4fd@%s:30305?discport=0",
	"enode://c84fdf7ea618a99c5b5d0099e3cd1f91c3bd5359e1b38a29166238fa6b7078600da745ea6f93b0f0c362a53464ab7c147186cd91522c272796490746ebfb92dd@%s:30305?discport=0"
]
_EOF_

printf "$template" "$1" "$2" "$3" "$4" >> static-nodes.json

echo -e "${GOK} Static Nodes Generated in ${BLUE}static-nodes.json${RESET}"
