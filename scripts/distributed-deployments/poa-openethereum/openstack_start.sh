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
fi

echo "[*] Sleeping before running copies"
sleep 5

if [[ $CLEAN -eq 1 ]]; then
  echo -e "[*] RUNNING CLEAN"
  for i in `seq 0 3`; do
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "pkill openethereum; sleep 1; rm -rf data passwords.txt diablo-spec.json peers.txt; touch peers.txt"
  done
fi


# ------------------------------------------------------------------------------
#                                 Copying
# ------------------------------------------------------------------------------

echo -e "[*] Copying"


for i in `seq 0 3`; do
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" passwords.txt ubuntu@localhost:~
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" -r node_directories/$i ubuntu@localhost:~/data
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" diablo-spec.json ubuntu@localhost:~/diablo-spec.json
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" bc_command.sh ubuntu@localhost:~
done

echo -e "${GOK} DONE"

# ------------------------------------------------------------------------------
#                                 Running Blockchain
# ------------------------------------------------------------------------------
echo -e "[*] Starting parity AuRa"
for i in `seq 0 3`; do
  echo "    [*] 812$i"
  sendcmd="nohup bash bc_command.sh $i > /dev/null 2>&1"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "$sendcmd &"
done

echo -e "[*] Waiting for network to be ready"

sleep 15

echo -e "${GOK} DONE"

# ------------------------------------------------------------------------------
#                                 Getting ENODES
# ------------------------------------------------------------------------------

rm peers.txt
touch peers.txt

for i in `seq 0 3`; do
  val=`ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "curl --data '{\"jsonrpc\":\"2.0\",\"method\":\"parity_enode\",\"params\":[],\"id\":0}' -H \"Content-Type: application/json\" -X POST localhost:8545"`
  echo $val
  x=`echo $val | awk '{split($0,a,","); split(a[2],b,":\""); gsub(/"/, "", b[2]); print b[2]}'`
  echo -e "[*] Enode ${BLUE}$x${RESET}"
  echo $x >> peers.txt

done

for i in `seq 0 3`; do
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "pkill openethereum"
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -P "812$i" peers.txt ubuntu@localhost:~/peers.txt
done
# ------------------------------------------------------------------------------
#                                 Running Blockchain
# ------------------------------------------------------------------------------
echo -e "[*] Starting parity AuRa"
for i in `seq 0 3`; do
  echo "    [*] 812$i"
  sendcmd="nohup bash bc_command.sh $i > /dev/null 2>&1"
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i $MACHINEKEY -p "812$i" -l ubuntu localhost "$sendcmd &"
done

echo -e "${GOK} DONE"
