package ndiem


import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"time"

	"github.com/diem/client-sdk-go/diemclient"
	"github.com/diem/client-sdk-go/diemjsonrpctypes"
	"github.com/diem/client-sdk-go/diemkeys"
	"github.com/diem/client-sdk-go/diemtypes"
)


type BlockchainBuilder struct {
	logger           core.Logger
	client           diemclient.Client
	ctx              context.Context
	premadeAccounts  []account
	usedAccounts     int
	builderAccount   *account
	compilers        []*moveCompiler
	applications     map[string]*application
	ownerAccounts    map[string]int
}

type account struct {
	key       ed25519.PrivateKey
	addr      diemtypes.AccountAddress
	sequence  uint64
}

type contract struct {
	app   *application
	addr  *account
}


func newBuilder(logger core.Logger, client diemclient.Client, ctx context.Context) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger: logger,
		client: client,
		ctx: ctx,
		premadeAccounts: make([]account, 0),
		usedAccounts: 0,
		builderAccount: nil,
		compilers: make([]*moveCompiler, 0),
		applications: make(map[string]*application),
		ownerAccounts: make(map[string]int),
	}
}

func (this *BlockchainBuilder) addAccount(key ed25519.PrivateKey) {
	var addr diemtypes.AccountAddress
	var keys *diemkeys.Keys

	keys = diemkeys.NewKeysFromPublicAndPrivateKeys(
		diemkeys.NewEd25519PublicKey(key.Public().(ed25519.PublicKey)),
		diemkeys.NewEd25519PrivateKey(key))

	addr = keys.AccountAddress()

	this.premadeAccounts = append(this.premadeAccounts, account{
		key: key,
		addr: addr,
		sequence: 0,
	})
}

func (this *BlockchainBuilder) addCompiler(path string, stdlibs []string) {
	var compiler *moveCompiler

	compiler = newMoveCompiler(this.logger, path, stdlibs)

	this.compilers = append(this.compilers, compiler)
}

func (this *BlockchainBuilder) initAccount(account *account) error {
	var acc *diemjsonrpctypes.Account
	var err error

	acc, err = this.client.GetAccount(account.addr)
	if err != nil {
		return err
	}

	if acc == nil {
		return fmt.Errorf("account does not exist")
	}

	account.sequence = acc.SequenceNumber

	return nil
}

func (this *BlockchainBuilder) getAccount(index *int) (*account, error) {
	var ret *account
	var err error

	if *index < len(this.premadeAccounts) {
		ret = &this.premadeAccounts[*index]
		*index += 1

		err = this.initAccount(ret)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("can only use %d premade accounts",
			*index)
	}

	return ret, nil
}

func (this *BlockchainBuilder) getBuilderAccount() (*account, error) {
	var index int = 0
	return this.getAccount(&index)
}

func (this *BlockchainBuilder) getOwnerAccount(name string) (*account, error) {
	var ret *account
	var index int
	var err error
	var ok bool

	index, ok = this.ownerAccounts[name]
	if !ok {
		index = 0
	}

	ret, err = this.getAccount(&index)
	this.ownerAccounts[name] = index

	return ret, err
}

func (this *BlockchainBuilder) CreateAccount(stake int) (interface{}, error) {
	return this.getAccount(&this.usedAccounts)
}

func (this *BlockchainBuilder) getApplication(name string) (*application, error) {
	var compiler *moveCompiler
	var appli *application
	var builder *account
	var err error
	var ok bool

	builder, err = this.getBuilderAccount()
	if err != nil {
		return nil, err
	}

	appli, ok = this.applications[name]
	if ok {
		return appli, nil
	}

	for _, compiler = range this.compilers {
		appli, err = compiler.compile(name, builder)

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

func (this *BlockchainBuilder) getDeployedApplication(name string) (*application, error) {
	var appli *application
	var err error

	appli, err = this.getApplication(name)
	if err != nil {
		return nil, err
	}

	if appli.deployed == false {
		this.logger.Debugf("deploy new module '%s'", name)
		err = this.deployApplication(appli)
		if err != nil {
			return nil, err
		}

		appli.deployed = true
	}

	return appli, nil
}

func (this *BlockchainBuilder) deployApplication(appli *application) error {
	var stx *diemtypes.SignedTransaction
	var tx *deployContractTransaction
	var builder *account
	var err error

	builder, err = this.getBuilderAccount()
	if err != nil {
		return err
	}

	tx = newDeployContractTransaction(builder.key, appli.moduleCode,
		builder.sequence)

	_, stx, err = tx.getSigned()
	if err != nil {
		return err
	}

	err = this.submitTransaction(stx)
	if err != nil {
		return err
	}

	return nil

}

func (this *BlockchainBuilder) CreateContract(name string) (interface{}, error) {
	var stx *diemtypes.SignedTransaction
	var tx *invokeTransaction
	var appli *application
	var owner *account
	var err error

	appli, err = this.getDeployedApplication(name)
	if err != nil {
		return nil, err
	}

	owner, err = this.getOwnerAccount(name)
	if err != nil {
		return nil, err
	}

	tx = newInvokeTransaction(owner.key, appli.ctorCode,
		[]diemtypes.TransactionArgument{}, owner.sequence)

	_, stx, err = tx.getSigned()
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("construct new instance of '%s'", name)

	err = this.submitTransaction(stx)
	if err != nil {
		return nil, err
	}

	return &contract{
		app: appli,
		addr: owner,
	}, nil
}

func (this *BlockchainBuilder) submitTransaction(stx *diemtypes.SignedTransaction) error {
	var state *diemjsonrpctypes.Transaction
	var err error

	err = this.client.SubmitTransaction(stx)
	if err != nil {
		return err
	}

	state, err = this.client.WaitForTransaction2(stx, 30 * time.Second)
	if err != nil {
		return err
	}

	if state.VmStatus.GetType() != "executed" {
		return fmt.Errorf("transaction failed to execute (%s)",
			state.VmStatus.GetType())
	}

	return nil
}

func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	var tx *transferTransaction
	var buffer bytes.Buffer
	var err error

	tx = newTransferTransaction(from.(*account).key, to.(*account).addr,
		uint64(amount), from.(*account).sequence)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	from.(*account).sequence += 1

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInvoke(from interface{}, contr interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	var args *applicationArguments
	var tx *invokeTransaction
	var buffer bytes.Buffer
	var cont *contract
	var err error

	cont = contr.(*contract)

	args, err = cont.app.arguments(function, cont.addr.addr)
	if err != nil {
		return nil, err
	}

	tx = newInvokeTransaction(from.(*account).key, args.funccode,
		args.funcargs, from.(*account).sequence)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	from.(*account).sequence += 1

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return []byte{}, nil
}
