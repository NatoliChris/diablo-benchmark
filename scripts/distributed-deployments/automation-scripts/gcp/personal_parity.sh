#!/bin/bash

NUMREPEATS=5
EXPERIMENTS=("NASDAQ")
# CHAINS=("poa-geth" "poa-openethereum" "quorum-ibft" "quorum-raft")

DIABLOPATH="/home/natc/dev/go/src/github.com/NatoliChris/diablo-benchmark"
SC="/home/natc/dev/go/src/github.com/NatoliChris/diablo-benchmark/workloads/DBW-NASDAQ/contracts/solidity/simple-market.json"
CHAIN="poa-openethereum"
PRIV="6bd90d76df8a8fcf7b6ccfcaef942de084d3c339474bac181d9ab84d0eea2d64"
DATAPATH="/home/natc/PhD/research/benchmark-system/diablo-experiments-paper/configs/NASDAQ"
RESULTPATH="/home/natc/PhD/research/benchmark-system/diablo-experiments-paper/results/NASDAQ/2secondaries/poa-openethereum/"

mkdir -p $RESULTPATH

BCIPS=("164" "232" "213" "172")
#DIABLOPORTS=("8220" "8221" "8222" "8223" "8224")
DIABLOMACHINES=("8220" "8221" "8222")

#bash ./copy_file_to_all_servers.sh ~/.ssh/servers_key $DATAPATH/premade_data-4.json premade_data.json 8220 8221 8222 8223
prinf '%s\n' "${DIABLOMACHINES[@]}" | xargs bash ./copy_file_to_all_servers.sh ~/.ssh/servers_key $DATAPATH/premade_data-$((${#DIABLOMACHINES[@]} - 1)).json premade_data.json
# Run the primary
echo "[*] Running Primary"
primarycmd="nohup bash primary.sh > nohup.out 2>&1"
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[0]} "rm -f primarystart primaryfin; ps aux | grep 'primary.sh' | awk '{print \$2}' | xargs kill"
scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key primary.sh $user@${DIABLOMACHINES[0]}:~
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[0]} "$primarycmd &"

echo "[*] Running secondaries"
secondarycmd="nohup bash secondary.sh '192.168.1.151:8323' > nohup.out 2>&1"
for i in `seq 1 2`; do
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[$i]} "ps aux | grep 'secondary.sh' | awk '{print \$2}' | xargs kill; rm -f secondarystart"
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key secondary.sh $user@${DIABLOMACHINES[$i]}:~
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[$i]} "$secondarycmd &"
done


# 1. We assume that the accounts and configurations are already ready
for i in `seq 1 $NUMREPEATS`; do
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[$i]} "rm -f primarystart primaryfin"
  echo "RUNNING $i"

  # Re-run the blockchain on each one
  (cd "$DIABLOPATH/scripts/distributed-deployments/$CHAIN"; bash gcp_start.sh ${BCIPS[0]} ${BCIPS[1]} ${BCIPS[2]} ${BCIPS[3]})

  sleep 30

  # Deploy the related contract
  # ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -NL 8555:192.168.1.${BCIPS[0]}:8545 -p 9822 -i ~/.ssh/mainkey -l $user fortigate-217-0.cs.usyd.edu.au &
  echo "[*] Deploying SC: $SC with $PRIV"

  while :
  do
    rm ../currentdiablocontract
    (cd ..; node deploy_contract.js "${BCIPS[0]}:8545" /home/natc/dev/go/src/github.com/NatoliChris/diablo-benchmark/workloads/DBW-NASDAQ/contracts/solidity/simple-market.json $PRIV "simple-market.sol:DiabloMarket")

    sleep 1

    newaddr=`cat ../currentdiablocontract`

    # if this didn't work (i.e. we get an error) we need to restart the blockchain
    if [[ ! -z $newaddr ]]; then
      break
    fi

    # Re-run the blockchain on each one
    (cd "$DIABLOPATH/scripts/distributed-deployments/$CHAIN"; bash gcp_start.sh ${BCIPS[0]} ${BCIPS[1]} ${BCIPS[2]} ${BCIPS[3]})

    sleep 30
  done



  # Make the workload with the new address
  cp $DATAPATH/DBW-NASDAQ-CONFIG-2.yaml $DATAPATH/active_config.yaml
  sed -i "s/<<>>/${newaddr}/g" $DATAPATH/active_config.yaml

  for dmachine in ${DIABLOMACHINES[@]}; do
    scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key $DATAPATH/active_config.yaml $user@$dmachine:~/config.yaml
  done

  # Run the primary
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[0]} "touch ~/primarystart"

  sleep 1

  # Run the secondary
  for i in `seq 1 2`; do
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[$i]} "touch ~/secondarystart"
  done

  sleep 2

  # wait until the primary is done
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -l $user ${DIABLOMACHINES[0]} "bash primary_wait.sh"

  sleep 3
  # Download the results
  scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.ssh/servers_key -r $user@${DIABLOMACHINES[0]}:~/diablo-benchmark/results $RESULTPATH

  sleep 2
done
