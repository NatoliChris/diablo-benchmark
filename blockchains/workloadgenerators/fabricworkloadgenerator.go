package workloadgenerators

import (
	"diablo-benchmark/blockchains/types"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"

	"go.uber.org/zap"
)

// FabricWorkloadGenerator is the workload generator implementation for the Hyperledger Fabric blockchain
type FabricWorkloadGenerator struct {
	BenchConfig *configs.BenchConfig // Benchmark configuration for workload intervals / type
	ChainConfig *configs.ChainConfig // Chain configuration to get number of transactions to make
	GenericWorkloadGenerator
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
	return nil, nil
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
	return nil, nil
}

//CreateInteractionTX main method to create transaction bytes for the workload
func (f FabricWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, functionType string, functionName string, contractParams []configs.ContractParam, value string) ([]byte, error) {

	var tx types.FabricTX
	tx.FunctionType = functionType // "read" or "write" to indicate query or submit
	tx.FunctionName = functionName

	//First argument of contractParams is used as the id for the transaction
	id, err := strconv.Atoi(contractParams[0].Value)
	tx.ID = uint64(id)

	// We don't need the type of the parameters for the transaction
	// in Fabric, so we map ContractsParams to only parameters values
	args := make([]string, 0)
	for _, v := range contractParams[1:] {
		args = append(args, v.Value)
	}

	tx.Args = args

	b, err := json.Marshal(&tx)
	if err != nil {
		return nil, err
	}

	return b, nil
}

//CreateSignedTransaction forms a signed transaction
//and returns bytes to be sent by the 'SendRawTransaction' call.
//(NOT NEEDED IN FABRIC) all signing is done in the client interface
// because users are already defined in the bench config
func (f FabricWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error) {
	return nil, nil
}

//generateTestWorkload generates a test workload given the test benchmark config and the blockchain config files
// returns: Workload ([secondary][threads][time][tx]) -> [][][][]byte
func (f FabricWorkloadGenerator) generateTestWorkload() (Workload, error) {

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
			zap.L().Debug("Info",
				zap.Int("secondary", secondaryID),
				zap.Int("thread", thread))
			for interval, txnum := range f.TPSIntervals {
				// Debug print for each interval to monitor correctness.
				zap.L().Debug("Making workload ",
					zap.Int("secondary", secondaryID),
					zap.Int("thread", thread),
					zap.Int("interval", interval),
					zap.Int("value", txnum))

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

					//function "CreateAsset" and its arguments
					functionToInvoke := f.BenchConfig.ContractInfo.Functions[0]

					// transactions are of the form  (assetID, color, size, owner, price)
					otherParams := functionToInvoke.Params

					// modifying assetID to get a unique transaction
					otherParams[0].Value = strconv.FormatUint(txID, 10)
					params = append(params, otherParams...)

					functionType := f.BenchConfig.ContractInfo.Functions[0].Type //function type gives us whether it a submit or read type transaction
					functionName := functionToInvoke.Name

					// The nil parameter is the key, which is not useful in Fabric
					tx, txerr := f.CreateInteractionTX(nil, functionType, functionName, params, "")

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
func (f FabricWorkloadGenerator) GenerateWorkload() (Workload, error) {

	// 1/ work out the total number of secondaries.
	numberOfWorkingSecondaries := f.BenchConfig.Secondaries * f.BenchConfig.Threads

	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(f.BenchConfig)

	if err != nil {
		return nil, err
	}

	// Total transactions
	totalTx := numberOfTransactions * numberOfWorkingSecondaries

	zap.L().Info(
		"Generating workload",
		zap.String("workloadType", string(f.BenchConfig.TxInfo.TxType)),
		zap.Int("secondaries", numberOfWorkingSecondaries),
		zap.Int("transactionsPerSecondary", numberOfTransactions),
		zap.Int("totalTransactions", totalTx),
	)

	switch f.BenchConfig.TxInfo.TxType {
	//asset transfer/creation/deletion based workload
	case configs.TxTypeTest:
		return f.generateTestWorkload()

	default:
		return nil, errors.New("unknown transaction type in config for workload generation")
	}

}
