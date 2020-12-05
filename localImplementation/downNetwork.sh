
#./artifacts/channel/create-artifacts.sh

docker-compose -f ./artifacts/docker-compose.yaml down
wait
docker rm $(docker ps -aq)
