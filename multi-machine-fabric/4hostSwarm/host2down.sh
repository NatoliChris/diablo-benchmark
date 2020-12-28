docker-compose -f host2.yaml down -v
docker rm $(docker ps -aq)

