package core


import (
	"fmt"
	"math/rand"
)


type SampleFactory interface {
	Instance(BenchmarkExpression) (Sample, error)
}

type interactionFactory interface {
	Instance(BenchmarkExpression) ([]byte, error)
}

type randomFactory interface {
	Instance(BenchmarkExpression) (Random, error)
}


type Core struct {
	seedGenerator         *rand.Rand
	setup                 Setup
	sampleFactories       map[string]SampleFactory
	randomFactories       map[string]randomFactory
	interactionFactories  map[string]interactionFactory
	blockchainIface       BlockchainInterface
	clientNextId          int   // todo stuff
}

func New(masterSeed int64, setup Setup, biface BlockchainInterface) *Core {
	var endpointFactory, locationFactory nodeSampleFactory
	var ret Core

	ret.seedGenerator = rand.New(rand.NewSource(masterSeed))
	ret.setup = setup
	ret.blockchainIface = biface

	endpointFactory.init(setup.getEndpoints())
	locationFactory.init(setup.getLocations())

	ret.sampleFactories = make(map[string]SampleFactory, 0)
	ret.sampleFactories["balance"] = newBalanceFactory(biface)
	ret.sampleFactories["endpoint"] = &endpointFactory
	ret.sampleFactories["float"] = &floatSampleFactory{}
	ret.sampleFactories["integer"] = &intSampleFactory{}
	ret.sampleFactories["location"] = &locationFactory

	ret.randomFactories = make(map[string]randomFactory, 0)
	ret.randomFactories["uniform"] = &uniformRandomFactory{}

	ret.interactionFactories = make(map[string]interactionFactory, 0)
	ret.interactionFactories["transfer"] = &transferInteractionFactory{
		biface,
	}

	return &ret
}


type remoteClient interface {
	sendInteraction(float64, []byte) error
}

type todoClient struct {
	id      int                // TODO logging
	remote  BlockchainClient   // should be remote
}

func (this *Core) createClient(addr string, view []string) (remoteClient, error) {
	var cli todoClient
	var err error

	cli.id = this.clientNextId
	cli.remote, err = this.blockchainIface.Client(view)
	if err != nil {
		return nil, err
	}

	this.clientNextId += 1

	fmt.Printf("TODO: create new client %d on '%s' with view: %v\n",
		cli.id, addr, view)

	return &cli, nil
}

func (this *todoClient) sendInteraction(when float64, payload []byte) error {
	var err error

	fmt.Printf("TODO: send %d bytes to client %d to trigger at %f\n",
		len(payload), this.id, when)

	_, err = this.remote.DecodePayload(payload)

	return err
}


func (this *Core) seed() int64 {
	return this.seedGenerator.Int63()
}

func (this *Core) sampleFactory(domain string) (SampleFactory, bool) {
	var factory SampleFactory
	var ok bool

	factory, ok = this.sampleFactories[domain]

	return factory, ok
}

func (this *Core) interactionFactory(itype string) (interactionFactory, bool) {
	var factory interactionFactory
	var ok bool

	factory, ok = this.interactionFactories[itype]

	return factory, ok
}

func (this *Core) random(rtype string, expr BenchmarkExpression) (Random, error) {
	var factory randomFactory
	var ok bool

	factory, ok = this.randomFactories[rtype]
	if !ok {
		return nil, fmt.Errorf("unknown random type '%s'", rtype)
	}

	return factory.Instance(expr)
}


type balanceFactory struct {
	biface BlockchainInterface
}

func newBalanceFactory(biface BlockchainInterface) *balanceFactory {
	return &balanceFactory{
		biface,
	}
}

func (this *balanceFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	var elements []interface{}
	var number, stake int
	var err error
	var i int

	number, err = expr.Field("number").GetInt()
	if err != nil {
		return nil, err
	}

	stake, err = expr.Field("stake").GetInt()
	if err != nil {
		return nil, err
	}

	elements = make([]interface{}, number)
	for i = range elements {
		elements[i], err = this.biface.CreateBalance(stake)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to create " +
				"sample: %s", expr.FullPosition(), err.Error())
		}
	}

	return newElementSample(elements), nil
}


type transferInteractionFactory struct {
	biface BlockchainInterface
}

func (this *transferInteractionFactory) Instance(expr BenchmarkExpression) ([]byte, error) {
	var field BenchmarkExpression
	var from, to interface{}
	var local scope
	var stake int
	var err error

	field, err = expr.TryField("let")
	if err == nil {
		local, err = field.scope()
		if err != nil {
			return nil, err
		}

		expr.specialize(local)
		defer expr.specialize(nil)
	}

	field, err = expr.TryField("stake")
	if err == nil {
		stake, err = field.GetInt()
		if err != nil {
			return nil, err
		}
	} else {
		stake = 1
	}

	from, err = expr.Field("from").GetResource("balance")
	if err != nil {
		return nil, err
	}

	to, err = expr.Field("to").GetResource("balance")
	if err != nil {
		return nil, err
	}

	return this.biface.EncodeTransfer(stake, from, to)
}
