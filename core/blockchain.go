package core


import (
	"fmt"
)


type Interaction interface {
	Payload() interface{}

	ReportSubmit()

	ReportCommit()

	ReportAbort()
}


type InteractionInfo interface {
	// Return the schedule sedning time of the interaction relative to the
	// beginning of the benchmark.
	//
	Timestamp() float64
}


type BlockchainInterface interface {
	// Create a blockchain initializer.
	// This initializer receives the given blockchain parameters `params`
	// (from the setup file), the environment parameters `env` from the
	// Diablo primary command line and the set of blockchain `endpoints`
	// along with their tags.
	//
	Builder(params map[string]string, env []string, endpoints map[string][]string, logger Logger) (BlockchainBuilder, error)

	// Create a client for the given `view` of this blockchain.
	// A `view` is a list of addresses indicating how to contact the
	// blockchain endpoints (i.e. the nodes).
	// These addresses are among the ones specified in the setup
	// configuration file and the address format is used specified.
	// This client receives the given blockchain parameters `params`
	// (from the setup file) and the environment parameters `env` from the
	// Diablo secondary command line.
	//
	Client(params map[string]string, env, view []string, logger Logger) (BlockchainClient, error)
}

type BlockchainBuilder interface {
	CreateAccount(stake int) (interface{}, error)

	CreateContract(name string) (interface{}, error)

	CreateResource(domain string) (SampleFactory, bool)

	//
	// Interactions implemented by the blockchain.
	// If the blockchain does not implement a specific interaction, the
	// associated encoding method returns an error.
	//

	// Encode a transfer interaction.
	// A transfer moves a fungible amount of currencies `stake` from an
	// account `from` to an account `to`.
	//
	EncodeTransfer(amount int, from, to interface{}, info InteractionInfo) ([]byte, error)

	EncodeInvoke(from interface{}, contract interface{}, function string, info InteractionInfo) ([]byte, error)

	EncodeInteraction(itype string, expr BenchmarkExpression, info InteractionInfo) ([]byte, error)
}

type BlockchainClient interface {
	DecodePayload(bytes []byte) (interface{}, error)

	TriggerInteraction(iact Interaction) error
}



type accountFactory struct {
	builder  BlockchainBuilder
}

func newAccountFactory(builder BlockchainBuilder) *accountFactory {
	return &accountFactory{
		builder: builder,
	}
}

func (this *accountFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	var field BenchmarkExpression
	var elements []interface{}
	var i, number, istake int
	var stake IntVariable
	var err error
	var ok bool

	field, err = expr.TryField("number")
	if err == nil {
		number, err = field.GetInt()
		if err != nil {
			return nil, err
		}
	} else {
		number = 1
	}

	stake, err = expr.Field("stake").Int()
	if err != nil {
		return nil, err
	}

	elements = make([]interface{}, number)
	for i = range elements {
		istake, ok = stake.TryGetInt()
		if !ok {
			return nil, fmt.Errorf("%s: variable exhausted",
				expr.Field("stake").FullPosition())
		}

		elements[i], err = this.builder.CreateAccount(istake)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to create " +
				"sample: %s", expr.FullPosition(), err.Error())
		}
	}

	return newElementSample(elements), nil
}


type contractFactory struct {
	builder  BlockchainBuilder
}

func newContractFactory(builder BlockchainBuilder) *contractFactory {
	return &contractFactory{
		builder: builder,
	}
}

func (this *contractFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	var field BenchmarkExpression
	var elements []interface{}
	var name StringVariable
	var i, number int
	var iname string
	var err error
	var ok bool

	field, err = expr.TryField("number")
	if err == nil {
		number, err = field.GetInt()
		if err != nil {
			return nil, err
		}
	} else {
		number = 1
	}

	name, err = expr.Field("name").String()
	if err != nil {
		return nil, err
	}

	elements = make([]interface{}, number)
	for i = range elements {
		iname, ok = name.TryGetString()
		if !ok {
			return nil, fmt.Errorf("%s: variable exhausted",
				expr.Field("name").FullPosition())
		}

		elements[i], err = this.builder.CreateContract(iname)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to create " +
				"sample: %s", expr.FullPosition(), err.Error())
		}
	}

	return newElementSample(elements), nil
}


type transferInteractionFactory struct {
	builder  BlockchainBuilder
}

func newTransferInteractionFactory(builder BlockchainBuilder) *transferInteractionFactory {
	return &transferInteractionFactory{
		builder: builder,
	}
}

func (this *transferInteractionFactory) Instance(expr BenchmarkExpression, info InteractionInfo) ([]byte, error) {
        var field BenchmarkExpression
	var from, to interface{}
	var stake int
	var err error

	field, err = expr.TryField("stake")
	if err == nil {
		stake, err = field.GetInt()
		if err != nil {
			return nil, err
		}
	} else {
		stake = 1
	}

	from, err = expr.Field("from").GetResource("account")
	if err != nil {
		return nil, err
	}

	to, err = expr.Field("to").GetResource("account")
	if err != nil {
		return nil, err
	}

	return this.builder.EncodeTransfer(stake, from, to, info)
}


type invokeInteractionFactory struct {
	builder  BlockchainBuilder
}

func newInvokeInteractionFactory(builder BlockchainBuilder) *invokeInteractionFactory {
	return &invokeInteractionFactory{
		builder: builder,
	}
}

func (this *invokeInteractionFactory) Instance(expr BenchmarkExpression, info InteractionInfo) ([]byte, error) {
	var from, contract interface{}
	var function string
	var err error

	from, err = expr.Field("from").GetResource("account")
	if err != nil {
		return nil, err
	}

	contract, err = expr.Field("contract").GetResource("contract")
	if err != nil {
		return nil, err
	}

	function, err = expr.Field("function").GetString()
	if err != nil {
		return nil, err
	}

	return this.builder.EncodeInvoke(from, contract, function, info)
}


type proxyInteractionFactory struct {
	builder  BlockchainBuilder
	itype    string
}

func newProxyInteractionFactory(builder BlockchainBuilder, itype string) *proxyInteractionFactory {
	return &proxyInteractionFactory{
		builder: builder,
		itype: itype,
	}
}

func (this *proxyInteractionFactory) Instance(expr BenchmarkExpression, info InteractionInfo) ([]byte, error) {
	return this.builder.EncodeInteraction(this.itype, expr, info)
}
