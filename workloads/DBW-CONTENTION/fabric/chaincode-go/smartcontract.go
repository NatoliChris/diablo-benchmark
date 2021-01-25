package main

import (
	"encoding/json"
	"fmt"
	"log"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

type Asset struct {
	ID             string `json:"ID"`
	Value int64    `json:"Value"`
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, value int64) error {
	temp, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}

	if temp!=nil {
		return fmt.Errorf("the asset %s already exists", id)
	}

	asset := Asset{
		ID: id,
		Value: value,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, value int64) error {

	temp, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}

	if temp==nil {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	// overwriting original asset with new asset
	asset := Asset{
		ID:             id,
		Value: value,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

func main() {
	assetChaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		log.Panicf("Error creating asset-transfer-basic chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting asset-transfer-basic chaincode: %v", err)
	}
}