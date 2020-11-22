package main

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	blockchaintypes "diablo-benchmark/blockchains/types"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/json"
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


	log.Println("sendRawTransaction via client FIRST TIME EXPECTING BUG")
	err = client.SendRawTransaction(createAssetTransaction(0,generator))


	for i := 15; i < 30; i++ {


		log.Println("sendRawTransaction via client for the " + strconv.Itoa(i) + "th time")
		err = client.SendRawTransaction(createAssetTransaction(i,generator))


	}



	log.Println("--> Evaluate Transaction: GetAllAssets, function returns every asset")
	result, err := client.Contract.EvaluateTransaction("GetAllAssets")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v\n", err)
	}
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
	)

	var parsedTxAsset blockchaintypes.FabricTX
	_ = json.Unmarshal(txAsset, &parsedTxAsset)

	return &parsedTxAsset
}

func readAssetTransaction(transactionID int, assetToRead string, generator workloadgenerators.WorkloadGenerator) (*blockchaintypes.FabricTX){

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
		CParamListQuery)

	if err != nil{
		panic(err)
	}


	var parsedTxQuery blockchaintypes.FabricTX
	_ = json.Unmarshal(txQuery,&parsedTxQuery)

	return &parsedTxQuery
}