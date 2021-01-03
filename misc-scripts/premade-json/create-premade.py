import json
import sys

import yaml

DEBUG=True

if len(sys.argv) < 2:
    print("Usage:\n $ python create-premade.py /path/to/config")
    sys.exit(1)


# TYPES: 0  = account to account; 1 = CONTRACT
TXTYPE = 0
# TXTYPE = 1
DATA_PARAMS = lambda x: []

# Read the configuration
CONFIG = {}
with open(sys.argv[1], 'r') as f:
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
full_transactions = total_transactions * int(CONFIG["secondaries"]) * int(CONFIG["threads"])

premade_workload = []

accID = 0
if TXTYPE == 0:
    for secondaryID in range(0, int(CONFIG["secondaries"])):
        secondaryWorkload = []
        for threadID in range(0, int(CONFIG["threads"])):
            threadWorkload = []
            for intervalTPS in intervals:
                interval_tx = []
                for i in range(0, intervalTPS):
                    interval_tx.append({
                        'ID' : accID,
                        'from': "account{}".format(accID),
                        'to': "account{}".format((accID + 1) % full_transactions),
                        'value': '1',
                        'function': '',
                        'params': DATA_PARAMS(i)
                    })
                    accID += 1
                threadWorkload.append(interval_tx)
            secondaryWorkload.append(threadWorkload)
            threadWorkload = []
        premade_workload.append(secondaryWorkload)
        secondaryWorkload = []

if DEBUG:
    print("[*] DEBUG: \n\t [*] Secondary {}\n\t [*] Thread {}\n\t [*] Interval {}".format(len(premade_workload), len(premade_workload[0]), len(premade_workload[0][0])))


with open("premade_data.json", "w") as f:
    json.dump(premade_workload, f, indent=" ")
