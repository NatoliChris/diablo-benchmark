package main

import (
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"log"
	"strconv"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}
const compositeName = "id~op~value"

func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, delta int64, operation string) error{

	if operation != "+" && operation != "-"{
		return fmt.Errorf("operation %s is not supported", operation)
	}

	deltaString := strconv.FormatInt(delta,10)
	compositeKey, err := ctx.GetStub().CreateCompositeKey(compositeName,[]string{id,operation,deltaString})

	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(compositeKey,[]byte{0x00})
}

func (s *SmartContract) Get(ctx contractapi.TransactionContextInterface, id string) error{
	compositeKeyIterator,err := ctx.GetStub().GetStateByPartialCompositeKey(compositeName, []string{id})

	if err != nil {
		return err
	}
	defer compositeKeyIterator.Close()

	if !compositeKeyIterator.HasNext(){
		return fmt.Errorf("No asset with id %s exists", id)
	}

	var finalValue int64

	for compositeKeyIterator.HasNext() {
		compositeKey, err := compositeKeyIterator.Next()
		if err != nil {
			return err
		}

		// from composite key {id~op~value, {id,op,value}}, we get id~op~value, {id,op,value}, err
		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(compositeKey.Key)
		if err != nil {
			return err
		}

		op := compositeKeyParts[1]
		deltaString := compositeKeyParts[2]

		delta,err := strconv.ParseInt(deltaString,10,64)
		if err != nil {
			return err
		}

		switch op {
		case "+":
			finalValue += delta
		case "-":
			finalValue -= delta
		default:
			return fmt.Errorf("operation %s is not supported", op)
		}

		err = ctx.GetStub().DelState(compositeKey.Key)
		if err != nil {
			return fmt.Errorf("could not delete composite key out of state, error is : %s",err)
		}

	}

	//updating the ledger with the final value
	return s.UpdateAsset(ctx, id, finalValue, "+")
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		log.Panicf("Error creating contention-workaround chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting contention-workaround chaincode: %v", err)
	}
}