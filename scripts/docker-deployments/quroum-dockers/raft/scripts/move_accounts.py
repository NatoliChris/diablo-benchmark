import json
accounts = []

keyaccounts = []
genesisaccounts = {
    "0xebf4298e6f75fbc97d16fb5f0dfe24727dd56ffb": {
        "balance": "1000010001000101010"
    },
    "0xed9d02e382b34818e88b88a309c7fe71e65f419d": {
        "balance": "1000010001000101010"
    },
    "0xca843569e3427144cead5e4d5999a3d0ccf92b8e": {
        "balance": "1000010001000101010"
    },
    "0x0fbdc686b912d7722dc86510934589e0aaf3b55a": {
        "balance": "1000010001000101010"
    },
    "0x9186eb3d20cbd1f5f992a950d808c4495153abd5": {
        "balance": "1000010001000101010"
    },
    "0x0638e1574728b6d862dd5d3a3e0942c3be47d996": {
        "balance": "1000010001000101010"
    }
}

fullgenesis = {
    "alloc": {
    },
    "coinbase": "0x0000000000000000000000000000000000000000",
    "config": {
        "homesteadBlock": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "petersburgBlock": 0,
        "istanbulBlock": 0,
        "chainId": 10,
        "eip150Block": 0,
        "eip155Block": 0,
        "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "eip158Block": 0,
        "isQuorum": True,
        "maxCodeSizeConfig": [
                {
                    "block": 0,
                    "size": 32
                }
        ]
    },
    "difficulty": "0x0",
    "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "gasLimit": "0x00000000",
    "mixhash": "0x00000000000000000000000000000000000000647572616c65787365646c6578",
    "nonce": "0x0",
    "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "timestamp": "0x00"
}

with open("accounts_raw", "r") as f:
    accounts = f.readlines()[:100000]

for row in accounts:
    accInfo = row.strip().split(":")

    genesisaccounts[accInfo[0]] = {"balance": "1000000000000000000000"}
    keyaccounts.append({"address": accInfo[0], "private": accInfo[1]})


with open("keys.json", "w") as outfile:
    json.dump(keyaccounts, outfile)


fullgenesis["alloc"] = genesisaccounts

with open("generated_genesis.json", "w") as outfile:
    json.dump(fullgenesis, outfile)
