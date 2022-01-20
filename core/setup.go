package core


import (
	"fmt"
	"regexp"
	"gopkg.in/yaml.v3"
	"os"
)


type Setup interface {
	getEndpoints() []node
	getLocations() []node
}

type setup struct {
	endpoints []node
	locations []node
}

func (this *setup) getEndpoints() []node {
	return this.endpoints
}

func (this *setup) getLocations() []node {
	return this.locations
}


// A node of the setup.
// This can represent either a Diablo secondary node or a blockchain server
// node.
//
type node struct {
	// The address of this node.
	// This is user specified and thus is arbitrary.
	//
	address string

	// The tags associated with this node.
	// These are user specified arbitrary strings.
	//
	tags []string
}


func ParseSetupYamlPath(path string) (Setup, error) {
	var decoder *yaml.Decoder
	var syaml setupYaml
	var file *os.File
	var err error

	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	decoder = yaml.NewDecoder(file)

	err = decoder.Decode(&syaml)
	if err != nil {
		return nil, err
	}

	return syaml.makeSetup(), nil
}


type setupYaml struct {
	Endpoints   []setupGroupYaml  `yaml:"endpoints"`
	Locations   []setupGroupYaml  `yaml:"locations"`
}

func (this *setupYaml) makeSetup() *setup {
	var ret setup
	var i int

	ret.endpoints = make([]node, 0)
	for i = range this.Endpoints {
		ret.endpoints = append(ret.endpoints,
			this.Endpoints[i].makeNodes() ...)
	}

	ret.locations = make([]node, 0)
	for i = range this.Locations {
		ret.locations = append(ret.locations,
			this.Locations[i].makeNodes() ...)
	}

	return &ret
}


type setupGroupYaml struct {
	Addresses   []string          `yaml:"addresses"`
	Tags        []string          `yaml:"tags"`
}

func (this *setupGroupYaml) makeNodes() []node {
	var nodes []node = make([]node, len(this.Addresses))
	var i int

	for i = range this.Addresses {
		nodes[i].address = this.Addresses[i]
		nodes[i].tags = this.Tags
	}

	return nodes
}


type nodeSample struct {
	values []node
}

func (this *nodeSample) Size() int {
	return len(this.values)
}

func (this *nodeSample) Get(index int) interface{} {
	return this.GetNode(index)
}

func (this *nodeSample) GetNode(index int) *node {
	return &this.values[index]
}


type nodeSampleFactory struct {
	nodes []node
}

func (this *nodeSampleFactory) init(nodes []node) {
	this.nodes = nodes
}

func (this *nodeSampleFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	var field BenchmarkExpression
	var match, tag string
	var re *regexp.Regexp
	var filtered []node
	var i, size int
	var err error
	var pass bool

	filtered = make([]node, len(this.nodes))
	for i = range this.nodes {
		filtered[i] = this.nodes[i]
	}

	size = len(filtered)

	for _, field = range expr.Slice() {
		match, err = field.GetString()
		if err != nil {
			return nil, err
		}

		re, err = regexp.Compile(match)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid regexp: %s",
				field.FullPosition(), err.Error())
		}

		for i = 0; i < size;  {
			pass = false

			for _, tag = range filtered[i].tags {
				if re.MatchString(tag) {
					pass = true
					break
				}
			}

			if !pass {
				filtered[i] = filtered[size-1]
				size -= 1
			} else {
				i += 1
			}
		}
	}

	return &nodeSample{ filtered[0:size] }, nil
}
