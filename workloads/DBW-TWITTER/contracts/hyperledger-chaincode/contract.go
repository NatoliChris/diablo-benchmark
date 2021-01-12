package main

import (
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// MaxLen defines the maximum size of a tweet
const MaxLen = 250

// Tweet function, super simple
func (s *SmartContract) Tweet(ctx contractapi.TransactionContextInterface, message string, owner string) error {

	if len(message) > MaxLen {
		return fmt.Errorf("tweet too large, %v", len(message))
	}

	b := []byte(message)

	return ctx.GetStub().PutState(owner, b)
}

// Main function initialises the smart contract
func main() {
	tweetChaincode, err := contractapi.NewChaincode(new(SmartContract))

	if err != nil {
		log.Panicf("Error creating chaincode: %v", err)
	}

	if err := tweetChaincode.Start(); err != nil {
		log.Panicf("Failed to start tweet chaincode: %v", err)
	}
}
