package ndiem


import (
	"context"
	"diablo-benchmark/core"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/yaml.v3"
	"os"
	"strings"

	"github.com/diem/client-sdk-go/diemclient"
)


const chainId = 4


type BlockchainInterface struct {
}

func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	var key, value, endpoint string
	var envmap map[string][]string
	var builder *BlockchainBuilder
	var client diemclient.Client
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
	client = diemclient.New(chainId, "http://" + endpoint)

	builder = newBuilder(logger, client, context.Background())

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


func addPremadeAccounts(builder *BlockchainBuilder, path string) error {
	var decoder *yaml.Decoder
	var file *os.File
	var keys []string
	var seed []byte
	var key string
	var err error

	file, err = os.Open(path)
	if err != nil {
		return err
	}

	decoder = yaml.NewDecoder(file)
	err = decoder.Decode(&keys)

	file.Close()

	if err != nil {
		return err
	}

	for _, key = range keys {
		seed, err = hex.DecodeString(key)
		if err != nil {
			return err
		}

		builder.addAccount(ed25519.NewKeyFromSeed(seed))
	}

	return nil
}


func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	var confirmer transactionConfirmer
	var preparer transactionPreparer
	var client diemclient.Client
	var key, value string
	var err error

	logger.Tracef("new client")

	logger.Tracef("use endpoint '%s'", view[0])
	client = diemclient.New(chainId, "http://" + view[0])

	for key, value = range params {
		if key == "confirm" {
			logger.Tracef("use confirm method '%s'", value)
			confirmer, err = parseConfirm(value, logger, client)
			if err != nil {
				return nil, err
			}
			continue
		}

		if key == "prepare" {
			logger.Tracef("use prepare method '%s'", value)
			preparer, err = parsePrepare(value, logger, client)
			if err != nil {
				return nil, err
			}
			continue
		}

		return nil, fmt.Errorf("unknown parameter '%s'", key)
	}

	if confirmer == nil {
		logger.Tracef("use default confirm method 'polltx'")
		confirmer = newPolltxTransactionConfirmer(logger, client)
	}

	if preparer == nil {
		logger.Tracef("use default prepare method 'signature'")
		preparer = newSignatureTransactionPreparer(logger)
	}

	return newClient(logger, client, preparer, confirmer), nil
}

func parseConfirm(value string, logger core.Logger, client diemclient.Client) (transactionConfirmer, error) {
	if value == "polltx" {
		return newPolltxTransactionConfirmer(logger, client), nil
	}

	if value == "pollblk" {
		return newPollblkTransactionConfirmer(logger, client), nil
	}

	return nil, fmt.Errorf("unknown confirm method '%s'", value)
}

func parsePrepare(value string, logger core.Logger, client diemclient.Client) (transactionPreparer, error) {
	var preparer transactionPreparer

	if value == "nothing" {
		preparer = newNothingTransactionPreparer()
		return preparer, nil
	}

	if value == "signature" {
		preparer = newSignatureTransactionPreparer(logger)
		return preparer, nil
	}

	return nil, fmt.Errorf("unknown prepare method '%s'", value)
}
