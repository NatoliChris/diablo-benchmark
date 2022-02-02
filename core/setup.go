package core


import (
	"gopkg.in/yaml.v3"
	"os"
)


// Parsed setup file.
// Describe what is the tested system and how it is deployed.
//
type setup interface {
	// Return the name of the tested system.
	//
	sysname() string

	// Return the blockchain client parameters.
	//
	parameters() map[string]string

	// Return the set of the endpoints of the tested system.
	//
	endpoints() []endpoint
}

// An access point to the tested system.
// Describe an endpoint to connect to in order to communicate with the tested
// system. This is typically the TCP/IP address of a blockchain node.
//
type endpoint interface {
	// Return the address of this endpoint.
	// The returned address is as specified by the user in the setup
	// configuration file.
	//
	address() string

	// Return a list of tags associated with this endpoint.
	//
	tags() []string
}


type setupConfig struct {
	Sysname     string              `yaml:"interface"`
	Parameters  map[string]string   `yaml:"parameters"`
	Endpoints   []setupGroupConfig  `yaml:"endpoints"`
}

type setupGroupConfig struct {
	Addresses  []string            `yaml:"addresses"`
	Tags       []string            `yaml:"tags"`
}

func parseSetupYamlPath(path string) (setup, error) {
	var decoder *yaml.Decoder
	var config setupConfig
	var file *os.File
	var err error

	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	decoder = yaml.NewDecoder(file)
	err = decoder.Decode(&config)

	file.Close()

	if err != nil {
		return nil, err
	}

	return buildParsedSetup(&config), nil
}

func buildParsedSetup(config *setupConfig) *parsedSetup {
	var eps []endpoint
	var addr string
	var i, nep int

	nep = 0
	for i = range config.Endpoints {
		nep += len(config.Endpoints[i].Addresses)
	}

	eps = make([]endpoint, 0, nep)
	for i = range config.Endpoints {
		for _, addr = range config.Endpoints[i].Addresses {
			eps = append(eps, newParsedEndpoint(addr,
				config.Endpoints[i].Tags))
		}
	}

	return newParsedSetup(config.Sysname, config.Parameters, eps)
}


type parsedSetup struct {
	_sysname     string
	_parameters  map[string]string
	_endpoints   []endpoint
}

func newParsedSetup(sysname string, parameters map[string]string, endpoints []endpoint) *parsedSetup {
	return &parsedSetup{
		_sysname: sysname,
		_parameters: parameters,
		_endpoints: endpoints,
	}
}

func (this *parsedSetup) sysname() string {
	return this._sysname
}

func (this *parsedSetup) parameters() map[string]string {
	return this._parameters
}

func (this *parsedSetup) endpoints() []endpoint {
	return this._endpoints
}


type parsedEndpoint struct {
	_address  string
	_tags     []string
}

func newParsedEndpoint(address string, tags []string) *parsedEndpoint {
	return &parsedEndpoint{
		_address: address,
		_tags: tags,
	}
}

func (this *parsedEndpoint) address() string {
	return this._address
}

func (this *parsedEndpoint) tags() []string {
	return this._tags
}


type endpointSample struct {
	elements  []endpoint
}

func newEndpointSample(elements []endpoint) Sample {
	return &endpointSample{
		elements: elements,
	}
}

func (this *endpointSample) Size() int {
	return len(this.elements)
}

func (this *endpointSample) Get(index int) interface{} {
	return this.elements[index]
}


type endpointSampleFactory struct {
	elements  []taggedElement
}

func newEndpointSampleFactory(setup setup) SampleFactory {
	var elements []taggedElement
	var endpoint endpoint
	var i int

	elements = make([]taggedElement, len(setup.endpoints()))

	for i, endpoint = range setup.endpoints() {
		elements[i] = endpoint
	}

	return &connLocationSampleFactory{
		elements: elements,
	}
}

func (this *endpointSampleFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	return parseFilteredElementSample(expr, this.elements)
}
