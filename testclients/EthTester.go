package main

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs/parsers"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

/*
NOTE: start ganache with the mnemonic phrase:

nice charge tank ivory warfare spin deposit ecology beauty unusual comic melt
*/

func main() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)

	testSize := 1000

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
	for i := 0; i < testSize; i++ {
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

	// startNum, err := E.GetBlockHeight()

	// if err != nil {
	// 	panic(err)
	// }

	for i := 0; i < len(workload); i++ {
		err = E.SendRawTransaction(parsedWorkload[i])
		if err != nil {
			panic(err)
		}
	}

	tNow := time.Now()

	for {
		if E.NumTxDone == uint64(len(workload)) {
			break
		}
		if time.Now().Sub(tNow) > 10*time.Second {
			break
		}

		fmt.Printf("Sent: %d, Complete: %d\n", E.NumTxSent, E.NumTxDone)
		time.Sleep(1000 * time.Millisecond)
	}

	res := E.Cleanup()

	// endNum, err := E.GetBlockHeight()

	// if err != nil {
	// 	panic(err)
	// }

	// err = E.ParseBlocksForTransactions(startNum, endNum)

	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(E.Transactions)

	fmt.Printf("LATENCY: %.2f ms\n", res.AverageLatency)
	fmt.Printf("Throughput %.2f tps\n", res.Throughput)

	// for _, v := range E.Transactions {
	// 	fmt.Println((v[2].Sub(v[0])).Microseconds())
	// }

	fmt.Println("DONE, ALL OK")
	E.Close()
}
