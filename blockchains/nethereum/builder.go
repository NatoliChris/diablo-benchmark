package nethereum


import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"diablo-benchmark/core"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)


type BlockchainBuilder struct {
	logger           core.Logger
	client           *ethclient.Client
	ctx              context.Context
	premadeAccounts  []account
	usedAccounts     int
	compilers        []*solidityCompiler
	applications     map[string]*application
	manager          nonceManager
	provider         parameterProvider
}

type account struct {
	address  common.Address
	private  *ecdsa.PrivateKey
	nonce    uint64
}

type contract struct {
	appli  *application
	appid  common.Address
}


func newBuilder(logger core.Logger, client *ethclient.Client) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger: logger,
		client: client,
		ctx: context.Background(),
		premadeAccounts: make([]account, 0),
		usedAccounts: 0,
		compilers: make([]*solidityCompiler, 0),
		applications: make(map[string]*application),
		manager: newStaticNonceManager(logger, client),
		provider: newLazyParameterProvider(client),
	}
}

func (this *BlockchainBuilder) addAccount(address common.Address, private *ecdsa.PrivateKey) {
	this.premadeAccounts = append(this.premadeAccounts, account{
		address: address,
		private: private,
		nonce: 0,
	})
}

func (this *BlockchainBuilder) addCompiler(path string) {
	var compiler *solidityCompiler

	compiler = newSolidityCompiler(this.logger, path)

	this.compilers = append(this.compilers, compiler)
}


func (this *BlockchainBuilder) getBuilderAccount() (*account, error) {
	if len(this.premadeAccounts) > 0 {
		return &this.premadeAccounts[0], nil
	} else {
		return nil, fmt.Errorf("no available premade accounts")
	}
}

func (this *BlockchainBuilder) getApplication(name string) (*application, error) {
	var compiler *solidityCompiler
	var appli *application
	var err error
	var ok bool

	appli, ok = this.applications[name]
	if ok {
		return appli, nil
	}

	for _, compiler = range this.compilers {
		appli, err = compiler.compile(name)

		if err == nil {
			break
		} else {
			this.logger.Debugf("failed to compile '%s': %s",
				name, err.Error())
		}
	}

	if appli == nil {
		return nil, fmt.Errorf("failed to compile contract '%s'", name)
	}

	this.applications[name] = appli

	return appli, nil
}


func (this *BlockchainBuilder) CreateAccount(int) (interface{}, error) {
	var ret *account

	if this.usedAccounts < len(this.premadeAccounts) {
		ret = &this.premadeAccounts[this.usedAccounts]
		this.usedAccounts += 1
	} else {
		return nil, fmt.Errorf("can only use %d premade accounts",
			this.usedAccounts)
	}

	return ret, nil
}

func (this *BlockchainBuilder) CreateContract(name string) (interface{}, error) {
	var tx *deployContractTransaction
	var stx *types.Transaction
	var receipt *types.Receipt
	var appli *application
	var from *account
	var err error

	appli, err = this.getApplication(name)
	if err != nil {
		return nil, err
	}

	from, err = this.getBuilderAccount()
	if err != nil {
		return nil, err
	}

	tx = newDeployContractTransaction(from.nonce, appli, from.private,
		this.manager, this.provider)

	_, stx, err = tx.getTx()
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("deploy new contract '%s'", name)

	receipt, err = this.submitTransaction(stx)
	if err != nil {
		return nil, err
	}

	from.nonce += 1

	this.logger.Tracef("new contract '%s' deployed with id %s", name,
		receipt.ContractAddress.String())

	return &contract{
		appli: appli,
		appid: receipt.ContractAddress,
	}, nil
}

func (this *BlockchainBuilder) submitTransaction(stx *types.Transaction) (*types.Receipt, error) {
	var receipt *types.Receipt
	var hash common.Hash
	var err error

	err = this.client.SendTransaction(this.ctx, stx)
	if err != nil {
		return nil, err
	}

	hash = stx.Hash()

	for {
		time.Sleep(1 * time.Second)

		receipt, err = this.client.TransactionReceipt(this.ctx, hash)
		if err == nil {
			break
		}

		if err == ethereum.NotFound {
			continue
		}

		return nil, err
	}

	return receipt, nil
}

func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	var tx *transferTransaction
	var buffer bytes.Buffer
	var faccount *account
	var err error

	faccount = from.(*account)

	tx = newTransferTransaction(faccount.nonce, uint64(amount),
		faccount.private, to.(*account).address, nil, nil)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	faccount.nonce += 1

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInvoke(from, to interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	var tx *invokeTransaction
	var buffer bytes.Buffer
	var tcontract *contract
	var faccount *account
	var payload []byte
	var err error

	faccount = from.(*account)
	tcontract = to.(*contract)

	payload, err = tcontract.appli.arguments(function)
	if err != nil {
		return nil, err
	}

	tx = newInvokeTransaction(faccount.nonce, faccount.private,
		tcontract.appid, payload, nil, nil)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	faccount.nonce += 1

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return nil, fmt.Errorf("unknown interaction type '%s'", itype)
}
