#!/bin/bash


function yaml_generate1 {
    sed -e "s/\${HOST1_IP}/$1/" \
        -e "s/\${HOST2_IP}/$2/" \
        -e "s/\${HOST3_IP}/$3/" \
        -e "s/\${HOST4_IP}/$4/" \
        host1-template.yaml | sed -e $'s/\\\\n/\\\n          /g'
}

function yaml_generate2 {
    sed -e "s/\${HOST1_IP}/$1/" \
        -e "s/\${HOST2_IP}/$2/" \
        -e "s/\${HOST3_IP}/$3/" \
        -e "s/\${HOST4_IP}/$4/" \
        host2-template.yaml | sed -e $'s/\\\\n/\\\n          /g'
}

function yaml_generate3 {
    sed -e "s/\${HOST1_IP}/$1/" \
        -e "s/\${HOST2_IP}/$2/" \
        -e "s/\${HOST3_IP}/$3/" \
        -e "s/\${HOST4_IP}/$4/" \
        host3-template.yaml | sed -e $'s/\\\\n/\\\n          /g'
}

function yaml_generate4 {
    sed -e "s/\${HOST1_IP}/$1/" \
        -e "s/\${HOST2_IP}/$2/" \
        -e "s/\${HOST3_IP}/$3/" \
        -e "s/\${HOST4_IP}/$4/" \
        host4-template.yaml | sed -e $'s/\\\\n/\\\n          /g'
}

HOST1_IP=$1
HOST2_IP=$2
HOST3_IP=$3
HOST4_IP=$4

echo "$(yaml_generate1 $HOST1_IP $HOST2_IP $HOST3_IP $HOST4_IP)" > ../host1.yaml
echo "$(yaml_generate2 $HOST1_IP $HOST2_IP $HOST3_IP $HOST4_IP)" > ../host2.yaml
echo "$(yaml_generate3 $HOST1_IP $HOST2_IP $HOST3_IP $HOST4_IP)" > ../host3.yaml
echo "$(yaml_generate4 $HOST1_IP $HOST2_IP $HOST3_IP $HOST4_IP)" > ../host4.yaml
