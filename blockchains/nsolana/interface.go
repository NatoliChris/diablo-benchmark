package nsolana

import (
	"bufio"
	"context"
	"diablo-benchmark/core"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

type BlockchainInterface struct {
}

func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	logger.Debugf("new builder")

	envmap, err := parseEnvmap(env)
	if err != nil {
		return nil, err
	}

	var endpoint string
	for key := range endpoints {
		endpoint = key
		break
	}

	logger.Debugf("use endpoint '%s'", endpoint)
	client := rpc.New("http://" + endpoint)

	builder := newBuilder(logger, client)

	for key, values := range envmap {
		if key == "accounts" {
			for _, value := range values {
				logger.Debugf("with accounts from '%s'", value)

				err = addPremadeAccounts(builder, value)
				if err != nil {
					return nil, err
				}
			}

			continue
		}

		if key == "contracts" {
			for _, value := range values {
				logger.Debugf("with contracts from '%s'", value)
				builder.addCompiler(value)
			}

			continue
		}

		return nil, fmt.Errorf("unknown environment key '%s'", key)
	}

	return builder, nil
}

func parseEnvmap(env []string) (map[string][]string, error) {
	ret := make(map[string][]string)

	for _, element := range env {
		eqindex := strings.Index(element, "=")
		if eqindex < 0 {
			return nil, fmt.Errorf("unexpected environment '%s'",
				element)
		}

		key := element[:eqindex]
		value := element[eqindex+1:]

		values, found := ret[key]
		if !found {
			values = make([]string, 0)
		}

		values = append(values, value)

		ret[key] = values
	}

	return ret, nil
}

func addPremadeAccounts(builder *BlockchainBuilder, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if line[1] == '[' {
			var private solana.PrivateKey
			err := json.Unmarshal(line[1:len(line)-2], &private)
			if err != nil {
				return err
			}

			builder.addAccount(private)
		}
	}

	if err = file.Close(); err != nil {
		return err
	}

	return nil
}

func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	ctx := context.Background()

	logger.Tracef("new client")

	logger.Tracef("use endpoint '%s'", view[0])
	client := rpc.New("http://" + view[0])

	ip, portStr, err := net.SplitHostPort(view[0])
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	sock, err := ws.Connect(context.Background(), fmt.Sprintf("ws://%s", net.JoinHostPort(ip, strconv.Itoa(port+1))))
	if err != nil {
		return nil, err
	}

	observerProvider := newObserverParameterProvider()
	confirmer := newPollblkTransactionConfirmer(logger, client, sock, ctx, observerProvider)

	var provider parameterProvider
	var preparer transactionPreparer
	for key, value := range params {
		if key == "prepare" {
			logger.Tracef("use prepare method '%s'", value)
			provider, preparer, err =
				parsePrepare(value, logger, client, ctx, observerProvider)
			if err != nil {
				return nil, err
			}
			continue
		}

		return nil, fmt.Errorf("unknown parameter '%s'", key)
	}

	if (provider == nil) && (preparer == nil) {
		logger.Tracef("use default prepare method 'observer'")

		provider = observerProvider

		preparer = newNothingTransactionPreparer()
	}

	return newClient(logger, client, provider, preparer, confirmer), nil
}

func parsePrepare(value string, logger core.Logger, client *rpc.Client, ctx context.Context, observerProvider *observerParameterProvider) (parameterProvider, transactionPreparer, error) {
	var preparer transactionPreparer
	var provider parameterProvider

	if value == "nothing" {
		provider = newDirectParameterProvider(client, ctx)

		preparer = newNothingTransactionPreparer()

		return provider, preparer, nil
	}

	if value == "observer" {
		provider = observerProvider

		preparer = newNothingTransactionPreparer()

		return provider, preparer, nil
	}

	return nil, nil, fmt.Errorf("unknown prepare method '%s'", value)
}
