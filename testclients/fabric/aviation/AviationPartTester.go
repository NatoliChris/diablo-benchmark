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

	cc, err := parsers.ParseChainConfig("workloads/aviationParts/configurations/aviationChainConfig.yaml")
	if err != nil {
		panic(err)
	}

	//bc, err := parsers.ParseBenchConfig("configurations/workloads/fabric/testDiabloFabric.yaml")
//
	//if err != nil {
	//	panic(err)
	//}


	var generator workloadgenerators.WorkloadGenerator

	intermediate := workloadgenerators.FabricWorkloadGenerator{}

	generator = intermediate.NewGenerator(cc, nil)
	client1 := clientinterfaces.FabricInterface{}
	//client2 := clientinterfaces.FabricInterface{}

	log.Println("Init client1 interface")
	if cc.Extra == nil{
		fmt.Println("EXTRA IS NIL")
		return
	}

	client1.Init(cc)
	//client2.Init(cc.Nodes, nil)


	//err = generator.BlockchainSetup()
	//if err != nil {
	//	panic(err)
	//}
//
	//err = generator.InitParams()
//
	//if err != nil {
	//	panic(err)
	//}


	log.Println("sendRawTransaction via client1 FIRST TIME EXPECTING BUG")
	err = client1.SendRawTransaction(createPartTransaction(0,"Alice",generator))

	for i := 1; i < 10; i++ {
		if i%2 == 0 {
			client1.SendRawTransaction(createPartTransaction(i,"Alice",generator))
		}else {
			client1.SendRawTransaction(createPartTransaction(i,"Bob",generator))
		}

	}

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


	log.Println("--> Evaluate Transaction: GetAllParts, function returns every part")
	result, err := client1.Contract.EvaluateTransaction("GetAllParts")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println("ALL PARTS IN THE LEDGER")
	log.Println(string(result))

	log.Println("-----------------------------------------------------------")


	log.Println("--> Evaluate Transaction: GetQueryPartsByOwner, function returns every part owned by ALice")
	result, err = client1.Contract.EvaluateTransaction("QueryPartsByOwner", "Alice")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println(string(result))

	log.Println("-----------------------------------------------------------")


	log.Println("--> Evaluate Transaction: GetQueryPartsByOwner, function returns every part owned by Bob")
	result, err = client1.Contract.EvaluateTransaction("QueryPartsByOwner", "Bob")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println(string(result))

	log.Println("-----------------------------------------------------------")

	log.Println("--> SubmitTransaction : TransferPart, transfering part#1 from Bob to Alice and checking if a purchase order is made !")
	result, err = client1.Contract.SubmitTransaction("TransferPart", "part#1", "purchaseOrder#0", "Alice")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println(string(result))

	log.Println("-----------------------------------------------------------")


	log.Println("--> Evaluate Transaction: GetAllParts, function returns every part")
	result, err = client1.Contract.EvaluateTransaction("GetAllParts")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println("ALL PARTS IN THE LEDGER")
	log.Println(string(result))

	log.Println("-----------------------------------------------------------")


	log.Println("--> Evaluate Transaction: GetPurchaseOrderByID, function returns every part")
	result, err = client1.Contract.EvaluateTransaction("QueryPurchaseOrderByID", "purchaseOrder#0")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
	log.Println("ALL PARTS IN THE LEDGER")
	log.Println(string(result))




}

func createPartTransaction(transactionID int, owner string, generator workloadgenerators.WorkloadGenerator) *blockchaintypes.FabricTX {

	var cParamList []configs.ContractParam

	cParamList = append(cParamList,
		configs.ContractParam{
			Type:  "ID for FabricTx",
			Value: strconv.Itoa(transactionID),
		},
		configs.ContractParam{
			Type:  "ID of the part",
			Value: "part#" + strconv.Itoa(transactionID),
		},
		configs.ContractParam{
			Type:  "description",
			Value: "",
		}, configs.ContractParam{
			Type:  "certification",
			Value: "",
		}, configs.ContractParam{
			Type:  "owner",
			Value: owner,
		},configs.ContractParam{
			Type:  "price",
			Value: "0",
		})

	txAsset,_ := generator.CreateInteractionTX(
		nil,
		"write",
		"CreatePart",
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
