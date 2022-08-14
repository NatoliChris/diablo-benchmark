package nsolana

import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type BlockchainBuilder struct {
	logger          core.Logger
	client          *rpc.Client
	ctx             context.Context
	commitment      rpc.CommitmentType
	premadeAccounts []account
	usedAccounts    int
	compilers       []*solidityCompiler
	applications    map[string]*application
	provider        parameterProvider
	amounts         map[*account]map[*account]int
}

type account struct {
	private solana.PrivateKey
	public  solana.PublicKey
}

func newAccount(private solana.PrivateKey) *account {
	public := private.PublicKey()
	return &account{private: private, public: public}
}

type contract struct {
	appli   *application
	program *account
	storage *account
}

func newBuilder(logger core.Logger, client *rpc.Client) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger:          logger,
		client:          client,
		ctx:             context.Background(),
		commitment:      rpc.CommitmentFinalized,
		premadeAccounts: make([]account, 0),
		usedAccounts:    0,
		compilers:       make([]*solidityCompiler, 0),
		applications:    make(map[string]*application),
		provider:        newDirectParameterProvider(client, context.Background()),
		amounts:         make(map[*account]map[*account]int),
	}
}

func (this *BlockchainBuilder) addAccount(private solana.PrivateKey) {
	this.premadeAccounts = append(this.premadeAccounts, account{
		private: private,
		public:  private.PublicKey(),
	})
}

func (this *BlockchainBuilder) addCompiler(path string) {
	compiler := newSolidityCompiler(this.logger, path)

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
	var appli *application

	appli, ok := this.applications[name]
	if ok {
		return appli, nil
	}

	var err error
	for _, compiler := range this.compilers {
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
	appli, err := this.getApplication(name)
	if err != nil {
		return nil, err
	}

	from, err := this.getBuilderAccount()
	if err != nil {
		return nil, err
	}

	program := newAccount(solana.NewWallet().PrivateKey)
	programLamports, err := this.client.GetMinimumBalanceForRentExemption(
		this.ctx,
		uint64(len(appli.text)),
		this.commitment)
	if err != nil {
		return nil, err
	}

	storage := newAccount(solana.NewWallet().PrivateKey)
	storageLamports, err := this.client.GetMinimumBalanceForRentExemption(
		this.ctx,
		8192*8,
		this.commitment)
	if err != nil {
		return nil, err
	}

	txBatches, err := newDeployContractTransactionBatches(appli, from, program,
		storage, programLamports, storageLamports, this.provider)
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("deploy new contract '%s'", name)

	for _, batch := range txBatches {
		var wg sync.WaitGroup
		results := make([]error, len(batch))
		for idx, tx := range batch {
			wg.Add(1)
			go func(idx int, tx virtualTransaction) {
				defer wg.Done()
				_, stx, err := tx.getTx()
				if err != nil {
					results[idx] = err
					return
				}
				err = this.submitTransaction(stx)
				if err != nil {
					results[idx] = err
					return
				}
			}(idx, tx)
		}
	}

	this.logger.Tracef("new contract '%s' deployed with id %s", name,
		program.public.String())

	return &contract{
		appli:   appli,
		program: program,
		storage: storage,
	}, nil
}

func (this *BlockchainBuilder) submitTransaction(stx *solana.Transaction) error {
	sig, err := this.client.SendTransactionWithOpts(
		this.ctx,
		stx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: this.commitment,
		})
	if err != nil {
		return err
	}

	for {
		time.Sleep(1 * time.Second)

		result, err := this.client.GetSignatureStatuses(this.ctx, true, sig)

		if err != nil {
			if err == rpc.ErrNotFound {
				continue
			}
			return err
		}

		if result != nil &&
			len(result.Value) > 0 &&
			result.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
			break
		}
	}

	return nil
}

func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	if amount != 1 {
		return nil, fmt.Errorf("unexpected amount value")
	}
	faccount := from.(*account)
	taccount := to.(*account)

	if _, ok := this.amounts[faccount]; !ok {
		this.amounts[faccount] = make(map[*account]int)
	}
	this.amounts[faccount][taccount]++
	amount = this.amounts[faccount][taccount]

	tx := newTransferTransaction(uint64(amount),
		faccount.private, &taccount.public, nil)

	var buffer bytes.Buffer
	err := tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInvoke(from, to interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	faccount := from.(*account)
	tcontract := to.(*contract)

	payload, err := tcontract.appli.arguments(function)
	if err != nil {
		return nil, err
	}

	tx := newInvokeTransaction(0, faccount.private,
		&tcontract.program.public, &tcontract.storage.public, payload, nil)

	var buffer bytes.Buffer
	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return nil, fmt.Errorf("unknown interaction type '%s'", itype)
}
