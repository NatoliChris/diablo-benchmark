package core


// A Diablo secondary seen as a benchmark resource from the Diablo primary.
// This is used by the benchmark parser to create clients used for the workload
// generation.
//
type location interface {
	// Create a new client with the given `view` of the blockchain.
	// The view is an ordered list of blockchain endpoint address as
	// specified in the setup configuration file.
	// The view represents the list of the blockchain nodes a real world
	// client would obtain from a bootstrap node.
	// The specified `kind` is a string representation of the type of
	// client it is, either a user defined string or a location in the
	// benchmark specification.
	//
	createClient(kind string, view []string) (client, error)

	// Return the list of tags associated with the location.
	// These tags are specified with the '--tag' command line option of the
	// Diablo secondary nodes.
	//
	tags() []string
}

// A blockchain client emulator seen from the Diablo primary.
// This is used by the benchmark parser to send encoded interactions to be
// triggered during the test.
//
type client interface {
	// Send an interaction `encoded` to trigger (i.e. send the transactions
	// it represents to the blockchain) at the specified `time` after the
	// begining of the test.
	//
	sendInteraction(kind string, time float64, encoded []byte) error
}


type connLocationSampleFactory struct {
	elements  []taggedElement
}

func newLocationSampleFactory(locs []location) SampleFactory {
	var elements []taggedElement = make([]taggedElement, len(locs))
	var loc location
	var i int

	for i, loc = range locs {
		elements[i] = loc
	}

	return &connLocationSampleFactory{
		elements: elements,
	}
}

func (this *connLocationSampleFactory) Instance(expr BenchmarkExpression) (Sample, error) {
	return parseFilteredElementSample(expr, this.elements)
}
