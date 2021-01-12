package main

import (
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}
const compositeName = "id~op~value"

func (s *SmartContract) Update(ctx contractapi.TransactionContextInterface, id string, delta float64, operation string) error{

	if operation != "+" || operation != "-"{
		return fmt.Errorf("Operation %s is not supported", operation)
	}

	deltaString := fmt.Sprintf("%f", delta)
	compositeKey, err := ctx.GetStub().CreateCompositeKey(compositeName,[]string{id,operation,deltaString})

	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(compositeKey,[]byte{0x00})
}

func (s *SmartContract) Get(ctx contractapi.TransactionContextInterface){


}

func main() {
	assetChaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		log.Panicf("Error creating contention-workaround chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting contention-workaround chaincode: %v", err)
	}
}