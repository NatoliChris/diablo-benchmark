#!/usr/bin/env python3

import copy
import json
import os
import random
import sys
import time

import yaml

DEBUG = True

if len(sys.argv) < 3:
    print("Error with program arguments:\nUsage: ./contention-creation.py <contention percentage> <yaml configuration file>")
    sys.exit(1)


if not os.path.isfile(sys.argv[2]):
    print("Error, file {} does not exist".format(sys.argv[2]))
    sys.exit(1)

# Read the configuration
CONFIG = {}
with open(sys.argv[2], 'r') as f:
    CONFIG = yaml.load(f.read(), Loader=yaml.SafeLoader)

# Figure out the number of total transactions
total_transactions = 0
intervals = []

interval_keys = [int(x) for x in CONFIG["bench"]["txs"].keys()]
interval_keys.sort()

current_key = interval_keys[0]
for next_key in interval_keys[1:]:
    start_tps = CONFIG["bench"]["txs"][current_key]
    end_tps = CONFIG["bench"]["txs"][next_key]

    increment = float(end_tps - start_tps) / (next_key - current_key)
    if DEBUG:
        print("[*] DEBUG: KEYS: ({}, {}); Start {}; End {}; Inc {}".format(current_key, next_key, start_tps, end_tps, increment))
        current_tps = start_tps
    for j in range(0, (next_key - current_key)):
        total_transactions += current_tps
        intervals.append(current_tps)
        current_tps = int(current_tps + increment)

    current_key = next_key
    if DEBUG:
        print("[*] DEBUG: Intervals: {}".format(intervals))
        print("[*] DEBUG: Total: {}".format(total_transactions))

total_updates = int((int(sys.argv[1]) / 100) * total_transactions)
total_creates = int(total_transactions - total_updates) + 1

print("Updates: {}; Create: {}".format(total_updates, total_creates))

createfunc = {
    "ID": "",
    "from": "",
    "to": "",
    "value": "0",
    "function": "CreateAsset",
    "txtype": "write",
    "params": [
        {
            "name": "id",
            "type": "string",
            "value": "0"
        },
        {
            "name": "value",
            "type": "uint",
            "value": "0"
        }
    ]
}

updatefunc = {
    "ID": "",
    "from": "",
    "to": "",
    "value": "0",
    "function": "UpdateAsset",
    "txtype": "write",
    "params": [
        {
            "name": "partID",
            "type": "string",
            "value": "0"
        },
        {
            "name": "value",
            "type": "uint",
            "value": ""
        },
    ]
}

all_functions = []

for i in range(0, total_creates):
    dc = copy.deepcopy(createfunc)
    dc["params"][0]["value"] = str(i)
    all_functions.append(dc)

for i in range(0, total_updates):
    dc = copy.deepcopy(updatefunc)
    dc["params"][1]["value"] = str(i)
    all_functions.append(dc)


# Shuffle all that
r = random.seed(time.time())
all_funcs_shuffled = all_functions[1:]
random.shuffle(all_funcs_shuffled)
all_funcs_shuffled = [all_functions[0]] + all_funcs_shuffled[:]


# Now create for each
num_workers = int(CONFIG["secondaries"]) * int(CONFIG["threads"])

premade_workload = []

accID = 0
for secondaryID in range(0, int(CONFIG["secondaries"])):
    secondaryWorkload = []
    for threadID in range(0, int(CONFIG["threads"])):
        threadWorkload = []
        for intervalTPS in intervals:
            interval_tx = []
            for i in range(0, int(intervalTPS / num_workers)):
                fn = all_funcs_shuffled[accID]
                fn["ID"] = str(accID)
                fn["from"] = str(accID)
                interval_tx.append(fn)
                accID += 1
            threadWorkload.append(interval_tx)
        secondaryWorkload.append(threadWorkload)
        threadWorkload = []
    premade_workload.append(secondaryWorkload)
    secondaryWorkload = []


if DEBUG:
    print("[*] DEBUG: \n\t [*] Secondary {}\n\t [*] Thread {}\n\t [*] Interval {}".format(len(premade_workload), len(premade_workload[0]), len(premade_workload[0][0])))

filename = "premade_data_contention_"+str(int(sys.argv[1]))+".json"

with open(filename, "w") as f:
    json.dump(premade_workload, f, indent=" ")
