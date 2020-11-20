// Package testclients provides testing clients to test and generate information
// used through development. The files in this folder are ONLY USED DURING
// DEVELOPMENT and act as a sandbox to test and write functionality.
package main

import (
	"context"
	"diablo-benchmark/blockchains/clientinterfaces"
	types2 "diablo-benchmark/blockchains/types"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

/*
NOTE: start ganache with the mnemonic phrase:

nice charge tank ivory warfare spin deposit ecology beauty unusual comic melt
*/

func getTransactionReceipt(E *clientinterfaces.EthereumInterface, hash common.Hash) bool {
	for {
		r, err := E.PrimaryNode.TransactionReceipt(context.Background(), hash)
		if err == nil {
			fmt.Println(r)
			m, _ := r.MarshalJSON()
			fmt.Println(fmt.Sprintf("%s", m))
			break
		}

		if err == ethereum.NotFound {
			time.Sleep(1 * time.Second)
			continue
		} else {
			break
		}
	}

	return true
}

func main() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)

	cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/fabric/fabric-basic.yaml")
	if err != nil {
		panic(err)
	}

	bc, err := parsers.ParseBenchConfig("configurations/workloads/fabric/testDiabloFabric.yaml")

	if err != nil {
		panic(err)
	}

	var generator workloadgenerators.WorkloadGenerator
	intermediate := workloadgenerators.FabricWorkloadGenerator{}
	generator = intermediate.NewGenerator(cc, bc)
	client := clientinterfaces.FabricInterface{}

	log.Println("Init client interface")
	client.Init(cc.Nodes)


	err = generator.BlockchainSetup()
	if err != nil {
		panic(err)
	}

	err = generator.InitParams()

	if err != nil {
		panic(err)
	}


	var cParamList []configs.ContractParam

	cParamList = append(cParamList,
		configs.ContractParam{
			Type:  "uint64",
			Value: "1",
		},
		configs.ContractParam{
			Type:  "string",
			Value: "assetID",
		},
		configs.ContractParam{
			Type:  "string",
			Value: "color",
		}, configs.ContractParam{
			Type:  "string",
			Value: "100",
		}, configs.ContractParam{
			Type:  "string",
			Value: "owner",
		},configs.ContractParam{
			Type:  "string",
			Value: "2103",
		})


	tx, err := generator.CreateInteractionTX(
		nil,
		"basic",
		"CreateAsset",
		cParamList,
	)


	var parsedTx types2.FabricTX
	_ = json.Unmarshal(tx, &parsedTx)

	log.Println("sendRawTransaction")
	err = client.SendRawTransaction(&parsedTx)

	log.Println("--> Evaluate Transaction: ReadAsset, function returns an asset with a given assetID")
	result, err := client.Contracts["basic"].EvaluateTransaction("ReadAsset", "asset1")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println(string(result))

}


	func fabricTest() {


	}



	func ethTest(){
	//addr := "0x9e3cf23f6fc76b77d2113db93ef388e057c8cc12"
	//a := common.HexToAddress(addr)
	//
	//fmt.Println(a.Bytes())
	//fmt.Println(len(a.Bytes()))
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)

	// get the contract path
	contractPath := "contracts/Store.sol"

	cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/ethereum/ethereum-basic.yaml")
	if err != nil {
		panic(err)
	}

	bc, err := parsers.ParseBenchConfig("configurations/workloads/sample/sample_simple.yaml")

	if err != nil {
		panic(err)
	}

	var G workloadgenerators.WorkloadGenerator
	intermediate := workloadgenerators.EthereumWorkloadGenerator{}
	G = intermediate.NewGenerator(cc, bc)
	E := clientinterfaces.EthereumInterface{}
	E.Init(cc.Nodes)
	err = E.ConnectOne(0)

	if err != nil {
		panic(err)
	}

	err = G.BlockchainSetup()
	if err != nil {
		panic(err)
	}

	err = G.InitParams()

	if err != nil {
		panic(err)
	}

	// priv, _ := crypto.HexToECDSA("4019ff3bdda2101efd4a84afbf375604e24328203d5b5bfb47839bd4c390ad28")
	b, _ := hex.DecodeString("4019ff3bdda2101efd4a84afbf375604e24328203d5b5bfb47839bd4c390ad28")

	contractAddr, err := G.DeployContract(b, contractPath)

	if err != nil {
		panic(err)
	}

	var cParamList []configs.ContractParam

	cParamList = append(cParamList,
		configs.ContractParam{
			Type:  "uint32",
			Value: "10000",
		})

	tx, err := G.CreateInteractionTX(
		b,
		contractAddr,
		"storeVal(uint32)",
		cParamList,
	)

	fmt.Println(contractAddr)

	var parsedTx types.Transaction
	_ = json.Unmarshal(tx, &parsedTx)

	err = E.SendRawTransaction(&parsedTx)


}

	// %---------------------------------------------------------%

	//c, err := compiler.CompileSolidity("solc", contractPath)
	//if err != nil {
	//	fmt.Println(err)
	//	zap.L().Error("err", zap.Error(err))
	//	os.Exit(1)
	//}

	//cli, err := ethclient.Dial("ws://127.0.0.1:8545")
	//
	//if err != nil {
	//	fmt.Println(err)
	//	zap.L().Error("err", zap.Error(err))
	//	os.Exit(1)
	//}
	//
	//price, err := cli.SuggestGasPrice(context.Background())

	//if err != nil {
	//	fmt.Println(err)
	//	zap.L().Error("err", zap.Error(err))
	//	os.Exit(1)
	//}

	//priv, err := crypto.HexToECDSA("4019ff3bdda2101efd4a84afbf375604e24328203d5b5bfb47839bd4c390ad28")
	//
	//if err != nil {
	//	fmt.Println(err)
	//	zap.L().Error("err", zap.Error(err))
	//	os.Exit(1)
	//}

	//addrFrom := crypto.PubkeyToAddress(priv.PublicKey)
	//addrTo := "0x3fe51231d3cc16f1ed59e9fe255e2813d519ff5b"

	//nonce, err := cli.PendingNonceAt(context.Background(), addrFrom)
	//
	//if err != nil {
	//	fmt.Println(err)
	//	zap.L().Error("err", zap.Error(err))
	//	os.Exit(1)
	//}
	//
	//chainID, err := cli.ChainID(context.Background())
	//
	//if err != nil {
	//	fmt.Println(err)
	//	zap.L().Error("err", zap.Error(err))
	//	os.Exit(1)
	//}

	// Get the transaction fields
	//toConverted := common.HexToAddress(addrTo)
	//gasLimit := uint64(300000)

	//fmt.Println(c)
	//
	//// Make and sign the transaction
	//for _, v := range c {
	//
	//	funcHash := v.Hashes["storeVal(uint32)"]
	//	fmt.Println("Func Hash: ", funcHash)
	//	funcHashBytes, err := hex.DecodeString(funcHash)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	// Store the number
	//	buf := new(bytes.Buffer)
	//	n := uint32(8)
	//	fmt.Println("Num: ", n)
	//	binary.Write(buf, binary.BigEndian, n)
	//	bts := buf.Bytes()
	//
	//	pad := make([]byte, 28)
	//	payload := append(funcHashBytes, pad...)
	//	payload = append(payload, bts...)
	//	fmt.Println("payload", payload)
	//	fmt.Println(hex.EncodeToString(payload))
	//
	//	// add := common.HexToAddress("0x7ee82060e8ea5f5daede2c16e0a7524072e3f147")
	//
	//	// cAddr := common.HexToAddress("0x1f840420B74471B674e0c86C77D43A32E367ED95")
	//	tx, err := G.CreateSignedTransaction(
	//		b,
	//		contractAddr,
	//		big.NewInt(0),
	//		payload,
	//	)
	//
	//	//err = cli.SendTransaction(context.Background(), signedTx)
	//	var um types.Transaction
	//	_ = json.Unmarshal(tx, &um)
	//	err = E.SendRawTransaction(&um)
	//	//nonce++
	//
	//	if err != nil {
	//		fmt.Println(err)
	//		zap.L().Error("err", zap.Error(err))
	//		os.Exit(1)
	//	}
	//
	//	for {
	//		r, err := E.PrimaryNode.TransactionReceipt(context.Background(), um.Hash())
	//		if err == nil {
	//			fmt.Println(r)
	//			m, _ := r.MarshalJSON()
	//			fmt.Println(fmt.Sprintf("%s", m))
	//			break
	//		}
	//
	//		if err == ethereum.NotFound {
	//			time.Sleep(1 * time.Second)
	//			continue
	//		} else {
	//			break
	//		}
	//	}
	//}

	////
	////testSize := 100
	////
	////cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/ethereum/ethereum-basic.yaml")
	////if err != nil {
	////	panic(err)
	////}
	////
	////bc, err := parsers.ParseBenchConfig("configurations/workloads/sample/sample_simple.yaml")
	////
	////if err != nil {
	////	panic(err)
	////}
	////
	////var G workloadgenerators.WorkloadGenerator
	////intermediate := workloadgenerators.EthereumWorkloadGenerator{}
	////G = intermediate.NewGenerator(cc, bc)
	////E := clientinterfaces.EthereumInterface{}
	////E.Init(cc.Nodes)
	////err = E.ConnectOne(0)
	////
	////if err != nil {
	////	panic(err)
	////}
	////
	////err = G.BlockchainSetup()
	////if err != nil {
	////	panic(err)
	////}
	////
	////err = G.InitParams()
	////
	////if err != nil {
	////	panic(err)
	////}
	////
	////var workload [][]byte
	////for i := 0; i < testSize; i++ {
	////	bN, _ := big.NewInt(0).SetString("10000000", 10)
	////	txByte, err := G.CreateSignedTransaction(
	////		cc.Keys[0].PrivateKey,
	////		"0x9e3cf23f6fc76b77d2113db93ef388e057c8cc12",
	////		bN,
	////	)
	////	if err != nil {
	////		panic(err)
	////	}
	////	workload = append(workload, txByte)
	////}
	////
	////parsedWorkload, err := E.ParseWorkload(workload)
	////
	////if err != nil {
	////	panic(err)
	////}
	////
	////// startNum, err := E.GetBlockHeight()
	////
	////// if err != nil {
	////// 	panic(err)
	////// }
	////
	////for i := 0; i < len(workload); i++ {
	////	err = E.SendRawTransaction(parsedWorkload[i])
	////	if err != nil {
	////		panic(err)
	////	}
	////}
	////
	////tNow := time.Now()
	////
	////for {
	////	if E.NumTxDone == uint64(len(workload)) {
	////		break
	////	}
	////	if time.Now().Sub(tNow) > 10*time.Second {
	////		break
	////	}
	////
	////	fmt.Printf("Sent: %d, Complete: %d\n", E.NumTxSent, E.NumTxDone)
	////	time.Sleep(1000 * time.Millisecond)
	////}
	////
	////res := E.Cleanup()
	////
	////// endNum, err := E.GetBlockHeight()
	////
	////// if err != nil {
	////// 	panic(err)
	////// }
	////
	////// err = E.ParseBlocksForTransactions(startNum, endNum)
	////
	////// if err != nil {
	////// 	panic(err)
	////// }
	////
	////// fmt.Println(E.Transactions)
	////
	////fmt.Printf("LATENCY: %.2f ms\n", res.AverageLatency)
	////fmt.Printf("Throughput %.2f tps\n", res.Throughput)
	////
	////// for _, v := range E.Transactions {
	////// 	fmt.Println((v[2].Sub(v[0])).Microseconds())
	////// }
	////
	////fmt.Println("DONE, ALL OK")
	////E.Close()

