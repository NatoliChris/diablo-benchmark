#!/bin/bash


function yaml_generate {
    sed -e "s/\${HOST1_IP}/$1/" \
        -e "s/\${HOST2_IP}/$2/" \
        connection-org1-template.yaml | sed -e $'s/\\\\n/\\\n          /g'
}

HOST1_IP=$1
HOST2_IP=$2


echo "$(yaml_generate $HOST1_IP $HOST2_IP)" > ../crypto-config/peerOrganizations/org1.example.com/connection-org1.yaml

