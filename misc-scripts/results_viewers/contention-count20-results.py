
######################################
# General Imports
######################################

import json
import os

import matplotlib
from matplotlib import pyplot as plt
import numpy as np

font = {"size": 20}
matplotlib.rc("font", **font)

experiments = [
    "0",
    "20",
    "40",
    "60",
    "80",
    "100",
    "workaround",
]

experiment_data = {x: [] for x in experiments}
barwidth = 0.25

for ex in experiments:
    # list the things in the directory
    files = os.listdir("contention-hyperledger/count20/{}/results".format(ex))
    for f in files:
        if "results" in f:
            # if it's a result, read it to the file
            with open("contention-hyperledger/count20/{}/results/{}".format(ex, f), "r") as fp:
                experiment_data[ex].append(json.load(fp))


def plot_contention_bars():
    r1 = np.arange(len(experiment_data))
    averages = []
    xticks = []
    for ex in experiments:
        # let's make a bar graph!

        avg = 0
        for d in experiment_data[ex]:
            avg += d['AverageThroughput']

        averages.append(avg / len(experiment_data[ex]))
        xticks.append("{}%".format(ex) if ex != "workaround" else "workaround")

    plt.bar(r1, averages)
    plt.xticks(r1, tuple(xticks))

    plt.ylabel("Throughput (TX/Second)")
    plt.xlabel("Contention")
    plt.tight_layout()
    plt.show()


def plot_throughput_windows():
    for ex in experiments:
        throughput_seconds = []
        for d in experiment_data[ex]:
            for l in range(0, len(d["TotalThroughputOverTime"])):
                if l >= len(throughput_seconds):
                    throughput_seconds.append(0)

                throughput_seconds[l] += d["TotalThroughputOverTime"][l]

        throughput_averages = [x/len(experiment_data[ex]) for x in throughput_seconds]
        print(throughput_averages)
        plt.plot(
            [x for x in range(0, len(throughput_averages))],
            throughput_averages,
            label="{}%".format(ex) if ex != "workaround" else "workaround",
            linewidth=2
        )

    plt.legend(bbox_to_anchor=(0,1.02,1,0.2), loc="lower left", mode="expand", borderaxespad=1, ncol=3)
    plt.show()


# plot_contention_bars()
# plot_throughput_windows()

