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

## Load the configuration file
. config

NUM_ACCOUNTS=9500

# ------------------------------------------------------------------------------
#                                 Args
# ------------------------------------------------------------------------------

if [[ $# -lt 4 ]]; then
  echo -e "${RERR} NOT ENOUGH ARGS"
  exit 1
fi


# ------------------------------------------------------------------------------
#                                 Plumbing
# ------------------------------------------------------------------------------

echo -e "${GOK} DONE"

if [[ $GENERATE -eq 1 ]]; then
  echo -e "[*] Generation Required"
  bash generate_files.sh $NUM_ACCOUNTS
  bash generate_static_nodes.sh $1 $2 $3 $4
fi

if [[ $CLEAN -eq 1 ]]; then
  echo -e "[*] RUNNING CLEAN"
  for i in "$@"; do
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "pkill geth; rm -rf data passwords.txt genesis.json"
  done
fi


# ------------------------------------------------------------------------------
#                                 Copying
# ------------------------------------------------------------------------------

echo -e "[*] Copying"


gethcmd="/home/ubuntu/quorum/build/bin/geth --datadir ~/data init ~/genesis.json"

for i in "$@"; do
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY passwords.txt $user@$i:~
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -r node_directories/$i $user@$i:~/data
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY static-nodes.json $user@$i:~/data
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY generated_genesis.json $user@$i:~/genesis.json
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY bc_command.sh $user@$i:~
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "cp data/permissioned-nodes.json data/static-nodes.json"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i $gethcmd
done

echo -e "${GOK} DONE"

# ------------------------------------------------------------------------------
#                                 Running Blockchain
# ------------------------------------------------------------------------------
echo -e "[*] Starting geth"
id=0
for i in "$@"; do
  echo "[*] $i"
  sendcmd="nohup bash bc_command.sh $id > /dev/null 2>&1"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "$sendcmd &"
  id=$((id + 1))
done

echo -e "${GOK} DONE"

