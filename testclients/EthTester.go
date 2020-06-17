package main

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs/parsers"
	"fmt"
	"time"
)

/*
NOTE: start ganache with the mnemonic phrase:

nice charge tank ivory warfare spin deposit ecology beauty unusual comic melt
*/

func main() {
	cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/ethereum/ethereum-basic.yaml")
	if err != nil {
		panic(err)
	}

	bc, err := parsers.ParseBenchConfig("configurations/workloads/sample/sample_simple.yaml")
	if err != nil {
		panic(err)
	}

	G := workloadgenerators.EthereumWorkloadGenerator{}
	E := clientinterfaces.EthereumInterface{}
	E.Init(cc.Nodes)
	err = E.ConnectOne(0)

	if err != nil {
		panic(err)
	}

	err = G.Init(cc, bc)

	if err != nil {
		panic(err)
	}

	var workload [][]byte
	for i := 0; i < 10; i++ {
		txByte, err := G.CreateSignedTransaction(
			"0x9e3cf23f6fc76b77d2113db93ef388e057c8cc12",
			"1000000",
			[]byte{},
			cc.Keys[0],
		)

		if err != nil {
			panic(err)
		}
		workload = append(workload, txByte)
	}

	parsedWorkload, err := E.ParseWorkload(workload)

	if err != nil {
		panic(err)
	}

	startNum, err := E.GetBlockHeight()

	if err != nil {
		panic(err)
	}

	for i := 0; i < len(workload); i++ {
		err = E.SendRawTransaction(parsedWorkload[i])
		if err != nil {
			panic(err)
		}
	}

	time.Sleep(5 * time.Second)

	endNum, err := E.GetBlockHeight()

	if err != nil {
		panic(err)
	}

	err = E.ParseBlocksForTransactions(startNum, endNum)

	if err != nil {
		panic(err)
	}

	fmt.Println(E.Transactions)

	for _, v := range E.Transactions {
		fmt.Println((v[2].Sub(v[0])).Microseconds())
	}

	fmt.Println("DONE, ALL OK")
}
