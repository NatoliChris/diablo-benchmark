package core


import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
)


type benchmarkContext interface {
	// Return the core providing used supplied parsing and instanciation.
	//
	system() *system

	// Return a string description of the benchmark configuration.
	// The source of an expression is the same as the source of its
	// subexpressions.
	//
	Source() string

	// Return the local scope of this context.
	// The local scope can be `nil` if this context has not been
	// specialized.
	//
	local() scope

	// Return the current scope of this context.
	// If a local scope is defined, then return it, otherwise, return the
	// scope of the parent context.
	//
	current() scope

	// Set the local scope of this context.
	//
	specialize(scope)
}

// An expression in a benchmark configuration.
//
type BenchmarkExpression interface {
	// An expression is always associated with a context.
	//
	benchmarkContext

	// Return the position in the benchmark configuration string where this
	// expression starts.
	//
	Position() string

	// Return the concatenation of the `Source()` and `Position()` return
	// values.
	//
	FullPosition() string

	// Return the used defined type of this expression or an error if there
	// is no used defined type.
	//
	etype() (string, error)

	// Return the used defined name of this expression or an error if there
	// is no used defined name.
	//
	name() (string, error)

	// Return the targeted used defined name if this expression is a
	// reference to a previously used defined name or an error otherwise.
	//
	target() (string, error)

	// Return a Benchmark expression being the named field of this
	// expression assumed as a mapping.
	// If this expression is not a mapping or if it has no field for the
	// given name, return an expression that yields an error whenever
	// possible.
	//
	Field(string) BenchmarkExpression

	// If this expression is a mapping having a field with the given name,
	// then return this field as a new expression.
	// Otherwise return an error.
	//
	TryField(string) (BenchmarkExpression, error)

	Map() []BenchmarkExpression

	TryMap() ([]BenchmarkExpression, error)

	Key() BenchmarkExpression

	Value() BenchmarkExpression

	// Return a slice of expression contained in this expression assumed to
	// be a sequence.
	// If this expression is not a sequence, return a slice of exactly one
	// expression that yields an error whenever possible.
	//
	Slice() []BenchmarkExpression

	// If this expression is a sequence, then return a slice of the
	// contained expressions.
	// Otherwise return an error.
	//
	TrySlice() ([]BenchmarkExpression, error)

	// Check that every subexpression of this expression have been
	// explored (i.e. returned by a method invocation).
	// This method goes recursively.
	// If at leat one subexpression has been unexplored, return an error.
	// 
	Finish() error

	// Parse this expression assuming it is a scope expression with the
	// current scope as parent.
	// If this expression is not a valid scope then return an error.
	//
	scope() (scope, error)

	// Parse this expression assuming it is the description of a resource
	// of the given domain in the current scope.
	// If not then return an error.
	//
	Resource(string) (Variable, error)
	GetResource(string) (interface{}, error)

	// Parse this expression assuming it is the description of an integer
	// in the current scope.
	// If not then return an error.
	//
	Int() (IntVariable, error)
	GetInt() (int, error)

	Float() (FloatVariable, error)
	GetFloat() (float64, error)

	String() (StringVariable, error)
	GetString() (string, error)
}


func parseBenchmarkYamlPath(path string, sys *system) error {
	var context parsingContext
	var decoder *yaml.Decoder
	var implicit basicScope
	var bench benchmark
	var file *os.File
	var err error

	file, err = os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	decoder = yaml.NewDecoder(file)

	implicit.init(nil)
	context.init(path, sys, &implicit)
	bench.init(&context)

	err = decoder.Decode(&bench)
	if err != nil {
		return err
	}

	return nil
}


type parsingContext struct {
	path  string
	sys   *system
	top   scope
}

func (this *parsingContext) init(path string, sys *system, top scope) {
	this.path = path
	this.sys = sys
	this.top = top
}

func (this *parsingContext) system() *system {
	return this.sys
}

func (this *parsingContext) Source() string {
	return this.path
}

func (this *parsingContext) local() scope {
	return this.top
}

func (this *parsingContext) current() scope {
	return this.top
}

func (this *parsingContext) specialize(top scope) {
	this.top = top
}


type benchmark struct {
	context    benchmarkContext
	workloads  []interactionGenerator
}

func (this *benchmark) init(context benchmarkContext) {
	this.context = context
}

func (this *benchmark) UnmarshalYAML(node *yaml.Node) error {
	var expr BenchmarkExpression
	var err error

	expr, err = parseBenchmarkYaml(this.context, node)
	if err != nil {
		return err
	}
	
	return this.parse(expr)
}

func (this *benchmark) parse(expr BenchmarkExpression) error {
	var fields []BenchmarkExpression
	var field BenchmarkExpression
	var global scope
	var err error
	var i int

	field, err = expr.TryField("let")
	if err == nil {
		Tracef("parse benchmark global scope: %s", field.Position())

		global, err = field.scope()
		if err != nil {
			return err
		}

		expr.specialize(global)
	}

	fields = expr.Field("workloads").Slice()

	this.workloads = make([]interactionGenerator, len(fields))

	for i, field = range fields {
		this.workloads[i], err = parseWorkload(field)
		if err != nil {
			return err
		}
	}

	return this.instantiate()
}

func (this *benchmark) generate() <-chan *benchmarkInteraction {
	var out chan *benchmarkInteraction = make(chan *benchmarkInteraction)
	var ins []<-chan *benchmarkInteraction
	var load interactionGenerator
	var i int

	ins = make([]<-chan *benchmarkInteraction, len(this.workloads))

	for i, load = range this.workloads {
		ins[i] = load.generate()
	}

	go fuseGenerators(out, ins)

	return out
}

func (this *benchmark) instantiate() error {
	var iact *benchmarkInteraction
	var globalIndex int
	var encoded []byte
	var err error

	globalIndex = 0

	for iact = range this.generate() {
		iact.source.specialize(iact.current)

		encoded, err = iact.factory.Instance(iact.source,
			&interactionInformation{
				globalIndex: globalIndex,
				scheduleTime: iact.scheduleTime,
			})

		iact.source.specialize(nil)

		if err != nil {
			return err
		}

		err = iact.sendingClient.sendInteraction(
			iact.source.FullPosition(), iact.scheduleTime, encoded)
		if err != nil {
			return err
		}

		globalIndex += 1
	}

	return nil
}


type benchmarkInteraction struct {
	scheduleTime   float64
	sendingClient  client
	factory        InteractionFactory
	source         BenchmarkExpression
	current        scope
}

type interactionInformation struct {
	globalIndex   int
	clientIndex   int
	localIndex    int
	scheduleTime  float64
}

func (this *interactionInformation) TotalOrder() int {
	return this.globalIndex
}

func (this *interactionInformation) ClientOrder() int {
	return this.localIndex
}

func (this *interactionInformation) Timestamp() float64 {
	return this.scheduleTime
}

func (this *interactionInformation) ClientId() int {
	return this.clientIndex
}

type interactionGenerator interface {
	generate() <-chan *benchmarkInteraction
}

func fuseGenerators(out chan<- *benchmarkInteraction, ins []<-chan *benchmarkInteraction) {
	var elements []*benchmarkInteraction
	var element *benchmarkInteraction
	var i, min int

	elements = make([]*benchmarkInteraction, len(ins))

	for i = range ins {
		elements[i], _ = <- ins[i]
	}

	for {
		min = 0

		for i, element = range elements {
			if element == nil {
				continue
			}

			if elements[min] == nil {
				min = i
				continue
			}

			if element.scheduleTime < elements[min].scheduleTime {
				min = i
				continue
			}
		}

		if elements[min] == nil {
			break
		}

		out <- elements[min]

		elements[min] = <- ins[min]
	}

	close(out)
}



type workloadGenerator struct {
	clientLoads  []interactionGenerator
}

func parseWorkload(expr BenchmarkExpression) (*workloadGenerator, error) {
	var field BenchmarkExpression
	var this workloadGenerator
	var i, number int
	var local scope
	var err error

	Tracef("parse benchmark workload: %s", expr.Position())

	field, err = expr.TryField("let")
	if err == nil {
		local, err = field.scope()
		if err != nil {
			return nil, err
		}

		expr.specialize(local)
	}

	field, err = expr.TryField("number")
	if err == nil {
		number, err = field.GetInt()
		if err != nil {
			return nil, err
		}
	} else {
		number = 1
	}

	this.clientLoads = make([]interactionGenerator, number)

	for i = 0; i < number; i++ {
		this.clientLoads[i], err = parseClient(expr.Field("client"))
		if err != nil {
			return nil, err
		}
	}

	return &this, nil
}

func (this *workloadGenerator) generate() <-chan *benchmarkInteraction {
	var out chan *benchmarkInteraction = make(chan *benchmarkInteraction)
	var ins []<-chan *benchmarkInteraction
	var load interactionGenerator
	var i int

	ins = make([]<-chan *benchmarkInteraction, len(this.clientLoads))

	for i, load = range this.clientLoads {
		ins[i] = load.generate()
	}

	go fuseGenerators(out, ins)

	return out
}


type clientLoadGenerator struct {
	target  client
	loads   []interactionGenerator
}

func parseClient(expr BenchmarkExpression) (*clientLoadGenerator, error) {
	var behaviors []BenchmarkExpression
	var field BenchmarkExpression
	var this clientLoadGenerator
	var element interface{}
	var view []string
	var loc location
	var local scope
	var err error
	var i int

	field, err = expr.TryField("let")
	if err == nil {
		local, err = field.scope()
		if err != nil {
			return nil, err
		}

		expr.specialize(local)
		defer expr.specialize(nil)
	}

	view, err = parseView(expr)
	if err != nil {
		return nil, err
	}

	element, err = expr.Field("location").GetResource("location")
	if err != nil {
		return nil, err
	}

	loc = element.(location)

	this.target, err = loc.createClient(expr.FullPosition(), view)
	if err != nil {
		return nil, err
	}

	behaviors = expr.Field("behavior").Slice()

	this.loads = make([]interactionGenerator, len(behaviors))

	for i, field = range behaviors {
		this.loads[i], err = parseLoad(field, this.target)
		if err != nil {
			return nil, err
		}

		
	}

	return &this, nil
}

func parseView(client BenchmarkExpression) ([]string, error) {
 	var vfield, sfield BenchmarkExpression
	var addrs []string = make([]string, 0)
	var element interface{}
	var view Variable
	var seed64 int64
	var err error
	var seed int

	vfield = client.Field("view")
	view, err = vfield.Resource("endpoint")
	if err == nil {
		seed64 = client.system().seed()
	} else {
		view, err = vfield.Field("endpoint").Resource("endpoint")
		if err != nil {
			return nil, err
		}

		sfield, err = vfield.TryField("seed")
		if err == nil {
			fmt.Printf("explicit seed\n")
			seed, err = sfield.GetInt()
			if err != nil {
				return nil, err
			}

			seed64 = int64(seed)
		} else {
			seed64 = client.system().seed()
		}
	}

	view = copyVariable(view, seed64, TypeOnce)

	for {
		element = view.Get()
		if element == nil {
			break
		}

		addrs = append(addrs, element.(endpoint).address())
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("%s: variable exhausted",
			vfield.FullPosition())
	}

	return addrs, nil
}

func (this *clientLoadGenerator) generate() <-chan *benchmarkInteraction {
	var out chan *benchmarkInteraction = make(chan *benchmarkInteraction)
	var ins []<-chan *benchmarkInteraction
	var load interactionGenerator
	var i int

	ins = make([]<-chan *benchmarkInteraction, len(this.loads))

	for i, load = range this.loads {
		ins[i] = load.generate()
	}

	go fuseGenerators(out, ins)

	return out
}


func parseLoad(expr BenchmarkExpression, sender client) (interactionGenerator, error) {
	var btype string
	var err error

	btype, err = expr.etype()
	if err != nil {
		btype = "timeload"
	}

	if btype == "timeload" {
		return parseTimeload(expr, sender)
	}

	return nil, fmt.Errorf("%s: unknown behavior '%s'",
		expr.FullPosition(), btype)
}


type timeloadGenerator struct {
	load     map[float64]float64
	sender   client
	factory  InteractionFactory
	source   BenchmarkExpression
	current  scope
}

func parseTimeload(expr BenchmarkExpression, sender client) (*timeloadGenerator, error) {
	var load, interaction BenchmarkExpression
	var this timeloadGenerator
	var itype string
	var time float64
	var err error
	var ok bool

	this.sender = sender

	interaction = expr.Field("interaction")
	itype, err = interaction.etype()
	if err != nil {
		return nil, err
	}

	this.source = interaction
	this.current = interaction.current()

	this.factory, ok = expr.system().interactionFactory(itype)
	if !ok {
		return nil, fmt.Errorf("%s: unknown interaction type '%s'",
			interaction.FullPosition(), itype)
	}

	this.load = make(map[float64]float64)
	for _, load = range expr.Field("load").Map() {
		time, err = load.Key().GetFloat()
		if err != nil {
			return nil, err
		}

		if time < 0 {
			return nil, fmt.Errorf("%s: must be positive or zero",
				load.Key().FullPosition())
		}

		this.load[time], err = load.Value().GetFloat()
		if err != nil {
			return nil, err
		}
	}

	return &this, nil
}

func (this *timeloadGenerator) generate() <-chan *benchmarkInteraction {
	var ret chan *benchmarkInteraction = make(chan *benchmarkInteraction)

	go this.flatten(ret)

	return ret
}

func (this *timeloadGenerator) flatten(out chan<- *benchmarkInteraction) {
	var key, value, tick, done, clock, wait float64
	var times []float64
	var i int

	times = make([]float64, 0, len(this.load))

	for key = range this.load {
		times = append(times, key)
	}

	sort.Float64s(times)

	key = 0
	value = 0
	done = 0
	for i = range times {
		if (times[i] != key) && (value != 0) {
			tick = 1 / value
			clock = key

			for {
				wait = (1 - done) * tick

				if (clock + wait) <= times[i] {
					done = 0
					clock += wait
					out <- &benchmarkInteraction{
						scheduleTime: clock,
						sendingClient: this.sender,
						factory: this.factory,
						source: this.source,
						current: this.current,
					}
				} else {
					done += (times[i] - clock) / tick
					clock = times[i]
					break
				}
			}
		}

		key = times[i]
		value = this.load[key]
	}

	close(out)
}


type errorExpression struct {
	parent    benchmarkContext
	position  string
	err       error
}

func newErrorExpression(parent benchmarkContext, position string, err error) BenchmarkExpression {
	var this errorExpression

	this.parent = parent
	this.position = position
	this.err = err

	return &this
}

func (this *errorExpression) system() *system {
	return this.parent.system()
}

func (this *errorExpression) Source() string {
	return this.parent.Source()
}

func (this *errorExpression) local() scope {
	return nil
}

func (this *errorExpression) current() scope {
	return this.parent.current()
}

func (this *errorExpression) specialize(scope) {
}

func (this *errorExpression) Position() string {
	return this.position
}

func (this *errorExpression) FullPosition() string {
	return fmt.Sprintf("%s:%s", this.Source(), this.Position())
}

func (this *errorExpression) etype() (string, error) {
	return "", this.err
}

func (this *errorExpression) name() (string, error) {
	return "", this.err
}

func (this *errorExpression) target() (string, error) {
	return "", this.err
}

func (this *errorExpression) Field(string) BenchmarkExpression {
	return this
}

func (this *errorExpression) TryField(string) (BenchmarkExpression, error) {
	return nil, this.err
}

func (this *errorExpression) Map() []BenchmarkExpression {
	var ret []BenchmarkExpression = make([]BenchmarkExpression, 1)

	ret[0] = this

	return ret
}

func (this *errorExpression) TryMap() ([]BenchmarkExpression, error) {
	return nil, this.err
}

func (this *errorExpression) Key() BenchmarkExpression {
	return this
}

func (this *errorExpression) Value() BenchmarkExpression {
	return this
}

func (this *errorExpression) Slice() []BenchmarkExpression {
	var ret []BenchmarkExpression = make([]BenchmarkExpression, 1)

	ret[0] = this

	return ret
}

func (this *errorExpression) TrySlice() ([]BenchmarkExpression, error) {
	return nil, this.err
}

func (this *errorExpression) Finish() error {
	return this.err
}

func (this *errorExpression) scope() (scope, error) {
	return nil, this.err
}

func (this *errorExpression) Resource(string) (Variable, error) {
	return nil, this.err
}

func (this *errorExpression) GetResource(string) (interface{}, error) {
	return nil, this.err
}

func (this *errorExpression) Int() (IntVariable, error) {
	return nil, this.err
}

func (this *errorExpression) GetInt() (int, error) {
	return 0, this.err
}

func (this *errorExpression) Float() (FloatVariable, error) {
	return nil, this.err
}

func (this *errorExpression) GetFloat() (float64, error) {
	return 0, this.err
}

func (this *errorExpression) String() (StringVariable, error) {
	return nil, this.err
}

func (this *errorExpression) GetString() (string, error) {
	return "", this.err
}


func parseBenchmarkYaml(context benchmarkContext, node *yaml.Node) (BenchmarkExpression, error) {
	if node.Kind == yaml.MappingNode {
		return parseBenchmarkYamlMapping(context, node)
	}

	if node.Kind == yaml.SequenceNode {
		return parseBenchmarkYamlSequence(context, node)
	}

	if node.Kind == yaml.ScalarNode {
		return parseBenchmarkYamlScalar(context, node)
	}

	if node.Kind == yaml.AliasNode {
		return parseBenchmarkYamlAlias(context, node)
	}

	panic(fmt.Errorf("not yet implemented (yaml kind %v at %s:%d:%d)",
		node.Kind, context.Source(), node.Line, node.Column))
}


type benchmarkYamlNode struct {
	parent  benchmarkContext
	node    *yaml.Node
	slocal  scope
	texpl   bool       // etype() has been called
	nexpl   bool       // name() has been called
}

func (this *benchmarkYamlNode) init(parent benchmarkContext, node *yaml.Node) {
	this.parent = parent
	this.node = node
	this.slocal = nil
	this.texpl = false
	this.nexpl = false
}

func (this *benchmarkYamlNode) system() *system {
	return this.parent.system()
}

func (this *benchmarkYamlNode) Source() string {
	return this.parent.Source()
}

func (this *benchmarkYamlNode) local() scope {
	return this.slocal
}

func (this *benchmarkYamlNode) current() scope {
	if this.slocal != nil {
		return this.slocal
	} else {
		return this.parent.current()
	}
}

func (this *benchmarkYamlNode) specialize(slocal scope) {
	this.slocal = slocal
}

func (this *benchmarkYamlNode) Position() string {
	return fmt.Sprintf("%d:%d", this.node.Line, this.node.Column)
}

func (this *benchmarkYamlNode) FullPosition() string {
	return fmt.Sprintf("%s:%s", this.Source(), this.Position())
}

func (this *benchmarkYamlNode) etype() (string, error) {
	this.texpl = true

	if (this.node.Style & yaml.TaggedStyle) == 0 {
		return "", fmt.Errorf("%s:%s: no specified type",
			this.FullPosition())
	}

	return this.node.Tag[1:], nil
}

func (this *benchmarkYamlNode) name() (string, error) {
	this.nexpl = true

	if this.node.Anchor == "" {
		return "", fmt.Errorf("%s: no specified name",
			this.FullPosition())
	}

	return this.node.Anchor, nil
}

func (this *benchmarkYamlNode) target() (string, error) {
	return "", fmt.Errorf("%s: must be a variable name",
		this.FullPosition())
}

func (this *benchmarkYamlNode) Field(name string) BenchmarkExpression {
	var err error

	_, err = this.TryField(name)

	return newErrorExpression(this, this.Position(), err)
}

func (this *benchmarkYamlNode) TryField(string) (BenchmarkExpression, error) {
	return nil, fmt.Errorf("%s: must be a mapping", this.FullPosition())
}

func (this *benchmarkYamlNode) Fields() (map[string]BenchmarkExpression, error) {
	return nil, fmt.Errorf("%s: must be a mapping", this.FullPosition())
}

func (this *benchmarkYamlNode) Map() []BenchmarkExpression {
	var ret []BenchmarkExpression = make([]BenchmarkExpression, 1)
	var err error

	_, err = this.TryMap()
	ret[0] = newErrorExpression(this, this.Position(), err)

	return ret
}

func (this *benchmarkYamlNode) TryMap() ([]BenchmarkExpression, error) {
	return nil, fmt.Errorf("%s: must be a mapping", this.FullPosition())
}

func (this *benchmarkYamlNode) Key() BenchmarkExpression {
	return newErrorExpression(this, this.Position(),
		fmt.Errorf("%s: must be a mapping field", this.FullPosition()))
}

func (this *benchmarkYamlNode) Value() BenchmarkExpression {
	return newErrorExpression(this, this.Position(),
		fmt.Errorf("%s: must be a mapping field", this.FullPosition()))
}

func (this *benchmarkYamlNode) Slice() []BenchmarkExpression {
	var ret []BenchmarkExpression = make([]BenchmarkExpression, 1)
	var err error

	_, err = this.TrySlice()
	ret[0] = newErrorExpression(this, this.Position(), err)

	return ret
}

func (this *benchmarkYamlNode) TrySlice() ([]BenchmarkExpression, error) {
	return nil, fmt.Errorf("%s: must be a sequence", this.FullPosition())
}

func (this *benchmarkYamlNode) scope() (scope, error) {
	return nil, fmt.Errorf("%s: must be a scope", this.FullPosition())
}

func (this *benchmarkYamlNode) Resource(domain string) (Variable, error) {
	return nil, fmt.Errorf("%s: must be a resource of domain '%s'",
		this.FullPosition(), domain)
}

func (this *benchmarkYamlNode) GetResource(domain string) (interface{}, error) {
	return nil, fmt.Errorf("%s: must be a resource of domain '%s'",
		this.FullPosition(), domain)
}

func (this *benchmarkYamlNode) Int() (IntVariable, error) {
	return nil, fmt.Errorf("%s: must be an int", this.FullPosition())
}

func (this *benchmarkYamlNode) GetInt() (int, error) {
	return 0, fmt.Errorf("%s: must be an int", this.FullPosition())
}

func (this *benchmarkYamlNode) Float() (FloatVariable, error) {
	return nil, fmt.Errorf("%s: must be a float", this.FullPosition())
}

func (this *benchmarkYamlNode) GetFloat() (float64, error) {
	return 0, fmt.Errorf("%s: must be a float", this.FullPosition())
}

func (this *benchmarkYamlNode) String() (StringVariable, error) {
	return nil, fmt.Errorf("%s: must be a string", this.FullPosition())
}

func (this *benchmarkYamlNode) GetString() (string, error) {
	return "", fmt.Errorf("%s: must be a string", this.FullPosition())
}


type benchmarkYamlField struct {
	benchmarkYamlNode
	key    BenchmarkExpression
	value  BenchmarkExpression
}

func parseBenchmarkYamlField(context benchmarkContext, key, value *yaml.Node) (BenchmarkExpression, error) {
	var this benchmarkYamlField
	var err error

	this.init(context, key)

	this.key, err = parseBenchmarkYaml(&this, key)
	if err != nil {
		return nil, err
	}

	this.value, err = parseBenchmarkYaml(&this, value)
	if err != nil {
		return nil, err
	}

	return &this, nil
}

func (this *benchmarkYamlField) Key() BenchmarkExpression {
	return this.key
}

func (this *benchmarkYamlField) Value() BenchmarkExpression {
	return this.value
}

func (this *benchmarkYamlField) Finish() error {
	return nil
}


type benchmarkYamlMapping struct {
	benchmarkYamlNode
	index  map[string]int
	fields []BenchmarkExpression
}

func parseBenchmarkYamlMapping(context benchmarkContext, node *yaml.Node) (BenchmarkExpression, error) {
	var this benchmarkYamlMapping
	var key, value *yaml.Node
	var i, index int
	var found bool
	var err error

	this.init(context, node)
	this.index = make(map[string]int, len(node.Content) / 2)
	this.fields = make([]BenchmarkExpression, len(node.Content) / 2)

	for i = 0; i < len(this.fields); i++ {
		key = node.Content[i * 2]
		value = node.Content[i * 2 + 1]

		index, found = this.index[key.Value]
		if found {
			return nil, fmt.Errorf("%s:%d:%d: field defined " +
				"twice (previously at %s)", this.Source(),
				key.Line, key.Column,
				this.fields[index].Key().Position())
		}

		this.fields[i], err = parseBenchmarkYamlField(&this, key,value)
		if err != nil {
			return nil, err
		}

		this.index[key.Value] = i
	}

	return &this, nil
}

func (this *benchmarkYamlMapping) Field(name string) BenchmarkExpression {
	var ret BenchmarkExpression
	var err error

	ret, err = this.TryField(name)
	if err != nil {
		return newErrorExpression(this, this.Position(), err)
	}

	return ret
}

func (this *benchmarkYamlMapping) TryField(name string) (BenchmarkExpression, error) {
	var index int
	var ok bool

	index, ok = this.index[name]
	if !ok {
		return nil, fmt.Errorf("%s: missing '%s' field",
			this.FullPosition(), name)
	}

	return this.fields[index].Value(), nil
}

func (this *benchmarkYamlMapping) Map() []BenchmarkExpression {
	return this.fields
}

func (this *benchmarkYamlMapping) TryMap() ([]BenchmarkExpression, error) {
	return this.fields, nil
}

func (this *benchmarkYamlMapping) Finish() error {
	return nil
}


type benchmarkYamlSequence struct {
	benchmarkYamlNode
	items []BenchmarkExpression
}

func parseBenchmarkYamlSequence(context benchmarkContext, node *yaml.Node) (BenchmarkExpression, error) {
	var this benchmarkYamlSequence
	var child *yaml.Node
	var err error
	var i int

	this.init(context, node)
	this.items = make([]BenchmarkExpression, len(node.Content))

	for i, child = range node.Content {
		this.items[i], err = parseBenchmarkYaml(&this, child)
		if err != nil {
			return nil, err
		}
	}

	return &this, nil
}

func (this *benchmarkYamlSequence) Slice() []BenchmarkExpression {
	return this.items
}

func (this *benchmarkYamlSequence) TrySlice() ([]BenchmarkExpression, error) {
	return this.items, nil
}

func (this *benchmarkYamlSequence) Finish() error {
	return nil
}

func (this *benchmarkYamlSequence) scope() (scope, error) {
	var local basicScope
	var err error

	local.init(this.current())

	err = local.parse(this)
	if err != nil {
		return nil, err
	}

	return &local, nil
}


type benchmarkYamlScalar struct {
	benchmarkYamlNode
}

func parseBenchmarkYamlScalar(context benchmarkContext, node *yaml.Node) (BenchmarkExpression, error) {
	var this benchmarkYamlScalar

	this.init(context, node)

	return &this, nil
}

func (this *benchmarkYamlScalar) Finish() error {
	return nil
}

func (this *benchmarkYamlScalar) Int() (IntVariable, error) {
	var err error
	var val int

	val, err = this.GetInt()
	if err != nil {
		return nil, err
	}

	return newIntImmediate(val), nil
}

func (this *benchmarkYamlScalar) GetInt() (int, error) {
	var opaque interface{}
	var err error
	var val int
	var ok bool

	if (this.node.Style & yaml.DoubleQuotedStyle) != 0 {
		return 0, fmt.Errorf("%s: must be an int",
			this.FullPosition())
	}

	if (this.node.Style & yaml.SingleQuotedStyle) != 0 {
		return 0, fmt.Errorf("%s: must be an int",
			this.FullPosition())
	}

	err = this.node.Decode(&opaque)
	if err != nil {
		return 0, fmt.Errorf("%s: must be an int",
			this.FullPosition())
	}

	val, ok = opaque.(int)
	if !ok {
		return 0, fmt.Errorf("%s: must be an int",
			this.FullPosition())
	}

	return val, nil
}

func (this *benchmarkYamlScalar) Float() (FloatVariable, error) {
	var value float64
	var err error

	value, err = this.GetFloat()
	if err != nil {
		return nil, err
	}

	return newFloatImmediate(value), nil
}

func (this *benchmarkYamlScalar) GetFloat() (float64, error) {
	var opaque interface{}
	var fval float64
	var err error
	var ival int
	var ok bool

	if (this.node.Style & yaml.DoubleQuotedStyle) != 0 {
		return 0, fmt.Errorf("%s: must be a float",
			this.FullPosition())
	}

	if (this.node.Style & yaml.SingleQuotedStyle) != 0 {
		return 0, fmt.Errorf("%s: must be a float",
			this.FullPosition())
	}

	err = this.node.Decode(&opaque)
	if err != nil {
		return 0, fmt.Errorf("%s: must be a float",
			this.FullPosition())
	}

	fval, ok = opaque.(float64)
	if ok {
		return fval, nil
	}

	ival, ok = opaque.(int)
	if ok {
		return float64(ival), nil
	}

	return 0, fmt.Errorf("%s: must be a float", this.FullPosition())
}

func (this *benchmarkYamlScalar) String() (StringVariable, error) {
	var value string
	var err error

	value, err = this.GetString()
	if err != nil {
		return nil, err
	}

	return newStringImmediate(value), nil
}

func (this *benchmarkYamlScalar) GetString() (string, error) {
	if (this.node.Style & yaml.DoubleQuotedStyle) != 0 {
		return this.node.Value, nil
	}

	if (this.node.Style & yaml.SingleQuotedStyle) != 0 {
		return this.node.Value, nil
	}

	return "", fmt.Errorf("%s: must be a string", this.FullPosition())
}


type benchmarkYamlAlias struct {
	benchmarkYamlNode
}

func parseBenchmarkYamlAlias(context benchmarkContext, node *yaml.Node) (BenchmarkExpression, error) {
	var this benchmarkYamlAlias

	this.init(context, node)

	return &this, nil
}

func (this *benchmarkYamlAlias) target() (string, error) {
	return this.node.Value, nil
}

func (this *benchmarkYamlAlias) Finish() error {
	return nil
}

func (this *benchmarkYamlAlias) Resource(domain string) (Variable, error) {
	var name, vdomain string
	var variable Variable
	var err error
	var ok bool

	name, err = this.target()
	if err != nil {
		return nil, err
	}

	variable, vdomain, ok = this.current().get(name)
	if !ok {
		return nil, fmt.Errorf("%s: unknown variable '%s'",
			this.FullPosition(), name)
	}

	if vdomain != domain {
		return nil, fmt.Errorf("%s: cannot convert '%s' to '%s'",
			this.FullPosition(), vdomain, domain)
	}

	return variable, nil
}

func (this *benchmarkYamlAlias) GetResource(domain string) (interface{}, error) {
	var resource interface{}
	var variable Variable
	var err error

	variable, err = this.Resource(domain)
	if err != nil {
		return nil, err
	}

	resource = variable.Get()
	if resource == nil {
		return nil, fmt.Errorf("%s: variable exhausted",
			this.FullPosition())
	}

	return resource, nil
}

func (this *benchmarkYamlAlias) Int() (IntVariable, error) {
	var name, domain string
	var variable Variable
	var err error
	var ok bool

	name, err = this.target()
	if err != nil {
		return nil, err
	}

	variable, domain, ok = this.current().get(name)
	if !ok {
		return nil, fmt.Errorf("%s: unknown variable '%s'",
			this.FullPosition(), name)
	}

	if domain == "integer" {
		return newIntVariable(variable), nil
	}

	return nil, fmt.Errorf("%s: cannot convert '%s' to int",
		this.FullPosition(), domain)
}

func (this *benchmarkYamlAlias) GetInt() (int, error) {
	var v IntVariable
	var err error
	var ok bool
	var i int

	v, err = this.Int()
	if err != nil {
		return 0, err
	}

	i, ok = v.TryGetInt()
	if !ok {
		return 0, fmt.Errorf("%s: variable exhausted",
			this.FullPosition())
	}

	return i, nil
}

func (this *benchmarkYamlAlias) Float() (FloatVariable, error) {
	var name, domain string
	var variable Variable
	var err error
	var ok bool

	name, err = this.target()
	if err != nil {
		return nil, err
	}

	variable, domain, ok = this.current().get(name)
	if !ok {
		return nil, fmt.Errorf("%s: unknown variable '%s'",
			this.FullPosition(), name)
	}

	if domain == "float" {
		return newFloatVariable(variable), nil
	}

	if domain == "integer" {
		return newFloatVariable(variable), nil
	}

	return nil, fmt.Errorf("%s: cannot convert '%s' to float",
		this.FullPosition(), domain)
}

func (this *benchmarkYamlAlias) GetFloat() (float64, error) {
	var v FloatVariable
	var err error
	var f float64
	var ok bool

	v, err = this.Float()
	if err != nil {
		return 0, err
	}

	f, ok = v.TryGetFloat()
	if !ok {
		return 0, fmt.Errorf("%s: variable exhausted",
			this.FullPosition())
	}

	return f, nil
}

func (this *benchmarkYamlAlias) String() (StringVariable, error) {
	var name, domain string
	var variable Variable
	var err error
	var ok bool

	name, err = this.target()
	if err != nil {
		return nil, err
	}

	variable, domain, ok = this.current().get(name)
	if !ok {
		return nil, fmt.Errorf("%s: unknown variable '%s'",
			this.FullPosition(), name)
	}

	if domain == "string" {
		return newStringVariable(variable), nil
	}

	return nil, fmt.Errorf("%s: cannot convert '%s' to string",
		this.FullPosition(), domain)
}

func (this *benchmarkYamlAlias) GetString() (string, error) {
	var v StringVariable
	var err error
	var s string
	var ok bool

	v, err = this.String()
	if err != nil {
		return "", err
	}

	s, ok = v.TryGetString()
	if !ok {
		return "", fmt.Errorf("%s: variable exhausted",
			this.FullPosition())
	}

	return s, nil
}
