#!/usr/bin/env python3

"""
Generates the configurations for DIABLO as well as the premade
set of data
"""

######################################
# General Imports
######################################

import json
import random
import sys
import time

from scripts import ascii_graph

import yaml

######################################
# Colours for pretty printing
######################################


class BColors:
    """
    BColours contains the ANSI colours for printing to terminals
    """
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'

######################################
# Check argument (hyperledger/ethereum)
######################################


if len(sys.argv) < 2:
    print(f"{BColors.FAIL}[ERROR] No Argument provided, please choose hyperledger or ethereum{BColors.ENDC}")
    print(f"Usage: {sys.argv[0]} <hyperledger|ethereum>")
    sys.exit(1)

if sys.argv[1].lower() not in ("hyperledger", "ethereum"):
    print(f"{BColors.FAIL}[ERROR] \"hyperledger\" or \"ethereum\" required as parameter{BColors.ENDC}")
    print(f"Usage: {sys.argv[0]} <hyperledger|ethereum>")
    sys.exit(1)


######################################
# Load data
######################################
DATA = []
with open("raw_data/results_second.json", "r") as f:
    DATA = json.load(f)


print("-" * 80)
print(f"{BColors.OKGREEN}Exchange Workload Creator{BColors.ENDC}")
print("-" * 80)

######################################
# Visualisation of entire dataset
######################################

print(ascii_graph.ascii_plot(
    [DATA[i]["total"]["requests"] for i in range(0, len(DATA), 5)],
    width=100,
    height=30,
    logscale=True,
    pch="*",
    xlabel="Time (seconds)",
    ylabel="TPS")
)


######################################
# Get start time + duration
######################################

start_time = input("Please select the start point of your benchmark\n> ")
print("")
if start_time == "":
    start_time = 0
    print(f"{BColors.WARNING}[WARN]{BColors.ENDC} No start input, default to 0")
else:
    start_time = int(start_time)
    if start_time < 0:
        print(f"{BColors.WARNING}[WARN]{BColors.ENDC} start should be >= 0, default to 0")
        start_time = 0

print("")
duration_time = input("Duration of the benchmark? (In Seconds)\n> ")
print("")
if duration_time == "":
    duration_time = 5
    print(f"{BColors.WARNING}[WARN]{BColors.ENDC} No duration input, default to 5")
else:
    duration_time = int(duration_time)
    if duration_time < 5:
        print(f"{BColors.WARNING}[WARN]{BColors.ENDC} duration should be >= 5, default to 5")
        duration_time = 5

if(start_time + (duration_time)) > len(DATA):
    print(f"{BColors.WARNING}[WARN]{BColors.ENDC} Duration too long, fixing to end")
    duration_time = len(DATA) - start_time
    print(f"\t[*] Duration: {BColors.OKGREEN}{duration_time}{BColors.ENDC}")


######################################
# Print dataset
######################################

selected_data = DATA[start_time:(start_time + duration_time)]

print(f"[*] Data selected: (start={start_time}, end={start_time + duration_time})")
print("[*] Visualisation of data:")

print(ascii_graph.ascii_plot(
    [x["total"]["requests"] for x in selected_data],
    width=50,
    height=10,
    logscale=True,
    pch="*",
    xlabel="Time (seconds)",
    ylabel="TPS")
)

######################################
# Create the configuration file
######################################

print("[*] Generating configuration data")
sec = input("Number of secondaries?\n> ")

if sec == "":
    sec = 1
    print(f"{BColors.WARNING}[WARN]{BColors.ENDC} No secondary input, default to 1")
else:
    sec = int(sec)
    if sec < 1:
        print(f"{BColors.WARNING}[WARN]{BColors.ENDC} Secondary < 1, default to 1")
        sec = 1


print("")
threads = input("Number of threads?\n> ")
if threads == "":
    threads = 1
    print(f"{BColors.WARNING}[WARN]{BColors.ENDC} No threads input, default to 1")
else:
    threads = int(threads)
    if threads < 1:
        print(f"{BColors.WARNING}[WARN]{BColors.ENDC} Threads < 1, default to 1")
        threads = 1


contractpath = ""
if sys.argv[1].lower() == "ethereum":
    contractpath = "workloads/DBW-NASDAQ/contracts/solidity/exchange.sol"
else:
    contractpath = "workloads/DBW-NASDAQ/contracts/chaincode/exchange.go"

config_template = {
    "name": "NASDAQ Benchmark Configuration (start={}; end={})".format(
            start_time,
            start_time + duration_time),
    "description": "NASDAQ Top 10 stock trades pulled 2021-01-11",
    "secondaries": sec,
    "threads": threads,
    "bench": {
        "type": "premade",
        "datapath": "workloads/DBW-NASDAQ/premade_data.json",
        "txs": {i: selected_data[i]["total"]["requests"] for i in range(0, len(selected_data))}
    },
    "contract": {
        "name": "ExchangeContract",
        "path": contractpath
    }
}

print("[*] Configuration: ")

print(f"{BColors.OKGREEN}")
print(yaml.dump(config_template, default_flow_style=False))
print(f"{BColors.ENDC}")

with open("DBW-NASDAQ-CONFIG.yaml", "w") as f:
    yaml.dump(config_template, f)

print(f"[*] Configuration in {BColors.OKCYAN}DBW-NASDAQ-CONFIG.yaml{BColors.ENDC}")

print("[*] Generatng Premade Data")

######################################
# Generate premade data
######################################

total_workers = sec * threads

interval_data = []
# Generate each interval's transaction information (which stock, value)
for current_second in selected_data:
    second_data = []
    for stockname in current_second["stocks"]:
        for i in range(0, current_second["stocks"][stockname][1]):
            second_data.append((
                stockname,
                int(current_second["stocks"][stockname][0] / current_second["stocks"][stockname][1])
            ))
    random.seed(time.time())
    random.shuffle(second_data)
    interval_data.append(second_data)

accID = 0
workload = [[[] for x in range(0, threads)] for y in range(0, sec)]
for current_interval in interval_data:
    tx_per_thread = int(len(current_interval) / total_workers)
    txcount = 0
    for i in range(0, sec):
        for j in range(0, threads):
            # create the thread workload
            interval_workload = []
            for num_tx in range(0, tx_per_thread):
                interval_workload.append({
                    "ID": "{}".format(accID),
                    "from": "{}".format(accID),
                    "to": "contract",
                    "value": "0",
                    "function": "Buy",
                    "txtype": "write",
                    "params": [
                        {
                            "name": "stock",
                            "type": "string",
                            "value": "{}".format(current_interval[txcount][0])
                        },
                        {
                            "name": "amount",
                            "type": "uint256",
                            "value": "{}".format(current_interval[txcount][1])
                        },
                    ]
                })
                accID += 1
                txcount += 1
            workload[i][j].append(interval_workload)
    remaining = len(current_interval) - (tx_per_thread * total_workers)
    for remainingtx in range(0, remaining):
        workload[0][0][-1].append({
            "ID": "{}".format(accID),
            "from": "{}".format(accID),
            "to": "contract",
            "value": "0",
            "function": "Buy",
            "txtype": "write",
            "params": [
                {
                    "name": "stock",
                    "type": "string",
                    "value": "{}".format(current_interval[txcount][0])
                },
                {
                    "name": "amount",
                    "type": "uint256",
                    "value": "{}".format(current_interval[txcount][1])
                },
            ]
        })
        accID += 1
        txcount += 1

with open("premade_data.json", "w") as f:
    json.dump(workload, f, indent=" ")



print("[*] Summary")
print(f"  [*] Configuration in {BColors.OKCYAN}DBW-NASDAQ-CONFIG.yaml{BColors.ENDC}")
print(f"  [*] Premade Data in {BColors.OKCYAN}premade_data.json{BColors.ENDC}")
