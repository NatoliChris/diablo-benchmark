
#./artifacts/channel/create-artifacts.sh

docker-compose -f ./artifacts/docker-compose.yaml up -d
wait
./createChannel.sh
wait
./deployChaincode.sh