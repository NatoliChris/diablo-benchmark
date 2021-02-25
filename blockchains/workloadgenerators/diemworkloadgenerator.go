package workloadgenerators

import (
	"diablo-benchmark/blockchains/types"
	"diablo-benchmark/core/configs"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

//NewGenerator returns a new instance of the generator
type DiemWorkloadGenerator struct {
	BenchConfig *configs.BenchConfig // Benchmark configuration for workload intervals / type
	ChainConfig *configs.ChainConfig // Chain configuration to get number of transactions to make
	GenericWorkloadGenerator
}

//NewGenerator returns a new instance of the generator
func (f DiemWorkloadGenerator) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	return &DiemWorkloadGenerator{
		BenchConfig: benchConfig,
		ChainConfig: chainConfig,
	}
}
// BlockchainSetup
// Does not do anything for now. Assume the blockchain is already setup before the test
func (f DiemWorkloadGenerator) BlockchainSetup() error {
	return nil
}

//InitParams sets up any needed parameters not initialized at construction
func (f DiemWorkloadGenerator) InitParams() error {
	return nil
}

//CreateAccount is used to create a generic account
//(NOT NEEDED IN Diem) the users are already setup in the initial config
// as Diem is a permissioned blockchain
func (f DiemWorkloadGenerator) CreateAccount() (interface{}, error) {
	return nil, nil
}

//DeployContract packages and installs the chaincode on the network
//DISCLAIMER: for now we assume that the Diem network has already been set up before
func (f DiemWorkloadGenerator) DeployContract(fromPrivKey []byte, contractPath string) (string, error) {
	return "not implemented", nil
}

//CreateContractDeployTX creates a transaction to deploy the smart contract
//(NOT NEEDED IN Diem) contract deployment is not something useful to
// be benchmarked in Diem for similar reasons as Hyperledger Fabric
func (f DiemWorkloadGenerator) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {
	return nil, nil
}
// CreateInteractionTX main method to create transaction bytes for the workload
// TODO implement basic interactive transaction
func (f DiemWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, functionType string, functionName string, contractParams []configs.ContractParam, value string) ([]byte, error){
	var tx types.DiemTX
	tx.FunctionType = functionType
	tx.Name = functionName
	tx.Path = f.BenchConfig.Path
	args := make([]string, 0)
	for _, v := range contractParams[1:] {
		args = append(args, v.Value)
	}
	tx.Args = args
	fmt.Println(tx)
	bytes, err := json.Marshal(&tx)
	if err != nil {
		panic(err)
	}
	return bytes, nil
}

// Not needed
func (f DiemWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error){
	return nil, nil
}

func (f DiemWorkloadGenerator) GenerateSimpleWorkload() (Workload, error) {
	var totalWorkload Workload
	// 1. Generate the transactions
	txID := uint64(0)
	accountBatch := 0
	for secondaryID := 0; secondaryID < f.BenchConfig.Secondaries; secondaryID++ {
		// secondaryWorkload = [thread][interval][tx=[]byte]
		// [][][][]byte
		secondaryWorkload := make(SecondaryWorkload, 0)
		for thread := 0; thread < f.BenchConfig.Threads; thread++ {
			// Thread workload = list of transactions in intervals
			// [interval][tx] = [][][]byte

			threadWorkload := make(WorkerThreadWorkload, 0)
			// for each thread, generate the intervals of transactions.
			for _, txnum := range f.BenchConfig.TxInfo.Intervals {
				// Time interval = list of transactions
				// [tx] = [][]byte
				intervalWorkload := make([][]byte, 0)
				for txIt := 0; txIt < txnum; txIt++ {
					var params = make([]configs.ContractParam, 0)
					//creating the id for the transaction
					id := strconv.FormatUint(txID, 10)
					params = append(params, configs.ContractParam{
						Type:  "uint64",
						Value: id,
					})
					//function "transfer" and its arguments
					functionToInvoke := f.BenchConfig.ContractInfo.Functions[0]
					// transactions are of the form  (assetID, color, size, owner, price)
					otherParams := functionToInvoke.Params
					// modifying assetID to get a unique transaction
					otherParams[0].Value = strconv.FormatUint(txID, 10)
					params = append(params, otherParams...)
					functionType := f.BenchConfig.ContractInfo.Functions[0].Type //function type gives us whether it a submit or read type transaction
					name := functionToInvoke.Name
					// The nil parameter is the key, which is not useful in Fabric
					tx, txerr := f.CreateInteractionTX(nil, functionType, name, params, "")
					if txerr != nil {
						return nil, txerr
					}
					intervalWorkload = append(intervalWorkload, tx)
					txID++
				}
				threadWorkload = append(threadWorkload, intervalWorkload)
			}
			secondaryWorkload = append(secondaryWorkload, threadWorkload)
			accountBatch++
		}
		totalWorkload = append(totalWorkload, secondaryWorkload)
	}
	return totalWorkload, nil
}

//GenerateWorkload generates a workload given the benchmark config and the blockchain config files
// returns: Workload ([secondary][threads][time][tx]) -> [][][][]byte
func (f DiemWorkloadGenerator) GenerateWorkload() (Workload, error) {
	return f.GenerateSimpleWorkload()
}