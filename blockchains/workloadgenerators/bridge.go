package workloadgenerators


import (
	"math/big"

	"diablo-benchmark/blockchains"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/workload"
)


type ControllerBridge struct {
	GenericWorkloadGenerator
	chainConfig  *configs.ChainConfig
	benchConfig  *configs.BenchConfig
	inner        blockchain.Controller
}

func NewControllerBridge(inner blockchain.Controller) *ControllerBridge {
	return &ControllerBridge{
		inner:  inner,
	}
}

func (this *ControllerBridge) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	return &ControllerBridge{
		chainConfig:  chainConfig,
		benchConfig:  benchConfig,
		inner:        this.inner,
	}
}

func (this *ControllerBridge) BlockchainSetup() error {
	return nil
}

func (this *ControllerBridge) InitParams() error {
	return nil
}

func (this *ControllerBridge) CreateAccount() (interface{}, error) {
	return nil, nil
}

func (this *ControllerBridge) DeployContract(fromPrivKey []byte, contractPath string) (string, error) {
	return "", nil
}

func (this *ControllerBridge) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {
	return nil, nil
}

func (this *ControllerBridge) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam, value string) ([]byte, error) {
	return nil, nil
}

func (this *ControllerBridge) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error) {
	return nil, nil
}

func (this *ControllerBridge) GenerateWorkload() (Workload, error) {
	var wl *workload.Workload
	var err error

	err = this.inner.Init(this.chainConfig, this.benchConfig,
		this.TPSIntervals)
	if err != nil {
		return nil, err
	}

	err = this.inner.Setup()
	if err != nil {
		return nil, err
	}

	wl, err = this.inner.Generate()
	if err != nil {
		return nil, err
	}

	return wl.BuildFlat(), nil
}
