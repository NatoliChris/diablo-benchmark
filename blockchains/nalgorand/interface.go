//
// Parameters:
//
//   confirm - Indicate how the client check that a submitted transaction has
//             been committed. Can be one of "polltx" or "pollblk".
//
//             polltx  - Poll the algod process once for each submitted
//                       transaction. This is the most realistic option but
//                       also the most demanding for both the Diablo secondary
//                       nodes and the blockchain nodes.
//
//             pollblk - Poll the algod process once for all transactions by
//                       parsing the committed blocks. This is the most
//                       lightweight option for both Diablo secondary nodes and
//                       blockchain nodes. This is the default value.
//
//   prepare - Indicate how much is computed offline, before the benchmark
//             starts. Can be one of "nothing", "params", "payload" or
//             "signature".
//
//             nothing   - Nothing is computed offline. This is the most
//                         realistic option but also the most demanding for the
//                         Diablo secondary nodes.
//
//             params    - Client parameters (e.g. transaction fee) are queried
//                         once and never updated. This reduces the read load
//                         on the blockchain.
//
//             signature - Transactions are fully packed and signed. This is
//                         the most lightweight option for Diablo secondary
//                         nodes. This is the default value.
//


package nalgorand


import (
	"context"
	"diablo-benchmark/core"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/yaml.v3"
	"os"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/mnemonic"
)


type BlockchainInterface struct {
}

const benchmarkToken =
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"


func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	var ctx context.Context = context.Background()
	var endpoint, key, value string
	var builder *BlockchainBuilder
	var envmap map[string][]string
	var client *algod.Client
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
	client, err = algod.MakeClient("http://" + endpoint, benchmarkToken)
	if err != nil {
		return nil, err
	}

	builder = newBuilder(logger, client, ctx)

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
	Address   string  `yaml:"address"`
	Mnemonic  string  `yaml:"mnemonic"`
}

func addPremadeAccounts(builder *BlockchainBuilder, path string) error {
	var key ed25519.PrivateKey
	var decoder *yaml.Decoder
	var ret []*yamlAccount
	var acc *yamlAccount
	var file *os.File
	var err error

	file, err = os.Open(path)
	if err != nil {
		return err
	}

	decoder = yaml.NewDecoder(file)
	err = decoder.Decode(&ret)

	file.Close()

	if err != nil {
		return err
	}

	for _, acc = range ret {
		key, err = mnemonic.ToPrivateKey(acc.Mnemonic)
		if err != nil {
			return err
		}

		builder.addAccount(acc.Address, key)
	}

	return nil
}


func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	var confirmer transactionConfirmer
	var preparer transactionPreparer
	var provider parameterProvider
	var client *algod.Client
	var ctx context.Context
	var key, value string
	var err error

	ctx = context.Background()

	logger.Tracef("new client")

	logger.Tracef("use endpoint '%s'", view[0])
	client, err = algod.MakeClient("http://" + view[0], benchmarkToken)
	if err != nil {
		return nil, err
	}

	for key, value = range params {
		if key == "confirm" {
			logger.Tracef("use confirm method '%s'", value)
			confirmer, err = parseConfirm(value, logger,client,ctx)
			if err != nil {
				return nil, err
			}
			continue
		}

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

	if confirmer == nil {
		logger.Tracef("use default confirm method 'polltx'")
		confirmer = newPolltxTransactionConfirmer(logger, client, ctx)
	}

	if (provider == nil) && (preparer == nil) {
		logger.Tracef("use default prepare method 'signature'")

		provider, err = makeStaticParameterProvider(client)
		if err != nil {
			return nil, err
		}

		preparer = newSignatureTransactionPreparer(logger)
	}

	return newClient(logger, client, preparer, provider, confirmer), nil
}

func parseConfirm(value string, logger core.Logger, client *algod.Client, ctx context.Context) (transactionConfirmer, error) {
	if value == "polltx" {
		return newPolltxTransactionConfirmer(logger, client, ctx), nil
	}

	if value == "pollblk" {
		return newPollblkTransactionConfirmer(logger, client, ctx), nil
	}

	return nil, fmt.Errorf("unknown confirm method '%s'", value)
}

func parsePrepare(value string, logger core.Logger, client *algod.Client) (parameterProvider, transactionPreparer, error) {
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
