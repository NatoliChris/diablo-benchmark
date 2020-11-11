package workloadgenerators

import (
	"diablo-benchmark/core/configs"
	"math/big"
)

// FabricWorkloadGenerator is the workload generator implementation for the Hyperledger Fabric blockchain
type FabricWorkloadGenerator struct {

}

func (f FabricWorkloadGenerator) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	panic("implement me")
}

func (f FabricWorkloadGenerator) BlockchainSetup() error {
	panic("implement me")
}

func (f FabricWorkloadGenerator) InitParams() error {
	panic("implement me")
}

func (f FabricWorkloadGenerator) CreateAccount() (interface{}, error) {
	panic("implement me")
}

func (f FabricWorkloadGenerator) DeployContract(fromPrivKey []byte, contractPath string) (string, error) {
	panic("implement me")
}

func (f FabricWorkloadGenerator) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {
	panic("implement me")
}

func (f FabricWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam) ([]byte, error) {
	panic("implement me")
}

func (f FabricWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error) {
	panic("implement me")
}

func (f FabricWorkloadGenerator) GenerateWorkload() (Workload, error) {
	panic("implement me")
}
