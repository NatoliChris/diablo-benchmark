package nalgorand


import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"fmt"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"

	"golang.org/x/crypto/ed25519"
)


type BlockchainBuilder struct {
	logger           core.Logger
	client           *algod.Client
	ctx              context.Context
	premadeAccounts  []account
	usedAccounts     int
	compilers        []*tealCompiler
	applications     map[string]*application
	provider         parameterProvider
	lastRound        uint64
	submitMaxTry     int
	nextTxuid        uint64
}

type account struct {
	address   string
	key       ed25519.PrivateKey
}

type contract struct {
	app    *application
	appid  uint64
}


func newBuilder(logger core.Logger, client *algod.Client, ctx context.Context) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger: logger,
		client: client,
		ctx: ctx,
		premadeAccounts: make([]account, 0),
		usedAccounts: 0,
		compilers: make([]*tealCompiler, 0),
		applications: make(map[string]*application),
		provider: newLazyParameterProvider(client),
		lastRound: 0,
		submitMaxTry: 10,
		nextTxuid: 0,
	}
}

func (this *BlockchainBuilder) getLogger() core.Logger {
	return this.logger
}

func (this *BlockchainBuilder) addAccount(address string,
	key ed25519.PrivateKey) {
	this.premadeAccounts = append(this.premadeAccounts, account{
		address: address,
		key: key,
	})
}

func (this *BlockchainBuilder) addCompiler(path string) {
	var compiler *tealCompiler

	compiler = newTealCompiler(this.logger, path, this.client, this.ctx)

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
	var compiler *tealCompiler
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
	var info *models.PendingTransactionInfoResponse
	var tx *deployContractTransaction
	var appli *application
	var from *account
	var appid uint64
	var raw []byte
	var err error

	from, err = this.getBuilderAccount()
	if err != nil {
		return nil, err
	}

	appli, err = this.getApplication(name)
	if err != nil {
		return nil, err
	}

	tx = newDeployContractTransaction(uint64(this.nextTxuid), appli,
		from.address, from.key, this.provider)

	_, raw, err = tx.getRaw()
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("deploy new contract '%s'", name)

	info, err = this.submitTransaction(raw)
	if err != nil {
		return nil, err
	}

	this.nextTxuid += 1
	appid = info.ApplicationIndex

	this.logger.Tracef("new contract '%s' deployed with id %d", name,appid)

	return &contract{ appli, appid }, nil
}

func (this *BlockchainBuilder) submitTransaction(raw []byte) (*models.PendingTransactionInfoResponse, error) {
	var info models.PendingTransactionInfoResponse
	var cli *algod.Client = this.client
	var c context.Context = this.ctx
	var status models.NodeStatus
	var txid string
	var try int = 0
	var err error

	txid, err = cli.SendRawTransaction(raw).Do(c)
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("submit new transaction '%s' (%d bytes)", txid,
		len(raw))

	if this.lastRound == 0 {
		status, err = cli.Status().Do(c)
		if err != nil {
			return nil, err
		}

		this.lastRound = status.LastRound
	}

	for {
		try += 1
		if try > this.submitMaxTry {
			break
		}

		this.logger.Tracef("observe transaction '%s' at round %d " +
			"(try %d/%d)", txid, this.lastRound, try,
			this.submitMaxTry)

		info, _, err = cli.PendingTransactionInformation(txid).Do(c)
		if err != nil {
			return nil, err
		}

		if info.PoolError != "" {
			return nil, fmt.Errorf("failed to deploy: %s",
				info.PoolError)
		}

		if info.ConfirmedRound > 0 {
			return &info, nil
		}

		status, err = cli.StatusAfterBlock(this.lastRound).Do(c)
		if err != nil {
			return nil, err
		}

		this.lastRound = status.LastRound
	}

	return nil, fmt.Errorf("failed to deploy: too slow")
}



func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}) ([]byte, error) {
	var tx *transferTransaction
	var buffer bytes.Buffer
	var err error

	tx = newTransferTransaction(uint64(this.nextTxuid), uint64(amount),
		from.(*account).address, to.(*account).address,
		from.(*account).key, nil)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	this.nextTxuid += 1

	return buffer.Bytes(), nil
}


func (this *BlockchainBuilder) EncodeInvoke(from, to interface{}, function string) ([]byte, error) {
	var tx *invokeTransaction
	var buffer bytes.Buffer
	var cont *contract
	var args [][]byte
	var err error

	cont = to.(*contract)

	args, err = cont.app.arguments(function)
	if err != nil {
		return nil, err
	}

	tx = newInvokeTransaction(uint64(this.nextTxuid), cont.appid,
		args, from.(*account).address, from.(*account).key, nil)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	this.nextTxuid += 1

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string) (core.InteractionFactory, bool) {
	return nil, false
}
