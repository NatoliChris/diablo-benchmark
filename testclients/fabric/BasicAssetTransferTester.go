package main

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	blockchaintypes "diablo-benchmark/blockchains/types"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"strconv"
)

func main(){
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)

	cc, err := parsers.ParseChainConfig("configurations/blockchain-configs/fabric/fabric-test.yaml")
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
	client1 := clientinterfaces.FabricInterface{}
	//client2 := clientinterfaces.FabricInterface{}

	log.Println("Init client1 interface")
	if cc.Extra == nil{
		fmt.Println("EXTRA IS NIL")
		return
	}

	client1.Init(cc)
	client1.SetWindow(2)
	//client2.Init(cc.Nodes, nil)
	client1.Start()


	err = generator.BlockchainSetup()
	if err != nil {
		panic(err)
	}

	err = generator.InitParams()

	if err != nil {
		panic(err)
	}


	//log.Println("sendRawTransaction via client1 FIRST TIME EXPECTING BUG")
	//err = client1.SendRawTransaction(createAssetTransaction(0,generator))
	////err = client2.SendRawTransaction(createAssetTransaction(0,generator))
//
	//workload,err := generator.GenerateWorkload()
//
	//if err != nil {
	//	panic(err)
	//}
//
	//parsedWorkload1,err := client1.ParseWorkload(workload[0][0])
//
	//if err != nil {
	//	panic(err)
	//}
//
//
	//for _,intervals := range parsedWorkload1 {
	//	for _, tx := range intervals {
	//		client1.SendRawTransaction(tx)
	//	}
	//}
//
	//parsedWorkload2,err := client2.ParseWorkload(workload[0][1])
	//	for _,intervals := range parsedWorkload2 {
	//		for _, tx := range intervals{
	//		client2.SendRawTransaction(tx)
	//	}
	//}

	//var txs []*gateway.Transaction
	//var listeners [] <- chan *fab.TxStatusEvent
//
	//for i := 0; i < 100; i++ {
	//	tx,err := client1.Contract.CreateTransaction("CreateAsset")
//
	//	if err != nil {
	//		log.Println(err)
	//	}
	//	ls := tx.RegisterCommitEvent()
//
	//	txs = append(txs,tx)
	//	listeners = append(listeners,ls)
	//}
//
	//for i, tx := range txs {
	//	s := strconv.FormatInt(int64(i),10)
	//	go tx.Submit(s,s,s,s,s)
	//}
//
	//for _,ls := range listeners {
	//	s := <- ls
	//	log.Println(s.SourceURL)
	//	log.Println(s.TxValidationCode.String())
	//	log.Println("block",s.BlockNumber)
	//}
//
//
	//log.Println("submitted transaction")


	log.Println("--> Evaluate Transaction: GetAllAssets, function returns every asset")
	result, err := client1.Contract.EvaluateTransaction("GetAllAssetsID")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println("ALL TRANSACTIONS IDS IN THE LEDGER")
	log.Println(string(result))

}

func createAssetTransaction(transactionID int, generator workloadgenerators.WorkloadGenerator) *blockchaintypes.FabricTX {

	var cParamList []configs.ContractParam

	cParamList = append(cParamList,
		configs.ContractParam{
			Type:  "uint64",
			Value: strconv.Itoa(transactionID),
		},
		configs.ContractParam{
			Type:  "string",
			Value: "asset#" + strconv.Itoa(transactionID),
		},
		configs.ContractParam{
			Type:  "color",
			Value: "c",
		}, configs.ContractParam{
			Type:  "size",
			Value: "100",
		}, configs.ContractParam{
			Type:  "owner",
			Value: "Bob Ross",
		},configs.ContractParam{
			Type:  "price",
			Value: "420",
		})


	txAsset,_ := generator.CreateInteractionTX(
		nil,
		"write",
		"CreateAsset",
		cParamList,
		"",
	)

	var parsedTxAsset blockchaintypes.FabricTX
	_ = json.Unmarshal(txAsset, &parsedTxAsset)

	return &parsedTxAsset
}

func readAssetTransaction(transactionID int, assetToRead string, generator workloadgenerators.WorkloadGenerator) *blockchaintypes.FabricTX {

	var CParamListQuery []configs.ContractParam

	CParamListQuery = append(CParamListQuery,	configs.ContractParam{
		Type:  "uint64",
		Value: strconv.Itoa(transactionID),
	}, configs.ContractParam{
		Type:  "string",
		Value: assetToRead,
	})



	txQuery, err := generator.CreateInteractionTX(
		nil,
		"read",
		"ReadAsset",
		CParamListQuery,
		"")

	if err != nil{
		panic(err)
	}


	var parsedTxQuery blockchaintypes.FabricTX
	_ = json.Unmarshal(txQuery,&parsedTxQuery)

	return &parsedTxQuery
}