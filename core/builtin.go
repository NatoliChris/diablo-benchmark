package core


import (
	"fmt"
	"math/rand"
)


type elementSample struct {
	elements []interface{}
}

func newElementSample(elements []interface{}) Sample {
	return &elementSample{
		elements,
	}
}

func (this *elementSample) Size() int {
	return len(this.elements)
}

func (this *elementSample) Get(index int) interface{} {
	return this.elements[index]
}


type IntSample struct {
	offset  int
	size    int
}

func NewIntSample(from, to int) *IntSample {
	if to < from {
		return &IntSample{ offset: from, size: 0 }
	} else {
		return &IntSample{ offset: from, size: to - from + 1 }
	}
}

func (this *IntSample) Size() int {
	return this.size
}

func (this *IntSample) GetInt(index int) int {
	return (this.offset + index)
}

func (this *IntSample) Get(index int) interface{} {
	return this.GetInt(index)
}

type intSampleFactory struct {
}

func (this *intSampleFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	var from, to int
	var err error

	from, err = expr.Field("from").GetInt()
	if err != nil {
		return nil, err
	}

	to, err = expr.Field("to").GetInt()
	if err != nil {
		return nil, err
	}

	return NewIntSample(from, to), nil
}


type floatSample struct {
	offset     int
	size       int
	precision  float64
}

func newFloatSample(from, to, precision float64) *floatSample {
	if to < from {
		return &floatSample{
			offset: int(from),
			size: 0,
			precision: precision, }
	} else {
		return &floatSample{
			offset: int(from / precision),
			size: int((to - from) / precision),
			precision: precision,
		}
	}
}

func (this *floatSample) Size() int {
	return this.size
}

func (this *floatSample) GetFloat(index int) float64 {
	return (float64(this.offset + index) * this.precision)
}

func (this *floatSample) Get(index int) interface{} {
	return this.GetFloat(index)
}

type floatSampleFactory struct {
}

func (this *floatSampleFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	var from, to, precision, tmp float64
	var field BenchmarkExpression
	var err error

	from, err = expr.Field("from").GetFloat()
	if err != nil {
		return nil, err
	}

	to, err = expr.Field("to").GetFloat()
	if err != nil {
		return nil, err
	}

	field, err = expr.TryField("precision")
	if err == nil {
		precision, err = field.GetFloat()
		if err != nil {
			return nil, err
		}
	} else {
		precision = 1

		for {
			if precision == 0 {
				return nil, fmt.Errorf("%s: failed to " +
					"infer precision", expr.FullPosition())
			}

			tmp = from / precision
			if float64(int(tmp)) != tmp {
				precision /= 10
				continue
			}

			tmp = to / precision
			if float64(int(tmp)) != tmp {
				precision /= 10
				continue
			}

			break
		}

	}

	return newFloatSample(from, to, precision), nil
}



type uniformRandom struct {
}

func (this *uniformRandom) Instance(size int, seed int64, rtype VariableType) Distribution {
	return newUniformDistribution(size, seed, rtype)
}


type uniformDistribution struct {
	rtype   VariableType
	rand    *rand.Rand
	size    int
	values  []int
}

func newUniformDistribution(size int, seed int64, rtype VariableType) *uniformDistribution {
	var values []int
	var i int

	if rtype == TypeRegular {
		values = nil
	} else {
		values = make([]int, size)
		for i = range values {
			values[i] = i
		}
	}

	return &uniformDistribution{
		rtype: rtype,
		rand: rand.New(rand.NewSource(seed)),
		size: size,
		values: values,
	}
}

func (this *uniformDistribution) Select() (int, error) {
	var index, value int

	if this.size == 0 {
		return -1, fmt.Errorf("random space exhausted")
	}

	index = this.rand.Int() % this.size

	if this.values == nil {
		value = index
	} else {
		value = this.values[index]
	}

	if this.rtype != TypeRegular {
		this.values[index] = this.values[this.size - 1]
		this.values[this.size - 1] = value
		this.size -= 1

		if this.size == 0 {
			if this.rtype == TypeOnce {
				this.values = nil
			} else if this.rtype == TypeLoop {
				this.size = len(this.values)
			}
		}
	}

	return value, nil
}

func (this *uniformDistribution) Copy(seed int64, rtype VariableType) Distribution {
	var values []int
	var i int

	if (rtype == TypeRegular) && (this.values == nil) {
		values = nil
	} else {
		values = make([]int, this.size)

		if this.values == nil {
			for i = range values {
				values[i] = i
			}
		} else {
			for i = range values {
				values[i] = this.values[i]
			}
		}
	}

	return &uniformDistribution{
		rtype: rtype,
		rand: rand.New(rand.NewSource(seed)),
		size: this.size,
		values: values,
	}
}


type uniformRandomFactory struct {
}

func (this *uniformRandomFactory) Instance(BenchmarkExpression) (Random, error) {
	return &uniformRandom{}, nil
}
