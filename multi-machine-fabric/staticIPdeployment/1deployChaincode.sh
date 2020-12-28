
CHANNEL_NAME="mychannel"
CC_RUNTIME_LANGUAGE="golang"
VERSION="1"
CC_NAME="basic"
CC_SRC_PATH="github.com/hyperledger/fabric/peer/chaincode/${CC_NAME}"



packageChaincode(){
  docker exec cli peer lifecycle chaincode package ${CC_NAME}.tar.gz \
        --path ${CC_SRC_PATH} \
        --label ${CC_NAME}_${VERSION}

        echo "===================== Chaincode is packaged on peer0.org1 ===================== "
}

installChaincode(){
  # copy packages directory to skip packaging chaincode

 docker exec cli peer lifecycle chaincode install ${CC_NAME}.tar.gz
  echo "===================== Chaincode is installed on peer0.org1 ===================== "

docker exec -e CORE_PEER_ADDRESS=peer1.org1.example.com:8051 -e \
CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/ca.crt \
cli peer lifecycle chaincode install ${CC_NAME}.tar.gz

echo "===================== Chaincode is installed on peer1.org1 ===================== "

docker exec -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp \
-e CORE_PEER_ADDRESS=peer0.org2.example.com:9051 \
-e CORE_PEER_LOCALMSPID="Org2MSP" \
-e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
cli peer lifecycle chaincode install ${CC_NAME}.tar.gz

echo "===================== Chaincode is installed on peer0.org2 ===================== "

docker exec -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp \
-e CORE_PEER_ADDRESS=peer1.org2.example.com:10051 \
-e CORE_PEER_LOCALMSPID="Org2MSP" \
-e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer1.org2.example.com/tls/ca.crt \
cli peer lifecycle chaincode install ${CC_NAME}.tar.gz

echo "===================== Chaincode is installed on peer1.org2 ===================== "


}

#packageChaincode
installChaincode
