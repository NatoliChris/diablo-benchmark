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
  bash generate_genesis.sh $NUM_ACCOUNTS
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


gethcmd="geth --datadir ~/data init ~/genesis.json"

id=0
for i in "$@"; do
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY passwords.txt $user@$i:~
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -r node_directories/$id $user@$i:~/data
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY diablo-spec.json $user@$i:~/diablo-spec.json
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY bc_command.sh $user@$i:~
  id=$((id + 1))
done

echo -e "${GOK} DONE"
# ------------------------------------------------------------------------------
#                                 Running Blockchain
# ------------------------------------------------------------------------------
echo -e "[*] Starting parity AuRa"
id=0
for i in "$@"; do
  echo "    [*] $i"
  sendcmd="nohup bash bc_command.sh $id > /dev/null 2>&1"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "$sendcmd &"
  id=$((id + 1))
done

echo -e "[*] Waiting for network to be ready"

sleep 15

echo -e "${GOK} DONE"

# ------------------------------------------------------------------------------
#                                 Getting ENODES
# ------------------------------------------------------------------------------

rm peers.txt
touch peers.txt

for i in "$@"; do
  val=`ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "curl --data '{\"jsonrpc\":\"2.0\",\"method\":\"parity_enode\",\"params\":[],\"id\":0}' -H \"Content-Type: application/json\" -X POST $i:8545"`
  echo $val
  x=`echo $val | awk '{split($0,a,","); split(a[2],b,":\""); gsub(/"/, "", b[2]); print b[2]}'`
  echo -e "[*] Enode ${BLUE}$x${RESET}"
  echo $x >> peers.txt

done

for i in "$@"; do
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "pkill openethereum"
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY peers.txt $user@$i:~/peers.txt
done
# ------------------------------------------------------------------------------
#                                 Running Blockchain
# ------------------------------------------------------------------------------
echo -e "[*] Starting parity AuRa"
id=0
for i in "$@"; do
  echo "    [*] $i"
  sendcmd="nohup bash bc_command.sh $id > /dev/null 2>&1"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -l $user $i "$sendcmd &"
done

echo -e "${GOK} DONE"
