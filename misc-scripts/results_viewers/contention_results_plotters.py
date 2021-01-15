######################################
# General Imports
######################################

import json
import os
from statistics import stdev

import matplotlib
from matplotlib import pyplot as plt
import numpy as np

font = {"size": 20}
matplotlib.rc("font", **font)

HYPERLEDGERPATH = "contention/count20"
QUORUMPATH = "quorum_experiments/new-contention"

experiments = [
    "0",
    "20",
    "40",
    "60",
    "80",
    "100"
]

all_experiments_info = {
    'hyperledger': {x: [] for x in experiments},
    'quorum': {x: [] for x in experiments}
}


for ex in experiments:
    # hyperledger
    files = os.listdir("{}/{}/results/".format(HYPERLEDGERPATH, ex))
    for f in files:
        if "results" in f:
            # if it's a result, read it to the file
            with open("{}/{}/results/{}".format(HYPERLEDGERPATH, ex, f), "r") as fp:
                all_experiments_info['hyperledger'][ex].append(json.load(fp))

    files = os.listdir("{}/{}/results/".format(QUORUMPATH, ex))
    for f in files:
        if "results" in f:
            # if it's a result, read it to the file
            with open("{}/{}/results/{}".format(QUORUMPATH, ex, f), "r") as fp:
                all_experiments_info['quorum'][ex].append(json.load(fp))


def plot_average_throughput_bars():
    BARWIDTH = 0.25

    hyperledgerthroughputs = []
    quorumthroughputs = []

    for ex in experiments:
        # hyperledger
        havg = 0
        for i in all_experiments_info['hyperledger'][ex]:
            havg += i['AverageThroughput']

        havg = (havg / len(all_experiments_info['hyperledger'][ex]))
        hyperledgerthroughputs.append(havg)

        qavg = 0
        for i in all_experiments_info['quorum'][ex]:
            qavg += i['AverageThroughput']

        qavg = (qavg / len(all_experiments_info['quorum'][ex]))
        quorumthroughputs.append(qavg)

    r1 = np.arange(len(hyperledgerthroughputs))
    r2 = [x + BARWIDTH for x in r1]

    plt.bar(r1, hyperledgerthroughputs, width=BARWIDTH,
            edgecolor='white', label='hyperledger')
    plt.bar(r2, quorumthroughputs, width=BARWIDTH,
            edgecolor='white', label='quorum')

    plt.xticks([r + BARWIDTH for r in range(len(hyperledgerthroughputs))],
               [ex for ex in experiments])
    plt.ylabel("Transactions per second")

    plt.legend(bbox_to_anchor=(0, 1.02, 1, 0.2), loc="lower left",
               mode="expand", borderaxespad=1, ncol=3)
    plt.tight_layout()

    plt.savefig('figures/contention/contention-comparison-average-throughput.png', dpi = 100)


def plot_max_throughput_bars():
    BARWIDTH = 0.25

    hyperledgerthroughputs = []
    quorumthroughputs = []

    for ex in experiments:
        # hyperledger
        havg = 0
        for i in all_experiments_info['hyperledger'][ex]:
            havg += i['MaximumThroughput']

        havg = (havg / len(all_experiments_info['hyperledger'][ex]))
        hyperledgerthroughputs.append(havg)

        qavg = 0
        for i in all_experiments_info['quorum'][ex]:
            qavg += i['MaximumThroughput']

        qavg = (qavg / len(all_experiments_info['quorum'][ex]))
        quorumthroughputs.append(qavg)

    r1 = np.arange(len(hyperledgerthroughputs))
    r2 = [x + BARWIDTH for x in r1]

    plt.bar(r1, hyperledgerthroughputs, width=BARWIDTH,
            edgecolor='white', label='hyperledger')
    plt.bar(r2, quorumthroughputs, width=BARWIDTH,
            edgecolor='white', label='quorum')

    plt.xticks([r + BARWIDTH for r in range(len(hyperledgerthroughputs))],
               [ex for ex in experiments])
    plt.ylabel("Transactions per second")

    plt.legend(bbox_to_anchor=(0, 1.02, 1, 0.2), loc="lower left",
               mode="expand", borderaxespad=1, ncol=3)
    plt.show()
    plt.close()


def plot_latency_bars():
    BARWIDTH = 0.25

    quorum_x = [x for x in range(0, len(experiments) * 2, 2)]
    hl_x = [x+0.5 for x in range(0, len(experiments) * 2, 2)]

    quorum_maxs = []
    hl_maxs = []

    quorum_avg = []
    hl_avg = []

    for ex in experiments:
        # hyperledger
        havg = 0
        maxs = 0
        for i in all_experiments_info['hyperledger'][ex]:
            havg += i['AverageLatency']
            maxs += i['MaxLatency']

        havg = havg / len(all_experiments_info['hyperledger'][ex])
        hl_avg.append(havg)
        hl_maxs.append(maxs)

        qavg = 0
        maxs = 0
        for i in all_experiments_info['quorum'][ex]:
            qavg += i['AverageLatency']
            maxs += i['MaxLatency']

        qavg = qavg / len(all_experiments_info['quorum'][ex])
        quorum_avg.append(qavg)
        quorum_maxs.append(maxs)

    q2_x = [x + BARWIDTH for x in quorum_x]
    hl2_x = [x + BARWIDTH for x in hl_x]

    plt.bar(quorum_x, quorum_maxs, width=BARWIDTH,
            edgecolor='white', label='Quorum (Max)')
    plt.bar(hl_x, hl_maxs, width=BARWIDTH,
            edgecolor='white', label='Hyperledger (Max)')

    plt.bar(q2_x, quorum_avg, width=BARWIDTH,
            edgecolor='white', label='Quorum (Avg)')
    plt.bar(hl2_x, hl_avg, width=BARWIDTH,
            edgecolor='white', label='Hyperledger (Avg)')

    plt.xticks([r + BARWIDTH for r in quorum_x], [ex for ex in experiments])
    plt.ylabel("Milliseconds")
    plt.yscale('log')
    plt.legend(bbox_to_anchor=(0, 1.02, 1, 0.2), loc="lower left",
               mode="expand", borderaxespad=1, ncol=2)

    plt.savefig('figures/contention/contention-comparison-average-latency.png', dpi=100)
    plt.close()


def plot_throughput_time_experiment(ex):
    hyperledger_windows = []
    for data in all_experiments_info['hyperledger'][ex][1:2]:
        for idx in range(0, len(data['TotalThroughputOverTime'])):
            if idx >= len(hyperledger_windows):
                hyperledger_windows.append([])
            hyperledger_windows[idx].append(
                data['TotalThroughputOverTime'][idx])

    hyperledger_averages = [sum(x) / len(x) for x in hyperledger_windows]

    quorum_windows = []
    for data in all_experiments_info['quorum'][ex][1:2]:
        for idx in range(0, len(data['TotalThroughputOverTime'])):
            if idx >= len(quorum_windows):
                quorum_windows.append([])
            quorum_windows[idx].append(data['TotalThroughputOverTime'][idx])

    quorum_averages = [sum(x) / len(x) for x in quorum_windows]

    plt.plot(
        [x*5 for x in range(0, len(hyperledger_averages))],
        hyperledger_averages,
        label="Hyperledger",
        linewidth=2,
        marker="s"
    )

    plt.plot(
        [x*5 for x in range(0, len(quorum_averages))],
        quorum_averages,
        label="Quorum",
        linewidth=2,
        marker="^"
    )

    plt.title("Throughput over time {}".format(ex))
    plt.xlabel("Time (seconds)")
    plt.ylabel("Transactions")
    plt.legend()
    plt.show()

def plot_throughput_time_all_onesystem(systemname, contention):

    markers=['s', '^', '*', 'x']
    filtered_exp = ["{}-{}%".format(x, contention) for x in ["100", "200", "250"]]

    fig = plt.figure(figsize=(10, 5))
    ax = plt.subplot(111)

    # Shrink current axis's height by 10% on the bottom
    box = ax.get_position()
    ax.set_position([box.x0, box.y0 + box.height * 0.3,
        box.width, box.height * 0.8])

    for exp in filtered_exp:
        avg_times = []
        for data in all_experiments_info[systemname][exp]:
            for idx in range(0, len(data['TotalThroughputOverTime'])):
                if idx >= len(avg_times):
                    avg_times.append([])
                avg_times[idx].append(
                    data['TotalThroughputOverTime'][idx]
                )

        avgs = [sum(x) / len(x) for x in avg_times]
        lower_errs = [min(x) for x in avg_times]
        upper_errs = [max(x) for x in avg_times]
        plt.errorbar(
            [x * 5 for x in range(0, len(avgs))],
            avgs,
            # yerr=[lower_errs, upper_errs],
            label=exp,
            linewidth=2,
            capsize=10,
            marker=markers.pop()
        )

    #plt.title("{} Throughput".format(systemname))
    plt.xlabel("Time (seconds)")
    plt.ylabel("Transactions")
    plt.legend(
        loc="upper center",
        ncol=3,
        borderaxespad=1,
        # mode="expand",
        bbox_to_anchor=(0.5, -0.2),
    )

    # Shrink current axis's height by 10% on the bottom
    # plt.legend(bbox_to_anchor=(), loc="",
    #            mode="expand", borderaxespad=1, ncol=3)
    plt.savefig('figures/contention/{}-{}-throughput.png'.format(systemname, contention), dpi=100)
    plt.close()


plot_average_throughput_bars()
#plot_max_throughput_bars()
#plot_throughput_time_experiment("100-3m")
#for i in ['hyperledger', 'quorum']:
#    for j in ['0', '20', '40', '60', '80','100']:
#        plot_throughput_time_all_onesystem(i, j)
plot_latency_bars()
