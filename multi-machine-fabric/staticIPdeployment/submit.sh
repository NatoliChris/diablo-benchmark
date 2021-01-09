#!/bin/bash
for ((i = 0; i <= 100; i++ ))
do
   sudo docker exec cli peer chaincode invoke -o orderer3.example.com:9050 \
   --tls true \
   --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer3.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    -C mychannel -n basic --peerAddresses peer0.org1.example.com:7051  \
    --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
     --peerAddresses peer0.org2.example.com:9051 \
     --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
     -c '{"function":"CreateAsset","Args":["'$i'","red","5","YUNGBULL","58"]}'

done