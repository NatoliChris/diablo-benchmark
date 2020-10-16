package workloadgenerators

import (
	"bytes"
	"context"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// EthereumWorkloadGenerator is the workload generator implementation for the Ethereum blockchain
type EthereumWorkloadGenerator struct {
	ActiveConn        *ethclient.Client    // Active connection to a blockchain node for information
	SuggestedGasPrice *big.Int             // Suggested gas price on the network
	BenchConfig       *configs.BenchConfig // Benchmark configuration for workload intervals / type
	ChainConfig       *configs.ChainConfig // Chain configuration to get number of transactions to make
	Nonces            map[string]uint64    // Nonce of the known accounts
	ChainID           *big.Int             // ChainID for transactions, provided through the ethereum API
	KnownAccounts     []configs.ChainKey   // Known accounds, public:private key pair
	CompiledContract  *compiler.Contract   // Compiled contract bytecode for the contract used in complex workloads
}

// NewGenerator returns a new instance of the generator
func (e *EthereumWorkloadGenerator) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	return &EthereumWorkloadGenerator{BenchConfig: benchConfig, ChainConfig: chainConfig}
}

// BlockchainSetup sets up the blockchain nodes with relevant information.
// This is the function that can be used to create and generate a genesis block
// as well as deliver the genesis block to the blockchain nodes and run the
// setup command. By the end of this function, there should be:
//  * Blockchain network of nodes running the blockchain
//  * Valid genesis block running on the blockchains
//  * List of accounts that are funded and known
//
// The main aspect of the blockchain setup is to provide a step to start the
// blockchain nodes
func (e *EthereumWorkloadGenerator) BlockchainSetup() error {
	// TODO implement
	// 1 - create N accounts only if we don't have accounts
	if len(e.ChainConfig.Keys) > 0 {
		e.KnownAccounts = e.ChainConfig.Keys
		return nil
	}
	// 2 - fund with genesis block, write to genesis location
	// 3 - copy genesis to blockchain nodes
	return nil
}

// InitParams sets initial aspects such as the suggested gas price and sets up a small connection to get information from the blockchain.
func (e *EthereumWorkloadGenerator) InitParams() error {

	// Connect to the blockchain
	c, err := ethclient.Dial(fmt.Sprintf("ws://%s", e.ChainConfig.Nodes[0]))

	if err != nil {
		return err
	}

	e.ActiveConn = c

	// Get the suggested gas price from the network using a client connected
	e.SuggestedGasPrice, err = e.ActiveConn.SuggestGasPrice(context.Background())

	if err != nil {
		return err
	}

	// Chain ID
	chainID, err := e.ActiveConn.NetworkID(context.Background())
	if err != nil {
		return err
	}
	e.ChainID = chainID

	// nonces
	e.Nonces = make(map[string]uint64, 0)

	for _, key := range e.KnownAccounts {
		v, err := e.ActiveConn.PendingNonceAt(context.Background(), common.HexToAddress(key.Address))
		if err != nil {
			return err
		}

		e.Nonces[strings.ToLower(key.Address)] = v
	}

	zap.L().Info("Blockchain client contacted and got params",
		zap.String("gasPrice", e.SuggestedGasPrice.String()),
		zap.String("chainID", e.ChainID.String()))

	zap.L().Debug("nonces", zap.String("noncemap", fmt.Sprintf("%v", e.Nonces)))

	return nil
}

// CreateAccount is used as a generic account creation to return the private key
func (e *EthereumWorkloadGenerator) CreateAccount() (interface{}, error) {
	// Generate a private key
	privKey, err := crypto.GenerateKey()

	if err != nil {
		return nil, err
	}

	return privKey, nil
}

// DeployContract deploys the contract and returns the address
func (e *EthereumWorkloadGenerator) DeployContract(fromPivKey []byte, contractPath string) (string, error) {
	tx, err := e.CreateContractDeployTX(fromPivKey, contractPath)
	if err != nil {
		return "", err
	}

	// Convert back to the transaction type
	var parsedTx types.Transaction
	err = json.Unmarshal(tx, &parsedTx)
	if err != nil {
		return "", err
	}

	// Deploy the transaction
	err = e.ActiveConn.SendTransaction(context.Background(), &parsedTx)
	if err != nil {
		return "", err
	}

	// Wait for the transaction information to come through with the
	// transaction receipt
	for {
		time.Sleep(1 * time.Second)

		txReceipt, err := e.ActiveConn.TransactionReceipt(context.Background(), parsedTx.Hash())

		if err == nil {
			// No error, return the receipt
			return txReceipt.ContractAddress.String(), nil
		}
		if err == ethereum.NotFound {
			continue
		} else {
			return "", err
		}
	}
}

// CreateContractDeployTX creates a transaction to deploy the smart contract
func (e *EthereumWorkloadGenerator) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {

	// Generate the relevant account information from the private key
	priv, err := crypto.HexToECDSA(hex.EncodeToString(fromPrivKey))
	if err != nil {
		return []byte{}, err
	}

	addrFrom := crypto.PubkeyToAddress(priv.PublicKey)

	// Check for the existence of the contract
	if _, err := os.Stat(contractPath); err == nil {
		// Path exists, compile the contract and prepare the transaction
		// TODO: check the 'solc' string
		contracts, err := compiler.CompileSolidity("", contractPath)
		if err != nil {
			return []byte{}, err
		}

		// TODO handle case where number of contracts is greater than one
		if len(contracts) > 1 {
			zap.L().Warn("multiple contracts compiled, only deploying first")
		}

		for k, v := range contracts {
			zap.L().Info("contract deploy transaction",
				zap.String("contract", k))

			bytecodeBytes, err := hex.DecodeString(v.Code[2:])

			if err != nil {
				return []byte{}, err
			}

			// TODO maybe estimate gas rather than have an upper bound
			gasLimit := uint64(300000)

			zap.L().Debug("tx params",
				zap.String("from", addrFrom.String()),
				zap.Uint64("Nonce", e.Nonces[strings.ToLower(addrFrom.String())]),
				zap.Uint64("gaslimit", gasLimit),
			)
			tx := types.NewContractCreation(
				e.Nonces[strings.ToLower(addrFrom.String())],
				big.NewInt(0),
				gasLimit,
				e.SuggestedGasPrice,
				bytecodeBytes,
			)
			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.ChainID), priv)

			// Update nonce
			e.Nonces[strings.ToLower(addrFrom.String())]++
			e.CompiledContract = v

			return signedTx.MarshalJSON()
		}

	} else if os.IsNotExist(err) {
		// Path doesn't exist - return an error
		return []byte{}, fmt.Errorf("contract does not exist: %s", contractPath)
	} else {
		// Corner case, it's another error - so we should handle it
		// like an error state
		return []byte{}, err
	}

	return []byte{}, errors.New("failed to create deploy tx")
}

// CreateInteractionTX forms a transaction that invokes a smart contract
func (e *EthereumWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam) ([]byte, error) {
	// Check that the contract has been compiled, if nto - then it's difficult to get the hashes from the ABI.
	if e.CompiledContract == nil {
		return nil, fmt.Errorf("contract does not exist in known generator")
	}

	if len(contractParams) < 1 {
		// empty
		return nil, fmt.Errorf("empty contract params for %s", functionName)
	}

	// next - get the function hash
	var funcHash string
	val, ok := e.CompiledContract.Hashes[functionName]
	if !ok {
		return nil, fmt.Errorf("contract does not contain function: %s", functionName)
	}
	funcHash = val

	// Now we need to parse the arguments to get them into the correct padding
	payloadBytes, err := hex.DecodeString(funcHash)
	if err != nil {
		return nil, err
	}

	// Then go through and convert the params
	// Types taken from: https://solidity.readthedocs.io/en/develop/abi-spec.html#types
	// NOTE: need to pad to 32 bytes
	for _, v := range contractParams {
		switch v.Type {
		// uints
		case "uint8":
			// uint 8 = 1 byte
			// padding = 31 bytes
			num, err := strconv.ParseUint(v.Value, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 31)
			payloadBytes = append(payloadBytes, padding...)
			payloadBytes = append(payloadBytes, uint8(num))
			break
		case "uint32":
			// uint 32 = 4 bytes
			// padding = 28 bytes
			num, err := strconv.ParseUint(v.Value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 28)
			payloadBytes = append(payloadBytes, padding...)
			numBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(numBytes, uint32(num))
			payloadBytes = append(payloadBytes, numBytes...)
			break
		case "uint64":
			// uint 64 = 8 bytes
			// padding = 24 bytes
			num, err := strconv.ParseUint(v.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 24)
			payloadBytes = append(payloadBytes, padding...)
			numBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(numBytes, num)
			payloadBytes = append(payloadBytes, numBytes...)
			break
		case "uint256", "uint":
			// uint 256 = 64 bytes
			//  padding = 0
			num, ok := big.NewInt(0).SetString(v.Value, 10)
			if !ok {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			payloadBytes = append(payloadBytes, num.Bytes()...)
			break
		// ints
		case "int8":
			// int 8 = 1 byte
			// padding = 31 bytes
			num, err := strconv.ParseInt(v.Value, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 31)
			payloadBytes = append(payloadBytes, padding...)
			buf := new(bytes.Buffer)
			err = binary.Write(buf, binary.BigEndian, int8(num))
			if err != nil {
				return nil, err
			}
			payloadBytes = append(payloadBytes, buf.Bytes()...)
			break
		case "int32":
			// int 32 = 4 bytes
			// padding = 28 bytes
			num, err := strconv.ParseInt(v.Value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 28)
			payloadBytes = append(payloadBytes, padding...)
			buf := new(bytes.Buffer)
			err = binary.Write(buf, binary.BigEndian, int32(num))
			if err != nil {
				return nil, err
			}
			payloadBytes = append(payloadBytes, buf.Bytes()...)
			break
		case "int64":
			// int 32 = 4 bytes
			// padding = 28 bytes
			num, err := strconv.ParseInt(v.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 24)
			payloadBytes = append(payloadBytes, padding...)
			buf := new(bytes.Buffer)
			err = binary.Write(buf, binary.BigEndian, num)
			if err != nil {
				return nil, err
			}
			payloadBytes = append(payloadBytes, buf.Bytes()...)
			break
		case "int256", "int":
			num, ok := big.NewInt(0).SetString(v.Value, 10)
			if !ok {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			payloadBytes = append(payloadBytes, num.Bytes()...)
			break
		// bool
		case "bool":
			// Bool is just a padded uint8 of value 0 or 1
			var bVal uint8
			if v.Value == "true" {
				bVal = 1
			} else if v.Value == "false" {
				bVal = 0
			} else {
				return nil, fmt.Errorf("failed to convert contract arg %s into %s", v.Value, v.Type)
			}
			padding := make([]byte, 31)
			payloadBytes = append(payloadBytes, padding...)
			payloadBytes = append(payloadBytes, bVal)
			break
		// address
		case "address":
			// uint160
			// get the address
			addr := common.HexToAddress(v.Value)
			// padding - address bytes should be 20bytes long.
			padding := make([]byte, 12)
			payloadBytes = append(payloadBytes, padding...)
			payloadBytes = append(payloadBytes, addr.Bytes()...)
			break
		// bytes
		case "bytes24":
			// TODO this needs improvement!
			s := []byte(v.Value)
			padding := make([]byte, 32-len(s))
			payloadBytes = append(payloadBytes, s...)
			payloadBytes = append(payloadBytes, padding...)
			break
		case "bytes32":
			// TODO this needs improvement!
			s := []byte(v.Value)
			payloadBytes = append(payloadBytes, s...)
			break
			// Default
		default:
			return nil, fmt.Errorf("invalid arg type: %s", v.Type)
		}
	}

	// Assume that the payload bytes have been correctly formed at this point?
	if len(payloadBytes) < 1 {
		return nil, fmt.Errorf("no payload generated")
	}

	// Create the signed transaction
	tx, err := e.CreateSignedTransaction(fromPrivKey, contractAddress, big.NewInt(0), payloadBytes)

	if err != nil {
		return nil, err
	}

	// return the transaction
	return tx, nil
}

// CreateSignedTransaction forms a signed transaction and returns bytes to be sent by the 'SendRawTransaction' call.
func (e *EthereumWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error) {

	// Get the private key
	priv, err := crypto.HexToECDSA(hex.EncodeToString(fromPrivKey))

	if err != nil {
		return []byte{}, err
	}

	// Get the address from the private key
	addrFrom := crypto.PubkeyToAddress(priv.PublicKey)

	// Get the transaction fields
	toConverted := common.HexToAddress(toAddress)
	gasLimit := uint64(300000)

	zap.L().Debug("transaction params",
		zap.String("addrFrom", addrFrom.String()),
		zap.String("addrTo", toAddress),
	)

	// Make and sign the transaction
	tx := types.NewTransaction(e.Nonces[strings.ToLower(addrFrom.String())], toConverted, value, gasLimit, e.SuggestedGasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.ChainID), priv)
	if err != nil {
		return []byte{}, nil
	}

	// Update the nonce (if we are using multiple transactions from the same account)
	e.Nonces[strings.ToLower(addrFrom.String())]++

	// Return the transaction in bytes ready to send to the secondaries and threads.
	return signedTx.MarshalJSON()
}

// generateSimpleWorkload generates a simple transaction value transfer workload
// returns: Workload ([secondary][threads][time][tx]) -> [][][][]byte
func (e *EthereumWorkloadGenerator) generateSimpleWorkload() (Workload, error) {

	// get the known accounts
	var totalWorkload Workload
	txIndex := 0

	for secondaryNum := 0; secondaryNum < e.BenchConfig.Secondaries; secondaryNum++ {
		// secondaryWorkload = [thread][interval][tx=[]byte]
		// [][][][]byte
		secondaryWorkload := make(SecondaryWorkload, 0)
		for thread := 0; thread < e.BenchConfig.Threads; thread++ {
			// Thread workload = list of transactions in intervals
			// [interval][tx] = [][][]byte
			threadWorkload := make(WorkerThreadWorkload, 0)
			// for each thread, generate the intervals of transactions.
			for interval, txnum := range e.BenchConfig.TxInfo.Intervals {
				// Debug print for each interval to monitor correctness
				zap.L().Debug("Making workload ",
					zap.Int("secondary", secondaryNum),
					zap.Int("thread", thread),
					zap.Int("interval", interval),
					zap.Int("value", txnum))

				// Time interval = list of transactions
				// [tx] = [][]byte
				intervalWorkload := make([][]byte, 0)
				for txIt := 0; txIt < txnum; txIt++ {

					var tx []byte
					var txerr error

					// Initial assumption: there's as many accounts as transactions
					// TODO allow for more intricate transaction generation, such as A->B, A->C, etc.
					txVal, ok := big.NewInt(0).SetString("1000000", 10)
					if !ok {
						return nil, errors.New("failed to set TX value")
					}

					accFrom := secondaryNum + thread + (secondaryNum * e.BenchConfig.Threads)
					accTo := accFrom + 1

					// If the number of accounts are equal, then we have one account per secondary
					if len(e.KnownAccounts) >= e.BenchConfig.Secondaries && len(e.KnownAccounts) < e.BenchConfig.Secondaries*e.BenchConfig.Threads {
						tx, txerr = e.CreateSignedTransaction(
							e.KnownAccounts[secondaryNum%len(e.KnownAccounts)].PrivateKey,
							e.KnownAccounts[(secondaryNum+1)%len(e.KnownAccounts)].Address,
							txVal,
							[]byte{},
						)
					} else if len(e.KnownAccounts) == e.BenchConfig.Secondaries*e.BenchConfig.Threads {
						// One account per thread.
						accFrom := secondaryNum + thread + (secondaryNum * e.BenchConfig.Threads)
						accTo := accFrom + 1
						tx, txerr = e.CreateSignedTransaction(
							e.KnownAccounts[accFrom%len(e.KnownAccounts)].PrivateKey,
							e.KnownAccounts[accTo%len(e.KnownAccounts)].Address,
							txVal,
							[]byte{},
						)
					} else {
						// One account per transaction for all other transactions
						tx, txerr = e.CreateSignedTransaction(
							e.KnownAccounts[txIndex%len(e.KnownAccounts)].PrivateKey,
							e.KnownAccounts[txIndex+1%len(e.KnownAccounts)].Address,
							txVal,
							[]byte{},
						)

					}

					if txerr != nil {
						return nil, txerr
					}

					intervalWorkload = append(intervalWorkload, tx)
					txIndex++
				}
				threadWorkload = append(threadWorkload, intervalWorkload)
			}
			secondaryWorkload = append(secondaryWorkload, threadWorkload)
		}
		totalWorkload = append(totalWorkload, secondaryWorkload)
	}

	return totalWorkload, nil
}

// generateContractWorkload generates the workload for smart contract integration (or deployment)
// NOTE: Future implementations can have a separation to test both
// smart contract deployment and interaction in the same benchmark
// This can simulate a very realistic blockchain trace to replay existing chains?
func (e *EthereumWorkloadGenerator) generateContractWorkload() (Workload, error) {
	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(e.BenchConfig)
	if err != nil {
		return nil, err
	}

	// Deploy the contract
	contractAddr, err := e.DeployContract(e.KnownAccounts[0].PrivateKey, e.BenchConfig.ContractInfo.Path)

	if err != nil {
		return nil, err
	}

	// List of functions to create for each thread
	// TODO this needs some tuning!
	// This is a list of [id] pointing to each function
	// It will occur K times in the list, which is representative of the
	// ratio.
	functionsToCreatePerThread := make([]int, numberOfTransactions)

	for idx, funcInfo := range e.BenchConfig.ContractInfo.Functions {
		// add index to functionsToCreate
		funcRatio := (funcInfo.Ratio / 100) * numberOfTransactions

		for i := 0; i < funcRatio; i++ {
			functionsToCreatePerThread = append(functionsToCreatePerThread, idx)
		}
	}

	// Shuffle the function interactions
	// TODO check this carefully - we may have workloads with dependent transactions in future - maybe add this as a flag in config?
	ShuffleFunctionCalls(functionsToCreatePerThread)

	// Now generate the workload as usual
	var totalWorkload Workload
	txIndex := 0
	for secondaryID := 0; secondaryID < e.BenchConfig.Secondaries; secondaryID++ {
		secondaryWorkload := make(SecondaryWorkload, 0)
		for threadID := 0; threadID < e.BenchConfig.Threads; threadID++ {
			threadWorkload := make(WorkerThreadWorkload, 0)
			txCount := 0
			for _, numTx := range e.BenchConfig.TxInfo.Intervals {
				intervalWorkload := make([][]byte, 0)

				for i := 0; i < numTx; i++ {
					// function to create

					var tx []byte
					var txerr error
					funcToCreate := e.BenchConfig.ContractInfo.Functions[functionsToCreatePerThread[txCount]]
					zap.L().Debug(fmt.Sprintf("tx %d for func %s", txCount, funcToCreate.Name),
						zap.Int("secondary", secondaryID),
						zap.Int("thread", threadID))

					// If the number of accounts are equal, then we have one account per secondary
					if len(e.KnownAccounts) >= e.BenchConfig.Secondaries && len(e.KnownAccounts) < e.BenchConfig.Secondaries*e.BenchConfig.Threads {
						zap.L().Warn("Only enough accounts for one per secondary, this means there may be delays/fails for more threads")
						tx, txerr = e.CreateInteractionTX(
							e.KnownAccounts[secondaryID%len(e.KnownAccounts)].PrivateKey,
							contractAddr,
							funcToCreate.Name,
							funcToCreate.Params,
						)
					} else if len(e.KnownAccounts) == e.BenchConfig.Secondaries*e.BenchConfig.Threads {
						zap.L().Warn("Workload has one account per thread")
						// One account per thread.
						accFrom := secondaryID + threadID + (secondaryID * e.BenchConfig.Threads)
						tx, txerr = e.CreateInteractionTX(
							e.KnownAccounts[accFrom%len(e.KnownAccounts)].PrivateKey,
							contractAddr,
							funcToCreate.Name,
							funcToCreate.Params,
						)
					} else {
						// If there's not enough accounts, send a message saying that some transactions will fail
						if len(e.KnownAccounts) < (e.BenchConfig.Secondaries * e.BenchConfig.Threads) {
							zap.L().Warn("Not enough accounts, will experience fails due to sending nonce at incorrect times.")
						}

						// One account per transaction for all other transactions
						tx, txerr = e.CreateInteractionTX(
							e.KnownAccounts[txIndex%len(e.KnownAccounts)].PrivateKey,
							contractAddr,
							funcToCreate.Name,
							funcToCreate.Params,
						)

					}

					if txerr != nil {
						return nil, txerr
					}

					intervalWorkload = append(intervalWorkload, tx)
					txCount++
					txIndex++
				}

				threadWorkload = append(threadWorkload, intervalWorkload)
			}
			secondaryWorkload = append(secondaryWorkload, threadWorkload)
		}
		totalWorkload = append(totalWorkload, secondaryWorkload)
	}

	// Get workload ready
	return totalWorkload, nil
}

// GenerateWorkload creates a workload of transactions to be used in the benchmark for all clients.
func (e *EthereumWorkloadGenerator) GenerateWorkload() (Workload, error) {
	// 1/ work out the total number of secondaries.
	numberOfWorkingSecondaries := e.BenchConfig.Secondaries * e.BenchConfig.Threads

	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(e.BenchConfig)

	if err != nil {
		return nil, err
	}

	// Total transactions
	totalTx := numberOfTransactions * numberOfWorkingSecondaries

	zap.L().Info(
		"Generating workload",
		zap.String("workloadType", string(e.BenchConfig.TxInfo.TxType)),
		zap.Int("secondaries", numberOfWorkingSecondaries),
		zap.Int("transactionsPerSecondary", numberOfTransactions),
		zap.Int("totalTransactions", totalTx),
	)

	// Print a warning about the accounts
	if len(e.KnownAccounts) >= e.BenchConfig.Secondaries && len(e.KnownAccounts) < e.BenchConfig.Secondaries*e.BenchConfig.Threads {
		zap.L().Warn("Only enough accounts for one per secondary, this means there may be delays/fails for more threads")
	} else if len(e.KnownAccounts) == e.BenchConfig.Secondaries*e.BenchConfig.Threads {
		zap.L().Warn("Workload has one account per thread")
	} else if len(e.KnownAccounts) < (e.BenchConfig.Secondaries * e.BenchConfig.Threads) {
		// If there's not enough accounts, send a message saying that some transactions will fail
		zap.L().Warn("Not enough accounts, will experience fails due to sending nonce at incorrect times.")
	}

	switch e.BenchConfig.TxInfo.TxType {
	case configs.TxTypeSimple:
		return e.generateSimpleWorkload()
	case configs.TxTypeContract:
		return e.generateContractWorkload()
	default:
		return nil, errors.New("unknown transaction type in config for workload generation")
	}
}
