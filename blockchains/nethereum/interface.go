package nethereum


import (
	"context"
	"crypto/ecdsa"
	"diablo-benchmark/core"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)


type BlockchainInterface struct {
}

func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	var key, value, endpoint string
	var builder *BlockchainBuilder
	var envmap map[string][]string
	var client *ethclient.Client
	var values []string
	var err error

	logger.Debugf("new builder")

	envmap, err = parseEnvmap(env)
	if err != nil {
		return nil, err
	}

	for key = range endpoints {
		endpoint = key
		break
	}

	logger.Debugf("use endpoint '%s'", endpoint)
	client, err = ethclient.Dial("ws://" + endpoint)
	if err != nil {
		return nil, err
	}

	builder = newBuilder(logger, client)

	for key, values = range envmap {
		if key == "accounts" {
			for _, value = range values {
				logger.Debugf("with accounts from '%s'", value)

				err = addPremadeAccounts(builder, value)
				if err != nil {
					return nil, err
				}
			}

			continue
		}

		if key == "contracts" {
			for _, value = range values {
				logger.Debugf("with contracts from '%s'",value)
				builder.addCompiler(value)
			}

			continue
		}

		return nil, fmt.Errorf("unknown environment key '%s'", key)
	}

	return builder, nil
}


func parseEnvmap(env []string) (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)
	var element, key, value string
	var values []string
	var eqindex int
	var found bool

	for _, element = range env {
		eqindex = strings.Index(element, "=")
		if eqindex < 0 {
			return nil, fmt.Errorf("unexpected environment '%s'",
				element)
		}

		key = element[:eqindex]
		value = element[eqindex + 1:]

		values, found = ret[key]
		if !found {
			values = make([]string, 0)
		}

		values = append(values, value)

		ret[key] = values
	}

	return ret, nil
}


type yamlAccount struct {
	Address  string  `yaml:"address"`
	Private  string  `yaml:"private"`
}

func addPremadeAccounts(builder *BlockchainBuilder, path string) error {
	var private *ecdsa.PrivateKey
	var accounts []*yamlAccount
	var address common.Address
	var decoder *yaml.Decoder
	var account *yamlAccount
	var file *os.File
	var bytes []byte
	var err error

	file, err = os.Open(path)
	if err != nil {
		return err
	}

	decoder = yaml.NewDecoder(file)
	err = decoder.Decode(&accounts)

	file.Close()

	if err != nil {
		return err
	}

	for _, account = range accounts {
		bytes, err = hex.DecodeString(account.Address)
		if err != nil {
			return err
		}

		if len(bytes) != common.AddressLength {
			return fmt.Errorf("invalid address length (%d bytes)",
				len(bytes))
		}

		address = common.BytesToAddress(bytes)

		private, err = crypto.HexToECDSA(account.Private)
		if err != nil {
			return err
		}

		builder.addAccount(address, private)
	}

	return nil
}


func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	var ctx context.Context = context.Background()
	var confirmer transactionConfirmer
	var preparer transactionPreparer
	var provider parameterProvider
	var client *ethclient.Client
	var manager nonceManager
	var key, value string
	var err error

	logger.Tracef("new client")

	logger.Tracef("use endpoint '%s'", view[0])
	client, err = ethclient.Dial("ws://" + view[0])
	if err != nil {
		return nil, err
	}

	for key, value = range params {
		if key == "prepare" {
			logger.Tracef("use prepare method '%s'", value)
			provider, preparer, err =
				parsePrepare(value, logger, client)
			if err != nil {
				return nil, err
			}
			continue
		}

		return nil, fmt.Errorf("unknown parameter '%s'", key)
	}

	if (provider == nil) && (preparer == nil) {
		logger.Tracef("use default prepare method 'signature'")

		provider, err = makeStaticParameterProvider(client)
		if err != nil {
			return nil, err
		}

		preparer = newSignatureTransactionPreparer(logger)
	}

	manager = newStaticNonceManager(logger, client)
	confirmer = newPollblkTransactionConfirmer(logger, client, ctx)

	return newClient(logger, client, manager, provider, preparer,
		confirmer), nil
}

func parsePrepare(value string, logger core.Logger, client *ethclient.Client) (parameterProvider, transactionPreparer, error) {
	var preparer transactionPreparer
	var provider parameterProvider
	var err error

	if value == "nothing" {
		provider = newDirectParameterProvider(client)
		if err != nil {
			return nil, nil, err
		}

		preparer = newNothingTransactionPreparer()

		return provider, preparer, nil
	}

	if value == "params" {
		provider, err = makeStaticParameterProvider(client)
		if err != nil {
			return nil, nil, err
		}

		preparer = newNothingTransactionPreparer()

		return provider, preparer, nil
	}

	if value == "signature" {
		provider, err = makeStaticParameterProvider(client)
		if err != nil {
			return nil, nil, err
		}

		preparer = newSignatureTransactionPreparer(logger)

		return provider, preparer, nil
	}

	return nil, nil, fmt.Errorf("unknown prepare method '%s'", value)
}
