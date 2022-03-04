package core


import (
	"math/rand"
)




type SampleFactory interface {
	Instance(expr BenchmarkExpression) (Sample, error)
}

type InteractionFactory interface {
	Instance(expr BenchmarkExpression, info InteractionInfo) ([]byte,error)
}

type randomFactory interface {
	instance(expr BenchmarkExpression) (Random, error)
}


type system struct {
	seedGenerator  *rand.Rand
	builder        BlockchainBuilder
	samples        map[string]SampleFactory
	randoms        map[string]randomFactory
	interactions   map[string]InteractionFactory
}

func newSystem(masterSeed int64, locations []location, setup setup, builder BlockchainBuilder) *system {
	var this system

	this.seedGenerator = rand.New(rand.NewSource(masterSeed))
	this.builder = builder

	this.samples = map[string]SampleFactory{
		"account": newAccountFactory(builder),
		"contract": newContractFactory(builder),
		"endpoint": newEndpointSampleFactory(setup),
		"float": newFloatSampleFactory(),
		"integer": newIntSampleFactory(),
		"location": newLocationSampleFactory(locations),
	}

	this.randoms = map[string]randomFactory{
		"uniform": newUniformRandomFactory(),
		"normal": newNormalRandomFactory(),
	}

	this.interactions = map[string]InteractionFactory{
		"transfer": newTransferInteractionFactory(builder),
		"invoke": newInvokeInteractionFactory(builder),
	}

	return &this
}

func (this *system) seed() int64 {
	return this.seedGenerator.Int63()
}

func (this *system) sampleFactory(domain string) (SampleFactory, bool) {
	var ret SampleFactory
	var ok bool

	ret, ok = this.samples[domain]
	if ok {
		return ret, ok
	}

	return this.builder.CreateResource(domain)
}

func (this *system) randomFactory(rtype string) (randomFactory, bool) {
	var ret randomFactory
	var ok bool

	ret, ok = this.randoms[rtype]

	return ret, ok
}

func (this *system) interactionFactory(itype string) (InteractionFactory, bool) {
	var ret InteractionFactory
	var ok bool

	ret, ok = this.interactions[itype]
	if ok {
		return ret, ok
	}

	return newProxyInteractionFactory(this.builder, itype), true
}
