package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math/big"
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

	// get the contract path
	contractPath := "contracts/Store.sol"

	c, err := compiler.CompileSolidity("", contractPath)

	if err != nil {
		fmt.Println(err)
		zap.L().Error("err", zap.Error(err))
		os.Exit(1)
	}

	cli, err := ethclient.Dial("ws://127.0.0.1:8545")

	if err != nil {
		fmt.Println(err)
		zap.L().Error("err", zap.Error(err))
		os.Exit(1)
	}

	price, err := cli.SuggestGasPrice(context.Background())

	if err != nil {
		fmt.Println(err)
		zap.L().Error("err", zap.Error(err))
		os.Exit(1)
	}

	priv, err := crypto.HexToECDSA("4019ff3bdda2101efd4a84afbf375604e24328203d5b5bfb47839bd4c390ad28")

	if err != nil {
		fmt.Println(err)
		zap.L().Error("err", zap.Error(err))
		os.Exit(1)
	}

	addrFrom := crypto.PubkeyToAddress(priv.PublicKey)
	//addrTo := "0x3fe51231d3cc16f1ed59e9fe255e2813d519ff5b"

	nonce, err := cli.PendingNonceAt(context.Background(), addrFrom)

	if err != nil {
		fmt.Println(err)
		zap.L().Error("err", zap.Error(err))
		os.Exit(1)
	}

	chainID, err := cli.ChainID(context.Background())

	if err != nil {
		fmt.Println(err)
		zap.L().Error("err", zap.Error(err))
		os.Exit(1)
	}

	// Get the transaction fields
	//toConverted := common.HexToAddress(addrTo)
	gasLimit := uint64(300000)

	// Make and sign the transaction
	for _, v := range c {
		fmt.Println(v.Code)
		fmt.Println(v.RuntimeCode)
		fmt.Println(v.Info)
		s, err := hex.DecodeString(v.Code)
		fmt.Println(s)
		tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, price, s)
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), priv)

		if err != nil {
			fmt.Println(err)
			zap.L().Error("err", zap.Error(err))
			os.Exit(1)
		}

		err = cli.SendTransaction(context.Background(), signedTx)

		if err != nil {
			fmt.Println(err)
			zap.L().Error("err", zap.Error(err))
			os.Exit(1)
		}

		for {
			r, err := cli.TransactionReceipt(context.Background(), signedTx.Hash())

			if err == nil {
				fmt.Println(r.ContractAddress)
				fmt.Println(r.ContractAddress)
				return
			}

			if err == ethereum.NotFound {
				time.Sleep(1 * time.Second)
				continue
			} else {
				break
			}
		}
	}

	//
	//testSize := 100
	//
	//cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/ethereum/ethereum-basic.yaml")
	//if err != nil {
	//	panic(err)
	//}
	//
	//bc, err := parsers.ParseBenchConfig("configurations/workloads/sample/sample_simple.yaml")
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//var G workloadgenerators.WorkloadGenerator
	//intermediate := workloadgenerators.EthereumWorkloadGenerator{}
	//G = intermediate.NewGenerator(cc, bc)
	//E := clientinterfaces.EthereumInterface{}
	//E.Init(cc.Nodes)
	//err = E.ConnectOne(0)
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = G.BlockchainSetup()
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = G.InitParams()
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//var workload [][]byte
	//for i := 0; i < testSize; i++ {
	//	bN, _ := big.NewInt(0).SetString("10000000", 10)
	//	txByte, err := G.CreateSignedTransaction(
	//		cc.Keys[0].PrivateKey,
	//		"0x9e3cf23f6fc76b77d2113db93ef388e057c8cc12",
	//		bN,
	//	)
	//	if err != nil {
	//		panic(err)
	//	}
	//	workload = append(workload, txByte)
	//}
	//
	//parsedWorkload, err := E.ParseWorkload(workload)
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//// startNum, err := E.GetBlockHeight()
	//
	//// if err != nil {
	//// 	panic(err)
	//// }
	//
	//for i := 0; i < len(workload); i++ {
	//	err = E.SendRawTransaction(parsedWorkload[i])
	//	if err != nil {
	//		panic(err)
	//	}
	//}
	//
	//tNow := time.Now()
	//
	//for {
	//	if E.NumTxDone == uint64(len(workload)) {
	//		break
	//	}
	//	if time.Now().Sub(tNow) > 10*time.Second {
	//		break
	//	}
	//
	//	fmt.Printf("Sent: %d, Complete: %d\n", E.NumTxSent, E.NumTxDone)
	//	time.Sleep(1000 * time.Millisecond)
	//}
	//
	//res := E.Cleanup()
	//
	//// endNum, err := E.GetBlockHeight()
	//
	//// if err != nil {
	//// 	panic(err)
	//// }
	//
	//// err = E.ParseBlocksForTransactions(startNum, endNum)
	//
	//// if err != nil {
	//// 	panic(err)
	//// }
	//
	//// fmt.Println(E.Transactions)
	//
	//fmt.Printf("LATENCY: %.2f ms\n", res.AverageLatency)
	//fmt.Printf("Throughput %.2f tps\n", res.Throughput)
	//
	//// for _, v := range E.Transactions {
	//// 	fmt.Println((v[2].Sub(v[0])).Microseconds())
	//// }
	//
	//fmt.Println("DONE, ALL OK")
	//E.Close()
}
