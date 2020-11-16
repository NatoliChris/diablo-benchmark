package workloadgenerators

import (
	"diablo-benchmark/core/configs"
	"math/big"
)

// FabricWorkloadGenerator is the workload generator implementation for the Hyperledger Fabric blockchain
type FabricWorkloadGenerator struct {
	BenchConfig       *configs.BenchConfig // Benchmark configuration for workload intervals / type
	ChainConfig       *configs.ChainConfig // Chain configuration to get number of transactions to make
}

//NewGenerator returns a new instance of the generator
func (f FabricWorkloadGenerator) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	return &FabricWorkloadGenerator{
		BenchConfig: benchConfig,
		ChainConfig: chainConfig,
	}
}
//BlockchainSetup ,in theory, should create all artifacts and genesis blocks necessary
// and spin up the network
// DISCLAIMER: for now we assume that the fabric network has already been set up before
func (f FabricWorkloadGenerator) BlockchainSetup() error {
	return nil
}
//InitParams sets up any needed parameters not initialized at construction
func (f FabricWorkloadGenerator) InitParams() error {
	return nil
}

//CreateAccount is used to create a generic account
//(NOT NEEDED IN FABRIC) the users are already setup in the inital config
// as Hyperledger Fabric is a permissioned blockchain
func (f FabricWorkloadGenerator) CreateAccount() (interface{}, error) {
	return nil,nil
}

//DeployContract packages and installs the chaincode on the network
//DISCLAIMER: for now we assume that the fabric network has already been set up before
func (f FabricWorkloadGenerator) DeployContract(fromPrivKey []byte, contractPath string) (string, error) {
	return "not implemented", nil
}

//CreateContractDeployTX creates a transaction to deploy the smart contract
//(NOT NEEDED IN FABRIC) contract deployment is not something useful to
// be benchmarked in a Hyperledger Fabric implementation as it is a permissioned
// blockchain and contract deployment is something agreed upon by organisations and
//not done regularly enough to hinder throughput (usually done during while low traffic)
func (f FabricWorkloadGenerator) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {
	return nil,nil
}

//CreateInteractionTX main method to create transaction bytes for the workload
func (f FabricWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam) ([]byte, error) {
	panic("implement me")
}

//CreateSignedTransaction forms a signed transaction
//and returns bytes to be sent by the 'SendRawTransaction' call.
//(NOT NEEDED IN FABRIC) all signing is done in the client interface
// because users are already defined in the bench config
func (f FabricWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error) {
	return nil, nil
}

//GenerateWorkload generates a workload given the benchmark config and the blockchain config filesgi
func (f FabricWorkloadGenerator) GenerateWorkload() (Workload, error) {
	panic("implement me")
}
