package workloadgenerators

import (
	"context"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"math/big"
)

// Generates the workload for the Ethereum blockchain
type EthereumWorkloadGenerator struct {
	ActiveConn        *ethclient.Client
	SuggestedGasPrice *big.Int
	BenchConfig       *configs.BenchConfig
	ChainConfig       *configs.ChainConfig
	Nonces            map[string]uint64
	ChainID           *big.Int
	KnownAccounts     []configs.ChainKey
}

// Returns a new instance of the generator
func (e *EthereumWorkloadGenerator) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	return &EthereumWorkloadGenerator{BenchConfig: benchConfig, ChainConfig: chainConfig}
}

// Set up the blockchain nodes
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

// Sets the suggested gas price and sets up a small connection to get information from the blockchain.
func (e *EthereumWorkloadGenerator) InitParams() error {

	// Connect to the blockchain
	c, err := ethclient.Dial(fmt.Sprintf("ws://%s", e.ChainConfig.Nodes[0]))

	if err != nil {
		return err
	}

	e.ActiveConn = c

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

		e.Nonces[key.Address] = v
	}

	return nil
}

// Generic account creation to return the private key
func (e *EthereumWorkloadGenerator) CreateAccount() (interface{}, error) {
	privKey, err := crypto.GenerateKey()

	if err != nil {
		return nil, err
	}

	return privKey, nil
}

// Deploy the contract
func (e *EthereumWorkloadGenerator) DeployContract(fromPivKey []byte, contractPath string) (string, error) {
	// TODO implement
	return "", nil
}

// Creates a transaction to deploy the contract
func (e *EthereumWorkloadGenerator) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {
	return []byte{}, nil
}

// Creates an interaction with the contract
func (e *EthereumWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}

// Create a signed transaction that returns the bytes
func (e *EthereumWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int) ([]byte, error) {

	// Get the private key
	priv, err := crypto.HexToECDSA(hex.EncodeToString(fromPrivKey))

	if err != nil {
		return []byte{}, err
	}

	// Get the address from the private key
	addrFrom := crypto.PubkeyToAddress(priv.PublicKey)

	// Get the transaction fields
	toConverted := common.HexToAddress(toAddress)
	gasLimit := uint64(21000)

	zap.L().Debug("transaction params",
		zap.String("addrFrom", addrFrom.String()),
		zap.String("addrTo", toAddress),
	)

	// Make and sign the transaction
	tx := types.NewTransaction(e.Nonces[addrFrom.String()], toConverted, value, gasLimit, e.SuggestedGasPrice, []byte{})
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(e.ChainID), priv)
	if err != nil {
		return []byte{}, nil
	}

	// Update the nonce (if we are using multiple transactions from the same account)
	e.Nonces[addrFrom.String()] += 1

	// Return the transaction in bytes ready to send to the clients.
	return signedTx.MarshalJSON()
}

func (e *EthereumWorkloadGenerator) generateSimpleWorkload() (Workload, error) {
	// get the known accounts
	var totalWorkload [][][]byte
	for clientNum := 0; clientNum < e.BenchConfig.Clients; clientNum++ {
		clientWorkload := make([][]byte, 0)
		for worker := 0; worker < e.BenchConfig.Workers; worker++ {
			// Initial assumption: there's as many accounts as transactions
			// TODO allow for more intricate transaction generation, such as A->B, A->C, etc.
			txVal, ok := big.NewInt(0).SetString("1000000", 10)
			if !ok {
				return nil, errors.New("failed to set TX value")
			}
			tx, err := e.CreateSignedTransaction(
				e.KnownAccounts[clientNum+worker].PrivateKey,
				e.KnownAccounts[((clientNum+worker)+1)%len(e.KnownAccounts)].Address,
				txVal,
			)

			if err != nil {
				return nil, err
			}

			clientWorkload = append(clientWorkload, tx)
		}

		totalWorkload = append(totalWorkload, clientWorkload)
	}

	return totalWorkload, nil
}

func (e *EthereumWorkloadGenerator) generateContractWorkload() (Workload, error) {
	return nil, nil
}

// Generate the workload, returning the slice of transactions. [clientID = [ list of transactions ] ]
func (e *EthereumWorkloadGenerator) GenerateWorkload() (Workload, error) {
	// 1/ work out the total number of clients.
	numberOfWorkingClients := e.BenchConfig.Clients * e.BenchConfig.Workers

	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(e.BenchConfig)

	if err != nil {
		return nil, err
	}

	// Total transactions
	totalTx := numberOfTransactions * numberOfWorkingClients

	zap.L().Info(
		"Generating workload",
		zap.String("workloadType", string(e.BenchConfig.TxInfo.TxType)),
		zap.Int("clients", numberOfWorkingClients),
		zap.Int("transactionsPerClient", numberOfTransactions),
		zap.Int("totalTransactions", totalTx),
	)

	switch e.BenchConfig.TxInfo.TxType {
	case configs.TxTypeSimple:
		return e.generateSimpleWorkload()
	case configs.TxTypeContract:
		return e.generateContractWorkload()
	default:
		return nil, errors.New("unknown transaction type in config for workload generation")
	}
}
