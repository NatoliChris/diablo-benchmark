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

# Get all the IPS and make them into localhost
echo -e "[*] Plumbing"
currentport=8120

if [[ $KILLPREV -eq 1 ]]; then
  # Kill previous ssh things
  echo "[*] Killing previous machines"
  ps aux | grep "ssh -NL" | awk '{print $2}' | xargs kill
fi

sleep 5

for machineip in $@; do
  echo "    [*] 192.168.1.$machineip -> localhost:$currentport"
  ssh -NL $currentport:192.168.1.$machineip:22 -p $GATEPORT -i $GATEKEY -l $GATEUSER $GATEURL &
  currentport=$((currentport + 1))
done

jobs -l

echo -e "${GOK} DONE"

if [[ $GENERATE -eq 1 ]]; then
  echo -e "[*] Generation Required"
  bash generate_genesis.sh 5000
  bash generate_static_nodes.sh "192.168.1.$1" "192.168.1.$2" "192.168.1.$3" "192.168.1.$4"
fi

echo "[*] Sleeping before running copies"
sleep 5

if [[ $CLEAN -eq 1 ]]; then
  echo -e "[*] RUNNING CLEAN"
  for i in `seq 0 3`; do
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "pkill geth; rm -rf data passwords.txt genesis.json"
  done
fi


# ------------------------------------------------------------------------------
#                                 Copying
# ------------------------------------------------------------------------------

echo -e "[*] Copying"


gethcmd="geth --datadir ~/data init ~/genesis.json"

for i in `seq 0 3`; do
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" passwords.txt ubuntu@localhost:~
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" -r node_directories/$i ubuntu@localhost:~/data
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" static-nodes.json ubuntu@localhost:~/data
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" generated_genesis.json ubuntu@localhost:~/genesis.json
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" bc_command.sh ubuntu@localhost:~
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost $gethcmd
done

echo -e "${GOK} DONE"

# ------------------------------------------------------------------------------
#                                 Running Blockchain
# ------------------------------------------------------------------------------
echo -e "[*] Starting geth"
for i in `seq 0 3`; do
  echo "    [*] 812$i"
  sendcmd="nohup bash bc_command.sh $i > /dev/null 2>&1"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "$sendcmd &"
done

echo -e "${GOK} DONE"
