package main

import (
	"fmt"
	"log"
	"math/big"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing the exchange
type SmartContract struct {
	contractapi.Contract
}

var availableStocks [10]string = [10]string{
	"AMD",
	"TSLA",
	"AMZN",
	"AAPL",
	"ZNGA",
	"NVDA",
	"MSFT",
	"JD",
	"CSCO",
	"FB",
}

func (s *SmartContract) checkStock(ctx contractapi.TransactionContextInterface, stock string, amount uint) error {
	data, err := ctx.GetStub().GetState(stock)

	if err != nil {
		return fmt.Errorf("Failed to read from world state: %v", err)
	}

	data_int := big.NewInt(0).SetBytes(data)

	if data_int.Cmp(big.NewInt(int64(amount))) < 0 {
		return fmt.Errorf("Too much of stock %v requested", stock)
	}

	return nil
}

func (s *SmartContract) Buy(ctx contractapi.TransactionContextInterface, stock string, value uint) error {
	isValid := false
	for _, val := range availableStocks {
		if val == stock {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("%v is not a stock", stock)
	}

	err := s.checkStock(ctx, stock, value)

	if err != nil {
		return err
	}

	for i := uint(0); i < value; i++ {
		data, err := ctx.GetStub().GetState(stock)

		if err != nil {
			return fmt.Errorf("Failed to read from world state: %v", err)
		}

		data_int := big.NewInt(0).SetBytes(data)

		data_int.Sub(data_int, big.NewInt(1))

		err = ctx.GetStub().PutState(stock, data_int.Bytes())

		if err != nil {
			return fmt.Errorf("Failed to set state: %v", err)
		}
	}

	return nil
}

func (s *SmartContract) Init(ctx contractapi.TransactionContextInterface) error {
	for _, v := range availableStocks {
		b := big.NewInt(1000000).Bytes()
		err := ctx.GetStub().PutState(v, b)
		if err != nil {
			return fmt.Errorf("Failed to initialise stock: %v", err)
		}
	}

	return nil
}

func main() {
	exchangeChaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		log.Panicf("Error creating exchange chaincode: %v", err)
	}

	if err := exchangeChaincode.Start(); err != nil {
		log.Panicf("Error starting exchange chaincode: %v", err)
	}
}
