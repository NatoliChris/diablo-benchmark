docker-compose -f host4.yaml down -v
docker rm $(docker ps -aq)
