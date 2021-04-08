import json

accounts = []
with open("/home/natc/dev/go/src/github.com/NatoliChris/diablo-benchmark/scripts/artifacts/ethereum_accounts", "r") as f:
    accounts = f.readlines()[:500000]


keyaccounts = []

for row in accounts:
    accInfo = row.strip().split(":")
    keyaccounts.append({"address": accInfo[0], "private": accInfo[1]})

with open('keys.json', 'w') as outfile:
    json.dump(keyaccounts, outfile)
