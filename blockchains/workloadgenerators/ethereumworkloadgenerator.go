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
	GenericWorkloadGenerator
}

const (
	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	wordBytes = wordBits / 8
)

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

type noncePair struct {
	account  string
	nonce    uint64
}

// InitParams sets initial aspects such as the suggested gas price and sets up a small connection to get information from the blockchain.
func (e *EthereumWorkloadGenerator) InitParams() error {
	// Connect to the blockchain
	zap.L().Debug("dial node[0]",
		zap.String("address", e.ChainConfig.Nodes[0]))
	c, err := ethclient.Dial(fmt.Sprintf("ws://%s", e.ChainConfig.Nodes[0]))

	if err != nil {
		return err
	}

	e.ActiveConn = c

	// Get the suggested gas price from the network using a client connected
	zap.L().Debug("get gas price")
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

	nonceChan := make(chan *noncePair, 100)

	for _, key := range e.KnownAccounts {
		if len(nonceChan) >= cap(nonceChan) {
			pair := <-nonceChan
			e.Nonces[pair.account] = pair.nonce
		}

		zap.L().Debug("get pending nonce",
			zap.String("account", fmt.Sprintf("%v", key.Address)))

		go func(account string) {
			v, err := e.ActiveConn.PendingNonceAt(context.Background(), common.HexToAddress(account))

			if err != nil {
				panic(err)
			}

			nonceChan <- &noncePair{account: account, nonce: v}
		}(key.Address)
	}

	for len(nonceChan) > 0 {
		pair := <-nonceChan
		e.Nonces[pair.account] = pair.nonce
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
		if len(contracts) == 0 {
			return nil, fmt.Errorf("no contracts to compile")
		}

		// TODO handle case where number of contracts is greater than one
		var contract *compiler.Contract

		if e.BenchConfig.ContractInfo.Name != "" {
			for k, v := range contracts {
				s := strings.Split(k, ":")
				if s[len(s)-1] == e.BenchConfig.ContractInfo.Name {
					contract = v
					break
				}
			}

			if contract == nil {
				zap.L().Error(fmt.Sprintf("Failed to find contract %v in %v", e.BenchConfig.ContractInfo.Name, contracts))
				return nil, fmt.Errorf("failed to find contract in compiled")
			}
		} else {
			for k, v := range contracts {
				zap.L().Warn("Name not provided, compiling first contract",
					zap.String("contract", k),
				)
				contract = v
				break
			}
		}

		zap.L().Info("Deploying Contract",
			zap.String("contract", e.BenchConfig.ContractInfo.Name),
			zap.String("path", e.BenchConfig.ContractInfo.Path),
		)

		bytecodeBytes, err := hex.DecodeString(contract.Code[2:])

		if err != nil {
			return []byte{}, err
		}

		// TODO maybe estimate gas rather than have an upper bound
		gasLimit := uint64(2000000)

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
		e.CompiledContract = contract

		return signedTx.MarshalJSON()

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

// ReadBits encodes the absolute value of bigint as big-endian bytes. Callers must ensure
// that buf has enough space. If buf is too short the result will be incomplete.
// This function is taken from: https://github.com/ethereum/go-ethereum/blob/master/common/math/big.go
func readBits(bigint *big.Int, buf []byte) {
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < wordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
}

// Converts the uint256 into padded bytes
func convertU256(i *big.Int) []byte {
	if i.BitLen()/8 >= 32 {
		return i.Bytes()
	}

	ret := make([]byte, 32)
	readBits(i, ret)
	return ret
}

// getCallDataBytes will return the ABI encoded bytes for the variable, or an error
// if it cannot be converted
func (e *EthereumWorkloadGenerator) getCallDataBytes(paramType string, val string) ([]byte, error) {

	payloadBytes := make([]byte, 0)

	switch paramType {
	// uints
	case "uint8":
		// uint 8 = 1 byte
		// padding = 31 bytes
		num, err := strconv.ParseUint(val, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
		}
		padding := make([]byte, 31)
		payloadBytes = append(payloadBytes, padding...)
		payloadBytes = append(payloadBytes, uint8(num))
		break
	case "uint32":
		// uint 32 = 4 bytes
		// padding = 28 bytes
		num, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
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
		num, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
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
		num, ok := big.NewInt(0).SetString(val, 10)
		if !ok {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
		}
		payloadBytes = append(payloadBytes, convertU256(num)...)
		break
	// ints
	case "int8":
		// int 8 = 1 byte
		// padding = 31 bytes
		num, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
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
		num, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
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
		num, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
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
		num, ok := big.NewInt(0).SetString(val, 10)
		if !ok {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
		}
		payloadBytes = append(payloadBytes, num.Bytes()...)
		break
	// bool
	case "bool":
		// Bool is just a padded uint8 of value 0 or 1
		var bVal uint8
		if val == "true" {
			bVal = 1
		} else if val == "false" {
			bVal = 0
		} else {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
		}
		padding := make([]byte, 31)
		payloadBytes = append(payloadBytes, padding...)
		payloadBytes = append(payloadBytes, bVal)
		break
	// address
	case "address":
		// uint160
		// get the address
		addr := common.HexToAddress(val)
		// padding - address bytes should be 20bytes long.
		padding := make([]byte, 12)
		payloadBytes = append(payloadBytes, padding...)
		payloadBytes = append(payloadBytes, addr.Bytes()...)
		break
	// bytes
	case "bytes24":
		// TODO this needs improvement!
		s := []byte(val)
		padding := make([]byte, 32-len(s))
		payloadBytes = append(payloadBytes, s...)
		payloadBytes = append(payloadBytes, padding...)
		break
	case "bytes32":
		// TODO this needs improvement!
		s := []byte(val)
		payloadBytes = append(payloadBytes, s...)
		break

	// DYNAMIC TYPES ARE HARD :(
	case "string", "bytes":
		// todo this needs to be checked!
		s := []byte(val)
		// 2 get the length of the bytes
		slen := uint32(len(s))
		// make a uint and pad that bigendian
		spadding := make([]byte, 28)
		payloadBytes = append(payloadBytes, spadding...)
		numBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(numBytes, uint32(slen))
		payloadBytes = append(payloadBytes, numBytes...)
		// 3 - add the padded string
		padding := make([]byte, (32 - (len(s) % 32)))
		payloadBytes = append(payloadBytes, s...)
		payloadBytes = append(payloadBytes, padding...)
		break
	// Default
	default:
		return nil, fmt.Errorf("invalid arg type: %s", paramType)
	}

	return payloadBytes, nil
}

// CreateInteractionTX forms a transaction that invokes a smart contract
func (e *EthereumWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam, value string) ([]byte, error) {
	// Check that the contract has been compiled, if nto - then it's difficult to get the hashes from the ABI.
	if e.CompiledContract == nil {
		return nil, fmt.Errorf("contract does not exist in known generator")
	}

	// next - get the function hash
	var funcHash string

	// If we are targeting the fallback function, or, just sending ether - we can ignore the
	// function name.
	if functionName == "fallback" || functionName == "receive" || functionName == "()" {
		funcHash = ""
	} else {
		val, ok := e.CompiledContract.Hashes[functionName]
		if !ok {
			zap.L().Debug("Failed to find function",
				zap.String("ABI", fmt.Sprintf("%v", e.CompiledContract.Hashes)),
				zap.String("Function", fmt.Sprintf("%v", functionName)))
			return nil, fmt.Errorf("contract does not contain function: %s", functionName)
		}
		funcHash = val
	}
	// Now we need to parse the arguments to get them into the correct padding
	payloadBytes, err := hex.DecodeString(funcHash)
	if err != nil {
		return nil, err
	}

	// Then go through and convert the params
	// Types taken from: https://solidity.readthedocs.io/en/develop/abi-spec.html#types
	// NOTE: need to pad to 32 bytes

	// NOTE#2: Dynamic Types require points to show where each type begin and ends
	// look at "abi.encode" for JS
	isDynamic := false
	for _, v := range contractParams {
		if v.Type == "string" || v.Type == "bytes" {
			isDynamic = true
			break
		}
	}

	// If it's dynamic - then we need to have to space things out :\
	// encoding = location in calldata
	// e.g. func(string, uint)
	//      = location_of_string, uint, stringlen, stringdata

	// e.g. 2 func(string, uint, string)
	//      = location_of_string1, uint, location_of_string2, stringlen, stringdata, stringlen, stringdata
	//
	// length (pad to 32 bytes)
	// data (pad to nearest 32 bytes)

	if !isDynamic {
		for _, v := range contractParams {
			encB, err := e.getCallDataBytes(v.Type, v.Value)
			if err != nil {
				return nil, err
			}
			payloadBytes = append(payloadBytes, encB...)
		}
	} else {
		zap.L().Debug("Contract call contains dynamic values - wizard time")

		// 1 get all the encoded values
		var nonDynArr [][]byte
		var dynArr [][]byte
		totalNonDynLength := 0
		for _, v := range contractParams {
			encB, err := e.getCallDataBytes(v.Type, v.Value)
			if err != nil {
				return nil, err
			}
			zap.L().Debug("Bytes",
				zap.String("Type", v.Type),
				zap.String("Val", v.Value),
				zap.String("Bytes", fmt.Sprintf("%x", encB)),
			)
			// if it's dynamic - add a 32byte placeholder
			// and append to the dynamic data
			if v.Type == "string" || v.Type == "bytes" {
				nonDynArr = append(nonDynArr, []byte{})
				dynArr = append(dynArr, encB)
				totalNonDynLength += 32
			} else {
				nonDynArr = append(nonDynArr, encB)
				totalNonDynLength += len(encB)
			}
		}

		// 2 work out positioning
		fullBytes := make([]byte, 0)
		allDynBytes := make([]byte, 0)
		currentOffset := totalNonDynLength
		dynIndex := 0
		for _, v := range nonDynArr {
			// if it has 0 length, it is dynamic so we work out based
			// on the offset
			if len(v) == 0 {

				// Set the offset
				offsetBytes := make([]byte, 28)
				numBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(numBytes, uint32(currentOffset))
				fullBytes = append(fullBytes, offsetBytes...)
				fullBytes = append(fullBytes, numBytes...)

				// Update the offset
				currentOffset += len(dynArr[dynIndex])

				// Append all the bytes to the end
				allDynBytes = append(allDynBytes, dynArr[dynIndex]...)
			} else {
				fullBytes = append(fullBytes, v...)
			}
		}

		// 3 fill in the final parts
		payloadBytes = append(payloadBytes, fullBytes...)
		payloadBytes = append(payloadBytes, allDynBytes...)
	}

	// Assume that the payload bytes have been correctly formed at this point?
	if len(payloadBytes) < 1 {
		zap.L().Warn(fmt.Sprintf("no payload generated, sending transaction with 0 data bytes"))
	}

	// Create the signed transaction
	if value == "" {
		value = "0"
	}
	sendVal, ok := big.NewInt(0).SetString(value, 16)
	if !ok {
		zap.L().Warn(fmt.Sprintf("Failed to set value of tx, could not convert %s to big number", value))
	}

	tx, err := e.CreateSignedTransaction(fromPrivKey, contractAddress, sendVal, payloadBytes)

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

	// 1. Set up the accounts into buckets for each
	accountDistribution := make([][]*configs.ChainKey, e.BenchConfig.Secondaries*e.BenchConfig.Threads)

	accountCount := 0
	for {
		// exit condition - each thread has an assigned account, and we've run out of accounts.
		if accountCount >= len(e.KnownAccounts) && accountCount >= len(accountDistribution) {
			break
		}

		currentAccount := accountCount % len(e.KnownAccounts)
		currentDist := accountCount % len(accountDistribution)

		accountDistribution[currentDist] = append(accountDistribution[currentDist], &e.KnownAccounts[currentAccount])

		accountCount++
	}

	// 2. Generate the transactions
	txID := 0
	accountBatch := 0
	for secondaryID := 0; secondaryID < e.BenchConfig.Secondaries; secondaryID++ {
		// secondaryWorkload = [thread][interval][tx=[]byte]
		// [][][][]byte
		secondaryWorkload := make(SecondaryWorkload, 0)
		for thread := 0; thread < e.BenchConfig.Threads; thread++ {
			// Thread workload = list of transactions in intervals
			// [interval][tx] = [][][]byte
			threadWorkload := make(WorkerThreadWorkload, 0)
			// for each thread, generate the intervals of transactions.
			zap.L().Debug("Info",
				zap.Int("secondary", secondaryID),
				zap.Int("thread", thread),
				zap.Int("len", len(accountDistribution)))
			accountsChoices := accountDistribution[accountBatch]
			for interval, txnum := range e.TPSIntervals {
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

					// Initial assumption: there's as many accounts as transactions
					// TODO allow for more intricate transaction generation, such as A->B, A->C, etc.
					txVal, ok := big.NewInt(0).SetString("1000000", 10)
					if !ok {
						return nil, errors.New("failed to set TX value")
					}

					// accFrom := secondaryID + thread + (secondaryID * e.BenchConfig.Threads)
					accFrom := accountsChoices[txID%len(accountsChoices)]
					accTo := accountsChoices[(txID+1)%len(accountsChoices)]

					tx, txerr := e.CreateSignedTransaction(
						accFrom.PrivateKey,
						accTo.Address,
						txVal,
						[]byte{},
					)

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
	functionsToCreatePerThread := make([]int, 0)

	for idx, funcInfo := range e.BenchConfig.ContractInfo.Functions {
		// add index to functionsToCreate
		var funcRatio int
		if idx == len(e.BenchConfig.ContractInfo.Functions) - 1 {
			funcRatio = numberOfTransactions - len(functionsToCreatePerThread)
		} else {
			funcRatio = (funcInfo.Ratio * numberOfTransactions) / 100
		}

		for i := 0; i < funcRatio; i++ {
			functionsToCreatePerThread = append(functionsToCreatePerThread, idx)
		}
	}

	// 1. Set up the accounts into buckets for each
	accountDistribution := make([][]*configs.ChainKey, e.BenchConfig.Secondaries*e.BenchConfig.Threads)

	accountCount := 0
	for {

		// exit condition - each thread has an assigned account, and we've run out of accounts.
		if accountCount >= len(e.KnownAccounts) && accountCount >= len(accountDistribution) {
			break
		}

		currentAccount := accountCount % len(e.KnownAccounts)
		currentDist := accountCount % len(accountDistribution)

		accountDistribution[currentDist] = append(accountDistribution[currentDist], &e.KnownAccounts[currentAccount])

		accountCount++
	}

	// Shuffle the function interactions
	// TODO check this carefully - we may have workloads with dependent transactions in future - maybe add this as a flag in config?
	// ShuffleFunctionCalls(functionsToCreatePerThread)

	// Now generate the workload as usual
	var totalWorkload Workload
	txIndex := 0
	accountBatch := 0
	for secondaryID := 0; secondaryID < e.BenchConfig.Secondaries; secondaryID++ {
		secondaryWorkload := make(SecondaryWorkload, 0)
		for threadID := 0; threadID < e.BenchConfig.Threads; threadID++ {
			threadWorkload := make(WorkerThreadWorkload, 0)
			txCount := 0

			accountsChoices := accountDistribution[accountBatch]

			// 			for _, numTx := range e.BenchConfig.TxInfo.Intervals {
			for _, numTx := range e.TPSIntervals {
				intervalWorkload := make([][]byte, 0)

				for i := 0; i < numTx; i++ {
					// function to create
					accFrom := accountsChoices[txIndex%len(accountsChoices)]
					funcToCreate := e.BenchConfig.ContractInfo.Functions[functionsToCreatePerThread[txCount]]
					zap.L().Debug(fmt.Sprintf("tx %d for func %s", txCount, funcToCreate.Name),
						zap.Int("secondary", secondaryID),
						zap.Int("thread", threadID))
					var functionParamSigs []string
					var functionFinal string
					if len(funcToCreate.Params) > 0 {

						for _, paramVal := range funcToCreate.Params {
							functionParamSigs = append(functionParamSigs, paramVal.Type)
						}

						functionFinal = fmt.Sprintf("%s(%s)", funcToCreate.Name, strings.Join(functionParamSigs[:], ","))
					} else {
						functionFinal = fmt.Sprintf("%s()", funcToCreate.Name)
					}

					tx, txerr := e.CreateInteractionTX(
						accFrom.PrivateKey,
						contractAddr,
						functionFinal,
						funcToCreate.Params,
						funcToCreate.PayValue,
					)

					if txerr != nil {
						return nil, txerr
					}

					intervalWorkload = append(intervalWorkload, tx)
					txCount++
					txIndex++
				}

				threadWorkload = append(threadWorkload, intervalWorkload)
			}
			accountBatch++
			secondaryWorkload = append(secondaryWorkload, threadWorkload)
		}
		totalWorkload = append(totalWorkload, secondaryWorkload)
	}

	// Get workload ready
	return totalWorkload, nil
}

// generatePremadeWorkload generates the workload for the "premade" json file that
// is associated with this workload.
func (e *EthereumWorkloadGenerator) generatePremadeWorkload() (Workload, error) {
	// 1 deploy the contract if it is a contract workload, get the address
	var contractAddr string
	if len(e.BenchConfig.ContractInfo.Path) > 0 && len(e.BenchConfig.ContractInfo.Name) > 0 {
		// Deploy the contract
		var err error
		contractAddr, err = e.DeployContract(e.KnownAccounts[0].PrivateKey, e.BenchConfig.ContractInfo.Path)

		if err != nil {
			return nil, err
		}
	}

	var fullWorkload Workload
	// 2 loop through the premade dataset and create the relevant transactions
	for secondaryIndex, secondaryWorkload := range e.BenchConfig.TxInfo.PremadeInfo {

		secondaryTransactions := make(SecondaryWorkload, 0)

		for threadIndex, threadWorkload := range secondaryWorkload {

			threadTransactions := make(WorkerThreadWorkload, 0)

			for intervalIndex, intervalWorkload := range threadWorkload {

				intervalTransactions := make([][]byte, 0)

				for _, txInfo := range intervalWorkload {
					// Make the transaction based on its
					fromID, err := strconv.Atoi(txInfo.From)
					fromAccount := e.KnownAccounts[fromID%len(e.KnownAccounts)]
					if err != nil {
						return nil, fmt.Errorf("[Premade tx: %v] Failed to convert %v to int", txInfo.ID, txInfo.From)
					}

					var toAccount string
					if txInfo.To == "contract" {
						toAccount = contractAddr
					} else {
						toID, err := strconv.Atoi(txInfo.To)
						if err != nil {
							return nil, fmt.Errorf("[Premade tx: %v] Failed to convert %v to int", txInfo.ID, txInfo.To)
						}
						toAccount = e.KnownAccounts[toID%len(e.KnownAccounts)].Address
					}

					zap.L().Debug("Premade Transaction",
						zap.String("Tx Info", fmt.Sprintf("[S: %v, T: %v, I: %v]", secondaryIndex, threadIndex, intervalIndex)),
						zap.String(fmt.Sprintf("From (%v): ", txInfo.From), fmt.Sprintf("%v", fromAccount.Address)),
						zap.String(fmt.Sprintf("To (%v): ", txInfo.To), fmt.Sprintf("%v", toAccount)),
						zap.String("ID", txInfo.ID),
						zap.String("Function", txInfo.Function),
					)

					var finalTx []byte

					txVal, ok := big.NewInt(0).SetString(txInfo.Value, 10)

					if !ok {
						return nil, fmt.Errorf("Failed to set value to big int: %s", txInfo.Value)
					}

					if txInfo.Function == "" && len(txInfo.DataParams) == 0 {
						// This is a simple transaction
						finalTx, err = e.CreateSignedTransaction(
							fromAccount.PrivateKey,
							toAccount,
							txVal,
							[]byte{},
						)

						if err != nil {
							return nil, err
						}

					} else {
						// This is a contract
						if txInfo.Function == "constructor" {
							// Constructor = make a deploy transaction
							finalTx, err = e.CreateContractDeployTX(
								fromAccount.PrivateKey,
								e.BenchConfig.ContractInfo.Path,
							)

							if err != nil {
								return nil, err
							}

						} else {
							// It's an interaction transaction

							// function name should be: function(type,type,type)
							var txParams []configs.ContractParam
							var functionParamSigs []string
							for _, paramVal := range txInfo.DataParams {
								functionParamSigs = append(functionParamSigs, paramVal.Type)
								txParams = append(txParams, configs.ContractParam{Type: paramVal.Type, Value: paramVal.Value})
							}

							functionFinal := fmt.Sprintf("%s(%s)", txInfo.Function, strings.Join(functionParamSigs[:], ","))

							finalTx, err = e.CreateInteractionTX(
								fromAccount.PrivateKey,
								toAccount,
								functionFinal,
								txParams,
								txInfo.Value,
							)
						}

					}

					intervalTransactions = append(intervalTransactions, finalTx)
				}

				threadTransactions = append(threadTransactions, intervalTransactions)
			}

			secondaryTransactions = append(secondaryTransactions, threadTransactions)
		}

		fullWorkload = append(fullWorkload, secondaryTransactions)
	}

	// 3 return the workload to be distributed
	return fullWorkload, nil
}

// GenerateWorkload creates a workload of transactions to be used in the benchmark for all clients.
func (e *EthereumWorkloadGenerator) GenerateWorkload() (Workload, error) {
	// 1/ work out the total number of secondaries.
	numberOfWorkers := e.BenchConfig.Secondaries * e.BenchConfig.Threads

	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(e.BenchConfig)

	if err != nil {
		return nil, err
	}

	// Total transactions
	totalTxPerWorker := numberOfTransactions / numberOfWorkers

	zap.L().Info(
		"Generating workload",
		zap.String("workloadType", string(e.BenchConfig.TxInfo.TxType)),
		zap.Int("threadsTotal", numberOfWorkers),
		zap.Int("totalTransactions per worker", totalTxPerWorker),
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
	case configs.TxTypePremade:
		return e.generatePremadeWorkload()
	default:
		return nil, errors.New("unknown transaction type in config for workload generation")
	}
}
