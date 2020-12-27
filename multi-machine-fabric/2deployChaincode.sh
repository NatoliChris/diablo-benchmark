


PACKAGE_ID=$1
CHANNEL_NAME="mychannel"
CC_RUNTIME_LANGUAGE="golang"
VERSION="1"
CC_SRC_PATH="github.com/chaincode/basic"
CC_NAME="basic"


approveForOrg1(){
  docker exec cli peer lifecycle chaincode approveformyorg --tls \
  --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --waitForEvent \
  --package-id ${CC_NAME}_${VERSION}:${PACKAGE_ID}

    echo "===================== chaincode approved from org 1 ===================== "

}

approveForOrg2(){
docker exec -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp \
-e CORE_PEER_ADDRESS=peer0.org2.example.com:9051 \
-e CORE_PEER_LOCALMSPID="Org2MSP" \
-e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
cli peer lifecycle chaincode approveformyorg --tls \
--cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
--channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --waitForEvent \
--package-id ${CC_NAME}_${VERSION}:${PACKAGE_ID}
}


checkCommitReadyness(){
docker exec cli peer lifecycle chaincode checkcommitreadiness --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION}
}

commitChaincode(){
  docker exec cli peer lifecycle chaincode commit -o orderer.example.com:7050 --tls \
  --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  --peerAddresses peer0.org1.example.com:7051 \
  --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
  --peerAddresses peer0.org2.example.com:9051 \
  --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
  --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION}
}

queryCommited(){
  docker exec cli peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CC_NAME}
}

approveForOrg1
checkCommitReadyness
approveForOrg2
checkCommitReadyness
commitChaincode
queryCommited

