package workloadgenerators

import (
	"bytes"
	"context"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	PACKET_DATA_SIZE int = 1280 - 40 - 8
)

func calculateMaxChunkSize(
	createTransaction func(offset int, data []byte) (*solana.Transaction, error),
) (size int, err error) {
	transaction, err := createTransaction(0, []byte{})
	if err != nil {
		return
	}
	signatures := make(
		[]solana.Signature,
		transaction.Message.Header.NumRequiredSignatures,
	)
	transaction.Signatures = append(transaction.Signatures, signatures...)
	serialized, err := transaction.MarshalBinary()
	if err != nil {
		return
	}
	size = PACKET_DATA_SIZE - len(serialized) - 1
	return
}

var (
	versionRegexp      = regexp.MustCompile(`([0-9]+)\.([0-9]+)\.([0-9]+)`)
	contractNameRegexp = regexp.MustCompile(`found contract ‘(.*)’`)
	dataUsageRegexp    = regexp.MustCompile(`least (.*) bytes`)
	binaryPathRegexp   = regexp.MustCompile(`binary (.*) for`)
	abiPathRegexp      = regexp.MustCompile(`ABI (.*) for`)
)

type solanaClient struct {
	rpcClient *rpc.Client
	wsClient  *ws.Client
}

type Solang struct {
	Path, Version, FullVersion string
	Major, Minor, Patch        int
}

type SolangContract struct {
	Name          string
	RequiredSpace uint64
	Data          []byte
	Abi           abi.ABI
	Hashes        map[string][]byte // method signature => hash
}

type SolangDeployedContract struct {
	Contract       *SolangContract
	ProgramAccount *solana.Wallet
	StorageAccount *solana.Wallet
}

// SolanaWorkloadGenerator is the workload generator implementation for the Solana blockchain
type SolanaWorkloadGenerator struct {
	// SuggestedGasPrice *big.Int             // Suggested gas price on the network
	ActiveConn  *solanaClient        // Active connection to a blockchain node for information
	BenchConfig *configs.BenchConfig // Benchmark configuration for workload intervals / type
	ChainConfig *configs.ChainConfig // Chain configuration to get number of transactions to make
	// Nonces            map[string]uint64    // Nonce of the known accounts
	// ChainID           *big.Int             // ChainID for transactions, provided through the ethereum API
	KnownAccounts    []configs.ChainKey      // Known accounds, public:private key pair
	CompiledContract *SolangDeployedContract // Compiled contract bytecode for the contract used in complex workloads
	logger           *zap.Logger
	GenericWorkloadGenerator
}

func NewSolanaWorkloadGenerator() *SolanaWorkloadGenerator {
	return &SolanaWorkloadGenerator{logger: zap.L().Named("SolanaWorkloadGenerator")}
}

func (s *SolanaWorkloadGenerator) NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator {
	return &SolanaWorkloadGenerator{ChainConfig: chainConfig, BenchConfig: benchConfig, logger: s.logger}
}

func (s *SolanaWorkloadGenerator) BlockchainSetup() error {
	// TODO implement
	// 1 - create N accounts only if we don't have accounts
	if len(s.ChainConfig.Keys) > 0 {
		s.KnownAccounts = s.ChainConfig.Keys
		return nil
	}
	if len(s.ChainConfig.Extra) > 0 {
		accountsData, err := os.ReadFile(s.ChainConfig.Extra[0].(string))
		if err != nil {
			return err
		}
		var accounts map[string]interface{}
		err = yaml.Unmarshal(accountsData, &accounts)
		if err != nil {
			return err
		}

		for k := range accounts {
			var priv solana.PrivateKey
			err := json.Unmarshal([]byte(k), &priv)
			if err != nil {
				return err
			}

			s.KnownAccounts = append(s.KnownAccounts, configs.ChainKey{PrivateKey: priv, Address: priv.PublicKey().String()})
		}
	}
	// 2 - fund with genesis block, write to genesis location
	// 3 - copy genesis to blockchain nodes
	return nil
}

func (s *SolanaWorkloadGenerator) InitParams() error {
	s.logger.Debug("InitParams")
	conn := rpc.New(fmt.Sprintf("http://%s", s.ChainConfig.Nodes[0]))

	ip, portStr, err := net.SplitHostPort(s.ChainConfig.Nodes[0])
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}

	sock, err := ws.Connect(context.Background(), fmt.Sprintf("ws://%s", net.JoinHostPort(ip, strconv.Itoa(port+1))))
	if err != nil {
		return err
	}

	s.ActiveConn = &solanaClient{conn, sock}

	return nil
}

func (s *SolanaWorkloadGenerator) CreateAccount() (interface{}, error) {
	return solana.NewWallet().PrivateKey, nil
}

func (s *SolanaWorkloadGenerator) DeployContract(fromPrivKey []byte, contractPath string) (string, error) {
	s.logger.Debug("DeployContract", zap.Binary("fromPrivKey", fromPrivKey), zap.String("contractPath", contractPath))
	txBatchesBytes, err := s.CreateContractDeployTX(fromPrivKey, contractPath)
	if err != nil {
		return "", err
	}

	var txBatches [][]*solana.Transaction
	err = json.Unmarshal(txBatchesBytes, &txBatches)
	if err != nil {
		return "", err
	}

	for n, batch := range txBatches {
		s.logger.Debug("Processing batch", zap.Int("index", n))
		var sigs []solana.Signature
		for _, tx := range batch {
			sig, err := s.ActiveConn.rpcClient.SendTransactionWithOpts(context.Background(), tx, false, rpc.CommitmentFinalized)
			if err != nil {
				return "", err
			}
			sigs = append(sigs, sig)
		}
		for i := 0; i < 60; i++ {
			s.logger.Debug("waiting for", zap.Int("txs", len(sigs)))
			got, err := s.ActiveConn.rpcClient.GetSignatureStatuses(context.Background(), true, sigs...)
			if err != nil {
				return "", err
			}
			if got == nil {
				return "", errors.New("empty signatures")
			}
			var requiredSigs []solana.Signature
			for idx, status := range got.Value {
				if status != nil && status.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
					continue
				}
				requiredSigs = append(requiredSigs, sigs[idx])
				if status != nil {
					s.logger.Debug("status", zap.String("sig", sigs[idx].String()), zap.String("status", string(status.ConfirmationStatus)))
				}
			}
			sigs = requiredSigs
			if len(requiredSigs) == 0 {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if len(sigs) > 0 {
			return "", errors.New("failed to wait for transaction finalization")
		}
	}

	return s.CompiledContract.ProgramAccount.PublicKey().String(), nil
}

func solangVersion(solang string) (*Solang, error) {
	if solang == "" {
		solang = "solang"
	}
	var out bytes.Buffer
	cmd := exec.Command(solang, "--version")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	matches := versionRegexp.FindStringSubmatch(out.String())
	if len(matches) != 4 {
		return nil, fmt.Errorf("can't parse solang version %q", out.String())
	}
	s := &Solang{Path: cmd.Path, FullVersion: out.String(), Version: matches[0]}
	if s.Major, err = strconv.Atoi(matches[1]); err != nil {
		return nil, err
	}
	if s.Minor, err = strconv.Atoi(matches[2]); err != nil {
		return nil, err
	}
	if s.Patch, err = strconv.Atoi(matches[3]); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SolanaWorkloadGenerator) compileSolidity(contractPath string) (contract *SolangContract, err error) {
	s.logger.Debug("compileSolidity", zap.String("contractPath", contractPath))
	dir, err := os.MkdirTemp("", "diablo-solang")
	if err != nil {
		return
	}
	s.logger.Debug("Using directory", zap.String("dir", dir))
	defer func() {
		tmpErr := os.RemoveAll(dir)
		if tmpErr != nil {
			err = tmpErr
		}
	}()
	solang, err := solangVersion("")
	if err != nil {
		return
	}
	args := []string{
		"--verbose",
		"--output", dir,
		"--target", "solana",
		contractPath,
	}
	cmd := exec.Command(solang.Path, args...)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("solang: %v\n%s", err, stderr.Bytes())
	}
	contract = &SolangContract{}
	contractNameMatches := contractNameRegexp.FindStringSubmatch(stderr.String())
	if len(contractNameMatches) != 2 {
		return nil, fmt.Errorf("can't parse contract name %q", stderr.String())
	}
	contract.Name = contractNameMatches[1]
	dataUsageMatches := dataUsageRegexp.FindStringSubmatch(stderr.String())
	if len(dataUsageMatches) != 2 {
		return nil, fmt.Errorf("can't parse data usage %q", stderr.String())
	}
	dataUsage, err := strconv.Atoi(dataUsageMatches[1])
	if err != nil {
		return nil, err
	}
	contract.RequiredSpace = uint64(dataUsage)
	binaryPathMatches := binaryPathRegexp.FindStringSubmatch(stderr.String())
	if len(binaryPathMatches) != 2 {
		return nil, fmt.Errorf("can't parse binary path %q", stderr.String())
	}
	if contract.Data, err = os.ReadFile(binaryPathMatches[1]); err != nil {
		return nil, err
	}
	abiPathMatches := abiPathRegexp.FindStringSubmatch(stderr.String())
	if len(abiPathMatches) != 2 {
		return nil, fmt.Errorf("can't parse ABI path %q", stderr.String())
	}
	abiData, err := os.ReadFile(abiPathMatches[1])
	if err != nil {
		return nil, err
	}
	if contract.Abi, err = abi.JSON(bytes.NewReader(abiData)); err != nil {
		return nil, err
	}
	contract.Hashes = make(map[string][]byte)
	for _, method := range contract.Abi.Methods {
		contract.Hashes[method.Sig] = method.ID
	}
	return
}

func makePrivateKeyGetter(
	wallets ...*solana.Wallet) func(key solana.PublicKey) *solana.PrivateKey {
	return func(key solana.PublicKey) *solana.PrivateKey {
		for _, wallet := range wallets {
			if wallet.PublicKey().Equals(key) {
				return &wallet.PrivateKey
			}
		}
		return nil
	}
}

type SolangSeed struct {
	seed []byte
	// address solana.PublicKey
}

func encodeSeeds(seeds ...SolangSeed) []byte {
	var length uint64 = 1
	for _, seed := range seeds {
		length += uint64(len(seed.seed)) + 1
	}
	seedEncoded := make([]byte, 0, length)

	seedEncoded = append(seedEncoded, uint8(len(seeds)))
	for _, seed := range seeds {
		seedEncoded = append(seedEncoded, uint8(len(seed.seed)))
		seedEncoded = append(seedEncoded, seed.seed...)
	}

	return seedEncoded
}

func (s *SolanaWorkloadGenerator) CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error) {
	s.logger.Debug("CreateContractDeployTX", zap.Binary("fromPrivKey", fromPrivKey), zap.String("contractPath", contractPath))

	priv := &solana.Wallet{PrivateKey: solana.PrivateKey(fromPrivKey)}

	// Check for the existence of the contract
	if _, err := os.Stat(contractPath); err == nil {
		// Path exists, compile the contract and prepare the transaction
		// TODO: check the 'solang' string
		contract, err := s.compileSolidity(contractPath)
		if err != nil {
			return []byte{}, err
		}
		if contract == nil {
			return nil, fmt.Errorf("no contracts to compile")
		}

		// TODO handle case where number of contracts is greater than one

		s.logger.Info("Deploying Contract",
			zap.String("contract", s.BenchConfig.ContractInfo.Name),
			zap.String("path", s.BenchConfig.ContractInfo.Path),
		)

		blockhash, err := s.ActiveConn.rpcClient.GetRecentBlockhash(
			context.Background(),
			rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}

		programAccount := solana.NewWallet()
		storageAccount := solana.NewWallet()
		lamports, err := s.ActiveConn.rpcClient.GetMinimumBalanceForRentExemption(
			context.Background(),
			uint64(len(contract.Data)),
			rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}

		// 1 - create program account
		// 2 - call loader writes
		// 3 - call loader finalize
		// 4 - create storage account and call contract constructor
		transactionBatches := make([][]*solana.Transaction, 4)

		createTransaction := func(instructions ...solana.Instruction) (*solana.Transaction, error) {
			tx, err := solana.NewTransaction(
				instructions,
				blockhash.Value.Blockhash,
				solana.TransactionPayer(priv.PublicKey()))
			if err != nil {
				return nil, err
			}
			_, err = tx.Sign(makePrivateKeyGetter(priv, programAccount, storageAccount))
			if err != nil {
				return nil, err
			}
			return tx, nil
		}

		transaction, err := createTransaction(
			system.NewCreateAccountInstruction(
				lamports,
				uint64(len(contract.Data)),
				solana.BPFLoaderProgramID,
				priv.PublicKey(),
				programAccount.PublicKey(),
			).Build())
		if err != nil {
			return nil, err
		}
		transactionBatches[0] = append(transactionBatches[0], transaction)

		createInstruction := func(offset int, chunk []byte) *solana.GenericInstruction {
			data := make([]byte, len(chunk)+16)
			binary.LittleEndian.PutUint32(data[0:], 0)
			binary.LittleEndian.PutUint32(data[4:], uint32(offset))
			binary.LittleEndian.PutUint32(data[8:], uint32(len(chunk)))
			binary.LittleEndian.PutUint32(data[12:], 0)
			copy(data[16:], chunk)
			return solana.NewInstruction(
				solana.BPFLoaderProgramID,
				solana.AccountMetaSlice{
					solana.NewAccountMeta(programAccount.PublicKey(), true, true),
				},
				data,
			)
		}

		chunkSize, err := calculateMaxChunkSize(func(offset int, chunk []byte) (*solana.Transaction, error) {
			return solana.NewTransaction(
				[]solana.Instruction{createInstruction(offset, chunk)},
				blockhash.Value.Blockhash,
				solana.TransactionPayer(priv.PublicKey()))
		})
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(contract.Data); i += chunkSize {
			end := i + chunkSize
			if end > len(contract.Data) {
				end = len(contract.Data)
			}
			transaction, err = createTransaction(createInstruction(i, contract.Data[i:end]))
			if err != nil {
				return nil, err
			}
			transactionBatches[1] = append(transactionBatches[1], transaction)
		}

		{
			data := make([]byte, 4)
			binary.LittleEndian.PutUint32(data[0:], 1)
			transaction, err = createTransaction(solana.NewInstruction(
				solana.BPFLoaderProgramID,
				solana.AccountMetaSlice{
					solana.NewAccountMeta(programAccount.PublicKey(), true, true),
				},
				data,
			))
			if err != nil {
				return nil, err
			}
			transactionBatches[2] = append(transactionBatches[2], transaction)
		}

		lamports, err = s.ActiveConn.rpcClient.GetMinimumBalanceForRentExemption(
			context.Background(),
			contract.RequiredSpace,
			rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}

		// assuming that constructor does not have arguments
		{
			input, err := contract.Abi.Constructor.Inputs.Pack()
			if err != nil {
				return nil, err
			}

			hash := crypto.Keccak256([]byte(contract.Name))

			value := make([]byte, 8)
			binary.LittleEndian.PutUint64(value[0:], 0)

			data := []byte{}
			data = append(data, storageAccount.PublicKey().Bytes()...)
			data = append(data, priv.PublicKey().Bytes()...)
			data = append(data, value...)
			data = append(data, hash[:4]...)
			data = append(data, encodeSeeds()...)
			data = append(data, input...)

			transaction, err = createTransaction(
				system.NewCreateAccountInstruction(
					lamports,
					contract.RequiredSpace,
					programAccount.PublicKey(),
					priv.PublicKey(),
					storageAccount.PublicKey()).Build(),
				solana.NewInstruction(
					programAccount.PublicKey(),
					[]*solana.AccountMeta{
						solana.NewAccountMeta(
							storageAccount.PublicKey(),
							true,
							false),
					}, data))
			if err != nil {
				return nil, err
			}
			transactionBatches[3] = append(transactionBatches[3], transaction)
		}

		s.CompiledContract = &SolangDeployedContract{Contract: contract, ProgramAccount: programAccount, StorageAccount: storageAccount}

		return json.Marshal(transactionBatches)
	} else if os.IsNotExist(err) {
		// Path doesn't exist - return an error
		return []byte{}, fmt.Errorf("contract does not exist: %s", contractPath)
	} else {
		// Corner case, it's another error - so we should handle it
		// like an error state
		return []byte{}, err
	}
}

// getCallDataBytes will return the ABI encoded bytes for the variable, or an error
// if it cannot be converted
func getCallDataBytes(paramType string, val string) ([]byte, error) {

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
	case "uint256", "uint":
		// uint 256 = 64 bytes
		//  padding = 0
		num, ok := big.NewInt(0).SetString(val, 10)
		if !ok {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
		}
		payloadBytes = append(payloadBytes, convertU256(num)...)
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
	case "int256", "int":
		num, ok := big.NewInt(0).SetString(val, 10)
		if !ok {
			return nil, fmt.Errorf("failed to convert contract arg %s into %s", val, paramType)
		}
		payloadBytes = append(payloadBytes, num.Bytes()...)
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
	// address
	case "address":
		// uint160
		// get the address
		addr := common.HexToAddress(val)
		// padding - address bytes should be 20bytes long.
		padding := make([]byte, 12)
		payloadBytes = append(payloadBytes, padding...)
		payloadBytes = append(payloadBytes, addr.Bytes()...)
	// bytes
	case "bytes24":
		// TODO this needs improvement!
		s := []byte(val)
		padding := make([]byte, 32-len(s))
		payloadBytes = append(payloadBytes, s...)
		payloadBytes = append(payloadBytes, padding...)
	case "bytes32":
		// TODO this needs improvement!
		s := []byte(val)
		payloadBytes = append(payloadBytes, s...)

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
	// Default
	default:
		return nil, fmt.Errorf("invalid arg type: %s", paramType)
	}

	return payloadBytes, nil
}

func (s *SolanaWorkloadGenerator) CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam, value string) ([]byte, error) {
	s.logger.Debug("CreateInteractionTX", zap.Binary("fromPrivKey", fromPrivKey), zap.String("contractAddress", contractAddress), zap.String("functionName", functionName), zap.Any("contractParams", contractParams), zap.String("value", value))
	// Check that the contract has been compiled, if nto - then it's difficult to get the hashes from the ABI.
	if s.CompiledContract == nil {
		return nil, fmt.Errorf("contract does not exist in known generator")
	}

	// If there are empty params, warn - just because this isn't super common
	if len(contractParams) < 1 {
		// empty
		zap.L().Warn(fmt.Sprintf("empty contract params for %s", functionName))
	}

	// next - get the function hash
	var payloadBytes []byte

	// If we are targeting the fallback function, or, just sending ether - we can ignore the
	// function name.
	if functionName != "fallback" && functionName != "receive" && functionName != "()" {
		val, ok := s.CompiledContract.Contract.Hashes[functionName]
		if !ok {
			zap.L().Debug("Failed to find function",
				zap.String("ABI", fmt.Sprintf("%v", s.CompiledContract.Contract.Abi)),
				zap.String("Function", fmt.Sprintf("%v", functionName)))
			return nil, fmt.Errorf("contract does not contain function: %s", functionName)
		}
		payloadBytes = val
	}
	// Now we need to parse the arguments to get them into the correct padding

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
			encB, err := getCallDataBytes(v.Type, v.Value)
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
			encB, err := getCallDataBytes(v.Type, v.Value)
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
		zap.L().Warn("no payload generated, sending transaction with 0 data bytes")
	}

	// Create the signed transaction
	if value == "" {
		value = "0"
	}
	sendVal, ok := big.NewInt(0).SetString(value, 16)
	if !ok {
		zap.L().Warn(fmt.Sprintf("Failed to set value of tx, could not convert %s to big number", value))
	}

	tx, err := s.CreateSignedTransaction(fromPrivKey, contractAddress, sendVal, payloadBytes)

	if err != nil {
		return nil, err
	}

	// return the transaction
	return tx, nil
}

func (s *SolanaWorkloadGenerator) CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error) {
	s.logger.Debug("CreateSignedTransaction", zap.Binary("fromPrivKey", fromPrivKey), zap.String("toAddress", toAddress), zap.Any("value", value), zap.Binary("data", data))

	priv := &solana.Wallet{PrivateKey: solana.PrivateKey(fromPrivKey)}

	var instruction solana.Instruction
	var privateKeyGetter func(solana.PublicKey) *solana.PrivateKey

	if s.CompiledContract != nil {
		valueBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(valueBytes[0:], value.Uint64())

		instructionData := []byte{}
		instructionData = append(instructionData, s.CompiledContract.StorageAccount.PublicKey().Bytes()...)
		instructionData = append(instructionData, priv.PublicKey().Bytes()...)
		instructionData = append(instructionData, valueBytes...)
		instructionData = append(instructionData, make([]byte, 4)...)
		instructionData = append(instructionData, encodeSeeds()...)
		instructionData = append(instructionData, data...)

		instruction = solana.NewInstruction(
			s.CompiledContract.ProgramAccount.PublicKey(),
			[]*solana.AccountMeta{
				solana.NewAccountMeta(
					s.CompiledContract.StorageAccount.PublicKey(),
					true,
					false),
				solana.NewAccountMeta(
					solana.SysVarClockPubkey,
					false,
					false),
				solana.NewAccountMeta(
					solana.PublicKey{},
					false,
					false),
			}, instructionData)
		privateKeyGetter = makePrivateKeyGetter(priv, s.CompiledContract.StorageAccount)
	} else {
		instruction = system.NewTransferInstruction(
			1,
			priv.PublicKey(),
			solana.MustPublicKeyFromBase58(toAddress)).Build()
		privateKeyGetter = makePrivateKeyGetter(priv)
	}

	blockhash, err := s.ActiveConn.rpcClient.GetRecentBlockhash(
		context.Background(),
		rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		blockhash.Value.Blockhash,
		solana.TransactionPayer(priv.PublicKey()))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(privateKeyGetter)
	if err != nil {
		return nil, err
	}

	return json.Marshal(tx)
}

func (s *SolanaWorkloadGenerator) generateSimpleWorkload() (Workload, error) {

	// get the known accounts
	var totalWorkload Workload

	// 1. Set up the accounts into buckets for each
	accountDistribution := make([][]*configs.ChainKey, s.BenchConfig.Secondaries*s.BenchConfig.Threads)

	accountCount := 0
	for {
		// exit condition - each thread has an assigned account, and we've run out of accounts.
		if accountCount >= len(s.KnownAccounts) && accountCount >= len(accountDistribution) {
			break
		}

		currentAccount := accountCount % len(s.KnownAccounts)
		currentDist := accountCount % len(accountDistribution)

		accountDistribution[currentDist] = append(accountDistribution[currentDist], &s.KnownAccounts[currentAccount])

		accountCount++
	}

	// 2. Generate the transactions
	txID := 0
	accountBatch := 0
	for secondaryID := 0; secondaryID < s.BenchConfig.Secondaries; secondaryID++ {
		// secondaryWorkload = [thread][interval][tx=[]byte]
		// [][][][]byte
		secondaryWorkload := make(SecondaryWorkload, 0)
		for thread := 0; thread < s.BenchConfig.Threads; thread++ {
			// Thread workload = list of transactions in intervals
			// [interval][tx] = [][][]byte
			threadWorkload := make(WorkerThreadWorkload, 0)
			// for each thread, generate the intervals of transactions.
			zap.L().Debug("Info",
				zap.Int("secondary", secondaryID),
				zap.Int("thread", thread),
				zap.Int("len", len(accountDistribution)))
			accountsChoices := accountDistribution[accountBatch]
			for interval, txnum := range s.TPSIntervals {
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

					// accFrom := secondaryID + thread + (secondaryID * s.BenchConfig.Threads)
					accFrom := accountsChoices[txID%len(accountsChoices)]
					accTo := accountsChoices[(txID+1)%len(accountsChoices)]

					tx, txerr := s.CreateSignedTransaction(
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

func (s *SolanaWorkloadGenerator) generateContractWorkload() (Workload, error) {
	s.logger.Debug("generateContractWorkload")
	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(s.BenchConfig)
	if err != nil {
		return nil, err
	}

	// Deploy the contract
	contractAddr, err := s.DeployContract(s.KnownAccounts[0].PrivateKey, s.BenchConfig.ContractInfo.Path)

	if err != nil {
		return nil, err
	}

	// List of functions to create for each thread
	// TODO this needs some tuning!
	// This is a list of [id] pointing to each function
	// It will occur K times in the list, which is representative of the
	// ratio.
	functionsToCreatePerThread := make([]int, numberOfTransactions)

	for idx, funcInfo := range s.BenchConfig.ContractInfo.Functions {
		// add index to functionsToCreate
		funcRatio := (funcInfo.Ratio / 100) * numberOfTransactions

		for i := 0; i < funcRatio; i++ {
			functionsToCreatePerThread = append(functionsToCreatePerThread, idx)
		}
	}

	// 1. Set up the accounts into buckets for each
	accountDistribution := make([][]*configs.ChainKey, s.BenchConfig.Secondaries*s.BenchConfig.Threads)

	accountCount := 0
	for {

		// exit condition - each thread has an assigned account, and we've run out of accounts.
		if accountCount >= len(s.KnownAccounts) && accountCount >= len(accountDistribution) {
			break
		}

		currentAccount := accountCount % len(s.KnownAccounts)
		currentDist := accountCount % len(accountDistribution)

		accountDistribution[currentDist] = append(accountDistribution[currentDist], &s.KnownAccounts[currentAccount])

		accountCount++
	}

	// Shuffle the function interactions
	// TODO check this carefully - we may have workloads with dependent transactions in future - maybe add this as a flag in config?
	// ShuffleFunctionCalls(functionsToCreatePerThread)

	// Now generate the workload as usual
	var totalWorkload Workload
	txIndex := 0
	accountBatch := 0
	for secondaryID := 0; secondaryID < s.BenchConfig.Secondaries; secondaryID++ {
		secondaryWorkload := make(SecondaryWorkload, 0)
		for threadID := 0; threadID < s.BenchConfig.Threads; threadID++ {
			threadWorkload := make(WorkerThreadWorkload, 0)
			txCount := 0

			accountsChoices := accountDistribution[accountBatch]

			// 			for _, numTx := range e.BenchConfig.TxInfo.Intervals {
			for _, numTx := range s.TPSIntervals {
				intervalWorkload := make([][]byte, 0)

				for i := 0; i < numTx; i++ {
					// function to create
					accFrom := accountsChoices[txIndex%len(accountsChoices)]
					funcToCreate := s.BenchConfig.ContractInfo.Functions[functionsToCreatePerThread[txCount]]
					s.logger.Debug(fmt.Sprintf("tx %d for func %s", txCount, funcToCreate.Name),
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
						functionFinal = funcToCreate.Name
					}

					tx, txerr := s.CreateInteractionTX(
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

func (s *SolanaWorkloadGenerator) generatePremadeWorkload() (Workload, error) {
	// 1 deploy the contract if it is a contract workload, get the address
	var contractAddr string
	if len(s.BenchConfig.ContractInfo.Path) > 0 && len(s.BenchConfig.ContractInfo.Name) > 0 {
		// Deploy the contract
		var err error
		contractAddr, err = s.DeployContract(s.KnownAccounts[0].PrivateKey, s.BenchConfig.ContractInfo.Path)

		if err != nil {
			return nil, err
		}
	}

	var fullWorkload Workload
	// 2 loop through the premade dataset and create the relevant transactions
	for secondaryIndex, secondaryWorkload := range s.BenchConfig.TxInfo.PremadeInfo {

		secondaryTransactions := make(SecondaryWorkload, 0)

		for threadIndex, threadWorkload := range secondaryWorkload {

			threadTransactions := make(WorkerThreadWorkload, 0)

			for intervalIndex, intervalWorkload := range threadWorkload {

				intervalTransactions := make([][]byte, 0)

				for _, txInfo := range intervalWorkload {
					// Make the transaction based on its
					fromID, err := strconv.Atoi(txInfo.From)
					fromAccount := s.KnownAccounts[fromID%len(s.KnownAccounts)]
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
						toAccount = s.KnownAccounts[toID%len(s.KnownAccounts)].Address
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
						return nil, fmt.Errorf("failed to set value to big int: %s", txInfo.Value)
					}

					if txInfo.Function == "" && len(txInfo.DataParams) == 0 {
						// This is a simple transaction
						finalTx, err = s.CreateSignedTransaction(
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
							finalTx, err = s.CreateContractDeployTX(
								fromAccount.PrivateKey,
								s.BenchConfig.ContractInfo.Path,
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

							finalTx, err = s.CreateInteractionTX(
								fromAccount.PrivateKey,
								toAccount,
								functionFinal,
								txParams,
								txInfo.Value,
							)
							if err != nil {
								return nil, err
							}
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

func (s *SolanaWorkloadGenerator) GenerateWorkload() (Workload, error) {
	s.logger.Debug("GenerateWorkload")
	// 1/ work out the total number of secondaries.
	numberOfWorkers := s.BenchConfig.Secondaries * s.BenchConfig.Threads

	// Get the number of transactions to be created
	numberOfTransactions, err := parsers.GetTotalNumberOfTransactions(s.BenchConfig)

	if err != nil {
		return nil, err
	}

	// Total transactions
	totalTxPerWorker := numberOfTransactions / numberOfWorkers

	s.logger.Info(
		"Generating workload",
		zap.String("workloadType", string(s.BenchConfig.TxInfo.TxType)),
		zap.Int("threadsTotal", numberOfWorkers),
		zap.Int("totalTransactions per worker", totalTxPerWorker),
	)

	// Print a warning about the accounts
	if len(s.KnownAccounts) >= s.BenchConfig.Secondaries && len(s.KnownAccounts) < s.BenchConfig.Secondaries*s.BenchConfig.Threads {
		s.logger.Warn("Only enough accounts for one per secondary, this means there may be delays/fails for more threads")
	} else if len(s.KnownAccounts) == s.BenchConfig.Secondaries*s.BenchConfig.Threads {
		s.logger.Warn("Workload has one account per thread")
	} else if len(s.KnownAccounts) < (s.BenchConfig.Secondaries * s.BenchConfig.Threads) {
		// If there's not enough accounts, send a message saying that some transactions will fail
		s.logger.Warn("Not enough accounts, will experience fails due to sending nonce at incorrect times.", zap.Int("s.BenchConfig.Secondaries * s.BenchConfig.Threads", s.BenchConfig.Secondaries*s.BenchConfig.Threads))
	}

	switch s.BenchConfig.TxInfo.TxType {
	case configs.TxTypeContract:
		return s.generateContractWorkload()
	case configs.TxTypeSimple:
		return s.generateSimpleWorkload()
	case configs.TxTypePremade:
		return s.generatePremadeWorkload()
	default:
		return nil, errors.New("unknown transaction type in config for workload generation")
	}
}
