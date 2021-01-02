import json
import sys

import yaml


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

idx = 0
current_key = list(CONFIG["bench"]["txs"].keys())[0]
for next_key in list(CONFIG["bench"]["txs"].keys())[1:]:

    start_tps = CONFIG["bench"]["txs"][current_key]
    end_tps = CONFIG["bench"]["txs"][next_key]

    increment = float(end_tps - start_tps) / (next_key - current_key)

    current_tps = start_tps
    for j in range(0, (next_key - current_key)):
        total_transactions += current_tps
        current_tps = int(current_tps + increment)

    current_key = next_key

full_transactions = total_transactions * int(CONFIG["secondaries"]) * int(CONFIG["threads"])

premade_workload = []

if TXTYPE == 0:
    for secondaryID in range(0, int(CONFIG["secondaries"])):
        secondaryWorkload = []
        for threadID in range(0, int(CONFIG["threads"])):
            threadWorkload = []
            for i in range(0, total_transactions):
                accID = (secondaryID * threadID * total_transactions) + i
                threadWorkload.append({
                    'from': "account{}".format(accID),
                    'to': "account{}".format((accID + 1) % full_transactions),
                    'value': '1',
                    'function': '',
                    'params': DATA_PARAMS(i)
                })
            secondaryWorkload.append(threadWorkload)
            threadWorkload = []
        premade_workload.append(secondaryWorkload)
        secondaryWorkload = []


with open("premade_data.json", "w") as f:
    json.dump(premade_workload, f, indent=" ")
