
#./artifacts/channel/create-artifacts.sh

docker-compose -f ./artifacts/docker-compose.yaml up -d

sleep 20
./createChannel.sh

sleep 10

./deployChaincode.sh