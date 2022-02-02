package core


import (
	"fmt"
)


// The sample space of a random variable.
// All sample spaces are discrete sample space.
//
type Sample interface {
	// Return the number of element in the sample.
	// The return value is positive or zero.
	//
	Size() int

	// Return the element associated with the given index.
	// The given index must be lesser or equal to the return value of the
	// `Size()` method.
	//
	Get(int) interface{}
}

// The type of a random variable.
//
type VariableType uint8

const (
	// regular random variable: the same event can happen many times.
	TypeRegular       VariableType = iota

	// A once iterator: the same event never happens twice.
	TypeOnce

	// A loop iterator: all events happen before any event happens twice.
	TypeLoop
)

// The random space of a random variable.
// This is essentially a factory for `Distribution`.
//
type Random interface {
	// Create a new probability distribution for a sample space of the
	// given size with the given seed and type.
	// The given size must be positive or zero.
	//
	Instance(int, int64, VariableType)  Distribution
}

// The probability distribution of a random variable.
//
type Distribution interface {
	// Select an event in the probability space.
	// Return the selected event with no error if such a possible event
	// exists.
	// Otherwise return an error.
	//
	Select() (int, error)

	// Create a new distribution of the same type than this distribution.
	// The new distribution maps the a sample space of the same size as
	// this one.
	// The new distribution set its seed to the given one.
	// The new distribution has the specified type but keeps the events
	// discarded if any.
	//
	Copy(int64, VariableType) Distribution
}

// A random variable.
//
type Variable interface {
	// Return the string description of this variable domain.
	//
	Domain() string

	// Return the sample space of this variable.
	//
	Sample() Sample

	// Return the distribution of this variable.
	//
	Distribution() Distribution

	// Select an event in the sample space of this variable according to
	// its distribution.
	// If the sample space is empty or if the distribution cannot select
	// any event, return nil.
	//
	Get() interface{}
}


type VariableImpl struct {
	domain   string
	sample   Sample
	distrib  Distribution
}

func newVariable(domain string, sample Sample, distrib Distribution) *VariableImpl {
	return &VariableImpl{
		domain: domain,
		sample: sample,
		distrib: distrib,
	}
}

func copyVariable(source Variable, seed int64, vtype VariableType) *VariableImpl {
	var dcopy Distribution = source.Distribution().Copy(seed, vtype)

	return newVariable(source.Domain(), source.Sample(), dcopy)
}

func (this *VariableImpl) Domain() string {
	return this.domain
}

func (this *VariableImpl) Sample() Sample {
	return this.sample
}

func (this *VariableImpl) Distribution() Distribution {
	return this.distrib
}

func (this *VariableImpl) Get() interface{} {
	var index int
	var err error

	index, err = this.distrib.Select()
	if err != nil {
		return nil
	}

	return this.sample.Get(index)
}


type variableWrapper struct {
	inner  Variable
}

func (this *variableWrapper) init(inner Variable) {
	this.inner = inner
}

func (this *variableWrapper) Domain() string {
	return this.inner.Domain()
}

func (this *variableWrapper) Sample() Sample {
	return this.inner.Sample()
}

func (this *variableWrapper) Distribution() Distribution {
	return this.inner.Distribution()
}

func (this *variableWrapper) Get() interface{} {
	return this.inner.Get()
}


type FloatVariable interface {
	Variable

	TryGetFloat() (float64, bool)

	GetFloat(float64) float64
}


type floatVariableWrapper struct {
	variableWrapper
}

func newFloatVariable(inner Variable) FloatVariable {
	var this floatVariableWrapper

	this.init(inner)

	return &this
}

func newFloatImmediate(value float64) FloatVariable {
	var precision, tmp float64

	precision = 1

	for {
		tmp = value / precision

		if float64(int(tmp)) != tmp {
			tmp = precision / 10

			if tmp == 0 {
				break
			}

			precision = tmp
		} else {
			break
		}
	}

	return newFloatVariable(newVariable("float",
		newFloatSample(value, value, precision),
		newUniformDistribution(1, 0, TypeRegular)))
}

func (this *floatVariableWrapper) TryGetFloat() (float64, bool) {
	var opaque interface{} = this.Get()

	if opaque == nil {
		return 0, false
	}

	return opaque.(float64), true
}

func (this *floatVariableWrapper) GetFloat(defaultValue float64) float64 {
	var ret float64
	var ok bool

	ret, ok = this.TryGetFloat()

	if ok {
		return ret
	} else {
		return defaultValue
	}
}


type IntVariable interface {
	FloatVariable

	TryGetInt() (int, bool)

	GetInt(int) int
}


type intVariableWrapper struct {
	variableWrapper
}

func newIntVariable(inner Variable) IntVariable {
	var this intVariableWrapper

	this.init(inner)

	return &this
}

func newIntImmediate(value int) IntVariable {
	return newIntVariable(newVariable("integer",
		newIntSample(value, value),
		newUniformDistribution(1, 0, TypeRegular)))
}

func (this *intVariableWrapper) TryGetInt() (int, bool) {
	var opaque interface{} = this.Get()

	if opaque == nil {
		return 0, false
	}

	return opaque.(int), true
}

func (this *intVariableWrapper) TryGetFloat() (float64, bool) {
	var val int
	var ok bool

	val, ok = this.TryGetInt()

	return float64(val), ok
}

func (this *intVariableWrapper) GetInt(defaultValue int) int {
	var ret int
	var ok bool

	ret, ok = this.TryGetInt()

	if ok {
		return ret
	} else {
		return defaultValue
	}
}

func (this *intVariableWrapper) GetFloat(defaultValue float64) float64 {
	var ret float64
	var ok bool

	ret, ok = this.TryGetFloat()

	if ok {
		return ret
	} else {
		return defaultValue
	}
}


type StringVariable interface {
	TryGetString() (string, bool)

	GetString(string) string
}

type stringVariableWrapper struct {
	variableWrapper
}

func newStringVariable(inner Variable) StringVariable {
	var this stringVariableWrapper

	this.init(inner)

	return &this
}

func newStringImmediate(value string) StringVariable {
	var elements []interface{} = make([]interface{}, 1)

	elements[0] = value

	return newStringVariable(newVariable("string", 
		newElementSample(elements), 
		newUniformDistribution(1, 0, TypeRegular)))
}

func (this *stringVariableWrapper) TryGetString() (string, bool) {
	var opaque interface{} = this.Get()

	if opaque == nil {
		return "", false
	}

	return opaque.(string), true
}

func (this *stringVariableWrapper) GetString(defaultValue string) string {
	var ret string
	var ok bool

	ret, ok = this.TryGetString()

	if ok {
		return ret
	} else {
		return defaultValue
	}
}


type variableBaseDefinition struct {
	expr     BenchmarkExpression
	name     string
	vtype    VariableType
	seed     int64
}

func parseVariable(expr BenchmarkExpression) (Variable, string, string, error) {
	var def variableBaseDefinition
	var field BenchmarkExpression
	var vtypestr string
	var err error

	def.name, err = expr.name()
	if err != nil {
		return nil, "", "", err
	}

	vtypestr, err = expr.etype()
	if err != nil {
		vtypestr = "rvar"
	}

	if vtypestr == "rvar" {
		def.vtype = TypeRegular
	} else if vtypestr == "iter" {
		def.vtype = TypeOnce
	} else if vtypestr == "loop" {
		def.vtype = TypeLoop
	} else {
		return nil, "", "", fmt.Errorf("%s: unknown variable type " +
			"'%s'", expr.FullPosition(), vtypestr)
			
	}

	field, err = expr.TryField("seed")
	if err == nil {
		def.seed, err = parseSeed(field)
		if err != nil {
			return nil, "", "", err
		}
	} else {
		def.seed = expr.system().seed()
	}

	def.expr = expr

	_, err = expr.TryField("sample")
	if err == nil {
		return parseSampleVariable(&def)
	}

	_, err = expr.TryField("copy")
	if err == nil {
		return parseCopyVariable(&def)
	}

	_, err = expr.TryField("compose")
	if err == nil {
		return parseComposeVariable(&def)
	}

	return nil, "", "", fmt.Errorf("%s: variable must be 'sample', " +
		"'copy' or 'compose'", expr.FullPosition())
}

func parseSeed(expr BenchmarkExpression) (int64, error) {
	// var svar StringVariable
	var ivar IntVariable
	var err error
	var ival int
	var ok bool

	ivar, err = expr.Int()
	if err == nil {
		ival, ok = ivar.TryGetInt()
		if !ok {
			return 0, fmt.Errorf("%s: invalid int seed",
				expr.FullPosition())
		}

		return int64(ival), nil
	}

	// svar, err = expr.String(nil)
	// if err == nil {
	// 	sval, err = svar.TryGetString()
	// 	if err != nil {
	// 		return 0, err
	// 	}

	// 	// hash it
	// }
	
	return 0, fmt.Errorf("%s: must be an int or a string",
		expr.FullPosition())
}


func parseSampleVariable(def *variableBaseDefinition) (Variable, string, string, error) {
	var sampleFactory SampleFactory
	var randomFactory randomFactory
	var field BenchmarkExpression
	var domain, rtype string
	var distrib Distribution
	var sample Sample
	var random Random
	var err error
	var ok bool

	field = def.expr.Field("sample")

	domain, err = field.etype()
	if err != nil {
		return nil, "", "", err
	}

	sampleFactory, ok = def.expr.system().sampleFactory(domain)
	if !ok {
		return nil, "", "", fmt.Errorf("%s: unknown domain '%s'",
			def.expr.FullPosition(), domain)
	}

	sample, err = sampleFactory.Instance(field)
	if err != nil {
		return nil, "", "", err
	}

	field, err = def.expr.TryField("random")
	if err == nil {
		rtype, err = field.etype()
		if err != nil {
			return nil, "", "", err
		}

		randomFactory, ok = def.expr.system().randomFactory(rtype)
		if !ok {
			return nil, "", "", fmt.Errorf("%s: unknown random " +
				"type '%s'", def.expr.FullPosition(), rtype)
		}

		random, err = randomFactory.instance(field)
		if err != nil {
			return nil, "", "", fmt.Errorf("%s: %s",
				def.expr.FullPosition(), err.Error())
		}
	} else {
		randomFactory, ok = def.expr.system().randomFactory("uniform")
		if !ok {
			return nil, "", "", fmt.Errorf("%s: unknown random " +
				"type '%s'", def.expr.FullPosition(), rtype)
		}

		random, err = randomFactory.instance(nil)
		if err != nil {
			return nil, "", "", fmt.Errorf("%s: %s",
				def.expr.FullPosition(), err.Error())
		}
	}

	distrib = random.Instance(sample.Size(), def.seed, def.vtype)

	Tracef("create variable '%s' (%d elements)", def.name, sample.Size())

	return newVariable(domain, sample, distrib), def.name, domain, nil
}

func parseCopyVariable(def *variableBaseDefinition) (Variable, string, string, error) {
	var variable, newvar Variable
	var domain string
	var target string
	var err error
	var ok bool

	target, err = def.expr.Field("copy").target()
	if err != nil {
		return nil, "", "", nil
	}

	variable, domain, ok = def.expr.current().get(target)
	if !ok {
		return nil, "", "", fmt.Errorf("%s: unknown variable '%s'",
			def.expr.FullPosition(), target)
	}

	newvar = copyVariable(variable, def.seed, def.vtype)

	return newvar, def.name, domain, nil
}

func parseComposeVariable(def *variableBaseDefinition) (Variable, string, string, error) {
	return nil, "", "", fmt.Errorf("%s: compose variable not yet " +
		"implemented", def.expr.FullPosition())
}
