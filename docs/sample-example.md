# Running the sample benchmark with an Ethereum dev network

To illustrate the way diablo operates, we utilise an Ethereum development
network to provide a single-machine blockchain with instant confirmation.
To do this, we utilise [ganache cli](https://github.com/trufflesuite/ganache-cli),
a simulation of the Ethereum blockchain.

To perform the benchmark as an example, follow the steps below.


## Pre-requisites

* [ganache-cli](https://github.com/trufflesuite/ganache-cli)
	* requires nodejs/npm
* Go version 1.14+
* Solidity `solc` verison 0.7.5 (prebuilt binary: [here](https://github.com/ethereum/solc-bin))


## Steps

* Build diablo
```sh
make
```
* In one terminal, start the ganache developer network with the following parameters, as this generates keys used in the configuration file already provided.
```sh
./scripts/sample/start_testnet.sh
```
	* Alternatively, you can also add the `-b` flag to make blocks created on a timed basis rather than per-transaction.
* View the benchmark configuration in ``scripts/sample/workloads/`` to a workload you would like to see, paying careful attention to the transaction intervals, as well as the number of secondaries and the number of threads.
* In another terminal, start the diablo primary
```sh
./diablo primary -c scripts/sample/workloads/sample_contract_store.yaml -cc scripts/sample/blockchain-configs/ganache-using-keyfile.yaml -a "0.0.0.0:8323" --level DEBUG
```

* You will need one terminal per secondary, so for 1 more secondary, start the secondary
```sh
./diablo secondary -m ":8323" -c scripts/sample/workloads/sample_contract_store.yaml -cc scripts/sample/blockchain-configs/ganache-using-keyfile.yaml --level DEBUG --timeout 2
```
	* Launch the number of secondaries specified in the configuration file.


The benchmark should run to completion and return the results.
Congratulations on running the benchmark!


## WARN json: cannot unmarshal hex number with leading 0

```
2020-09-14T10:47:40.080+1000	WARN	clientinterfaces/ethereum_interface.go:107	json: cannot unmarshal hex number with leading zero digits into Go struct field rpcBlock.transactions of type *hexutil.Big
diablo-benchmark/blockchains/clientinterfaces.(*EthereumInterface).parseBlocksForTransactions
	diablo-benchmark/blockchains/clientinterfaces/ethereum_interface.go:107
```

Ganache has a slightly different block structure, which is incorrectly decoded
by the Ethereum rpc client. This is a known issue with Go structs:

* [Example Issue](https://github.com/trufflesuite/ganache-core/issues/166)
