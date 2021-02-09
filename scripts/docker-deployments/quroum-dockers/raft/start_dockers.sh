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

echo "----------------------------------------"
echo ${GREEN}Quorum Diablo Docker System${RESET}
echo "----------------------------------------"


if [[ $# -lt 1 ]]; then
  echo -e "\033[31m[ERR]\033[0m INVALID ARGS\n"
  echo "Usage: ./start_docker.sh <aviation|twitter|1000|5000|10000|50000>"
  exit 1
fi

case $1 in
  aviation)
    echo -e "${GOK} Aviation Workload Selected"
    echo -e "[*] Copying node folders to selected locations"
    echo -e "[*] Keys found at: ${GREEN}keys.json${RESET}"
    cp 100000/keys.json .
    cp 100000/genesis.json nodedata/genesis.json

    echo -e "[*] Chain Configuration: ${GREEN}quroum-docker.yaml${GREEN}"
    echo -e "${GOK} Configurations DONE"
    break;;
  twitter)
    echo -e "${GOK} Aviation Workload Selected"
    echo -e "[*] Copying node folders to selected locations"
    echo -e "[*] Keys found at: ${GREEN}keys.json${RESET}"
    cp twitter_accounts/keys.json .
    cp twitter_accounts/genesis.json nodedata/genesis.json

    echo -e "[*] Chain Configuration: ${GREEN}quroum-docker.yaml${GREEN}"
    echo -e "${GOK} Configurations DONE"
    break;;
  1000)
    echo -e "${GOK} Aviation Workload Selected"
    echo -e "[*] Copying node folders to selected locations"
    echo -e "[*] Keys found at: ${GREEN}keys.json${RESET}"
    cp 1000_accounts/keys.json .
    cp 1000_accounts/genesis.json nodedata/genesis.json

    echo -e "[*] Chain Configuration: ${GREEN}quroum-docker.yaml${GREEN}"
    echo -e "${GOK} Configurations DONE"
    break;;
  5000)
    echo -e "${GOK} Aviation Workload Selected"
    echo -e "[*] Copying node folders to selected locations"
    echo -e "[*] Keys found at: ${GREEN}keys.json${RESET}"
    cp 5000_accounts/keys.json .
    cp 5000_accounts/genesis.json nodedata/genesis.json

    echo -e "[*] Chain Configuration: ${GREEN}quroum-docker.yaml${GREEN}"
    echo -e "${GOK} Configurations DONE"
    break;;
  10000)
    echo -e "${GOK} Aviation Workload Selected"
    echo -e "[*] Copying node folders to selected locations"
    echo -e "[*] Keys found at: ${GREEN}keys.json${RESET}"
    cp 10000_accounts/keys.json .
    cp 10000_accounts/genesis.json nodedata/genesis.json

    echo -e "[*] Chain Configuration: ${GREEN}quroum-docker.yaml${GREEN}"
    echo -e "${GOK} Configurations DONE"
    break;;
  50000)
    echo -e "${GOK} Aviation Workload Selected"
    echo -e "[*] Copying node folders to selected locations"
    echo -e "[*] Keys found at: ${GREEN}keys.json${RESET}"
    cp 50000_accounts/keys.json .
    cp 50000_accounts/genesis.json nodedata/genesis.json

    echo -e "[*] Chain Configuration: ${GREEN}quroum-docker.yaml${GREEN}"
    echo -e "${GOK} Configurations DONE"
    break;;
  *)
    echo -e "${RERR} Invalid number of accounts selected, please generate this"
    exit 1;;
esac

echo -e "[*] Starting Docker"

sudo docker-compose up -d

echo -e "${GREEN}[DONE]{RESET}"

sudo docker ps -a
