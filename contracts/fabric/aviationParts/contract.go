package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const OwnerToIDCompositeName = "owner->id"



// SmartContract provides functions for managing the buying and selling of aircraft parts
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple part
type AircraftPart struct {
	ID             string `json:"ID"`   //format is "PARTn" where n is a positive integer
	Description	   string `json:"description"`
	Certification  string `json:"certification"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}




type PurchaseOrder struct {
	ID              string `json:"ID"`   //format is "ORDERn" where n is a positive integer
	From 			string `json:"from"` //seller name
	To 				string `json:"to"`	// buyer name
	SoldPart 		AircraftPart `json:"soldPart"` // the aircraft part sold in the purchase order
}



//CreatePart issues a new part to the world state with given details.
func (s *SmartContract) CreatePart(ctx contractapi.TransactionContextInterface, id string, description string, certification string,
	owner string, appraisedValue int) error {

	exists, err := s.PartExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the part %s already exists", id)
	}

	asset := AircraftPart{
		ID:             id,
		Description:   description,
		Certification:  certification,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}

	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}


	err = ctx.GetStub().PutState(id, assetJSON)
	if err != nil {
		return err
	}

	//creation of compositeKey, a composite key allows us to quickly query the ledger by the owner name
	ownerPartIDIndexKey,err := ctx.GetStub().CreateCompositeKey(OwnerToIDCompositeName,[]string{owner,id})
	value := []byte{0x00}

	return ctx.GetStub().PutState(ownerPartIDIndexKey,value)
}


//QueryPartByID returns the part stored in the world state with given id.
func (s *SmartContract) QueryPartByID(ctx contractapi.TransactionContextInterface, id string) (*AircraftPart, error) {
	partJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if partJSON == nil {
		return nil, fmt.Errorf("the part %s does not exist", id)
	}

	var part AircraftPart
	err = json.Unmarshal(partJSON, &part)
	if err != nil {
		return nil, err
	}

	return &part, nil
}

//QueryPartsByOwner returns all of the parts tied to the specified owner, using composites keys for fast querying
func (s *SmartContract) QueryPartsByOwner(ctx contractapi.TransactionContextInterface, owner string) ([]*AircraftPart, error){

	compositeKeyIterator, err := ctx.GetStub().GetStateByPartialCompositeKey(OwnerToIDCompositeName, []string{owner})
	if err != nil {
		return nil, err
	}
	defer compositeKeyIterator.Close()

	var parts []*AircraftPart
	for compositeKeyIterator.HasNext() {
		compositeKey, err := compositeKeyIterator.Next()
		if err != nil {
			return nil, err
		}

		// from composite key {owner->id, {owner,id}}, we get owner->id, {owner,id}, err
		_,compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(compositeKey.Key)
		if err != nil {
			return nil, err
		}

		// the aircraft part id associated to the owner
		partID := compositeKeyParts[1]
		// getting the part
		partJSON,err := ctx.GetStub().GetState(partID)
		if err != nil {
			return nil, fmt.Errorf("failed to read from world state: %v", err)
		}

		var part AircraftPart
		err = json.Unmarshal(partJSON, &part)
		if err != nil {
			return nil, err
		}
		parts = append(parts, &part)
	}

	return parts, nil

}

//DeletePart deletes a given part from the world state.
func (s *SmartContract) DeletePart(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.PartExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the part %s does not exist", id)
	}

	partJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}

	var part AircraftPart
	err = json.Unmarshal(partJSON,&part)
	if err != nil {
		return err
	}

	// deleting the composite key
	ownerPartIDIndexKey,err := ctx.GetStub().CreateCompositeKey(OwnerToIDCompositeName,[]string{part.Owner,part.ID})

	err = ctx.GetStub().DelState(ownerPartIDIndexKey)
	if err != nil {
		return err
	}

	return ctx.GetStub().DelState(id)
}

//PartExists returns true when a part with given ID exists in world state
func (s *SmartContract) PartExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	partJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return partJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state.
func (s *SmartContract) TransferPart(ctx contractapi.TransactionContextInterface, partID string, purchaseOrderID string,newOwner string) error {
	part, err := s.QueryPartByID(ctx, partID)
	if err != nil {
		return err
	}

	//creating the purchase order
	purchaseOrder := PurchaseOrder{
		ID:       purchaseOrderID,
		From:     part.Owner,
		To:       newOwner,
		SoldPart: *part,
	}

	//putting the purchaseOrder
	purchaseOrderJSON, err := json.Marshal(purchaseOrder)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(purchaseOrderID,purchaseOrderJSON)
	if err != nil {
		return err
	}

	//updating the composite key

	//deleting it
	ownerPartIDIndexKey,err := ctx.GetStub().CreateCompositeKey(OwnerToIDCompositeName,[]string{part.Owner,part.ID})
	if err != nil {
		return err
	}
	err = ctx.GetStub().DelState(ownerPartIDIndexKey)
	if err != nil {
		return err
	}

	// putting the new in
	ownerPartIDIndexKey,err = ctx.GetStub().CreateCompositeKey(OwnerToIDCompositeName,[]string{newOwner,part.ID})
	if err != nil {
		return err
	}
	value := []byte{0x00}
	err = ctx.GetStub().PutState(ownerPartIDIndexKey,value)
	if err != nil {
		return err
	}


	// putting the updated part in the state
	part.Owner = newOwner
	assetJSON, err := json.Marshal(part)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(partID, assetJSON)
}

// GetAllAssets returns all the parts found in world state
func (s *SmartContract) GetAllParts(ctx contractapi.TransactionContextInterface) ([]*AircraftPart, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all parts in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var parts []*AircraftPart
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var part AircraftPart
		err = json.Unmarshal(queryResponse.Value, &part)
		if err != nil {
			return nil, err
		}
		parts = append(parts, &part)
	}

	return parts, nil
}


func main() {
	assetChaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		log.Panicf("Error creating aviationParts chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting aviationParts chaincode: %v", err)
	}
}