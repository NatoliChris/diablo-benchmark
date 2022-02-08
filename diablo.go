package main


import (
	"diablo-benchmark/core"
	"diablo-benchmark/blockchains/mock"
	"compress/gzip"
	"fmt"
	"encoding/json"
	"hash"
	"hash/fnv"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)


const (
	VERBOSITY_SILENT   int = 0
	VERBOSITY_FATAL    int = 1
	VERBOSITY_ERROR    int = 2
	VERBOSITY_WARNING  int = 3
	VERBOSITY_INFO     int = 4
	VERBOSITY_DEBUG    int = 5
	VERBOSITY_TRACE    int = 6

	PORT_DEFAULT       int = 5000

	MAX_DELAY_DEFAULT  float64 = 1.0
	MAX_SKEW_DEFAULT   float64 = 0.2
)


func buildSystemMap() map[string]core.BlockchainInterface {
	return map[string]core.BlockchainInterface{
		"mock": &mock.BlockchainInterface{},
	}
}

func printResult(dest io.Writer, result *core.Result) {
	var encoder *json.Encoder = json.NewEncoder(dest)
	var err error

	err = encoder.Encode(result)
	if err != nil {
		fatal("cannot encode result: %s", err.Error())
	}
}

func printStat(result *core.Result) {
	var latencies []float64 = make([]float64, 0)
	var latency, sumLatencies, lastTime float64
	var secondary *core.SecondaryResult
	var iact *core.InteractionResult
	var client *core.ClientResult
	var numSubmitted int

	numSubmitted = 0
	sumLatencies = 0
	lastTime = 0

	for _, secondary = range result.Locations {
		for _, client = range secondary.Clients {
			for _, iact = range client.Interactions {
				if iact.SubmitTime > lastTime {
					lastTime = iact.SubmitTime
				}
				if iact.CommitTime > lastTime {
					lastTime = iact.CommitTime
				}
				if iact.AbortTime > lastTime {
					lastTime = iact.AbortTime
				}

				if iact.SubmitTime < 0 {
					continue
				}

				numSubmitted += 1

				if iact.CommitTime < 0 {
					continue
				} else if iact.AbortTime >= 0 {
					continue
				} else if iact.CommitTime < iact.SubmitTime {
					continue
				}

				latency = iact.CommitTime - iact.SubmitTime
				latencies = append(latencies, latency)
				sumLatencies += latency
			}
		}
	}

	if lastTime <= 0 {
		fmt.Printf("average load: -\n")
	} else {
		fmt.Printf("average load: %.1f tx/s\n",
			float64(numSubmitted) / lastTime)
	}

	if (len(latencies) == 0) || (lastTime <= 0) {
		fmt.Printf("average throughput: -\n")
		fmt.Printf("average latency: -\n")
		fmt.Printf("median latency: -\n")
		return
	}

	sort.Float64s(latencies)

	fmt.Printf("average throughput: %.1f tx/s\n",
		float64(len(latencies)) / lastTime)
	fmt.Printf("average latency: %.3f s\n",
		sumLatencies / float64(len(latencies)))
	fmt.Printf("median latency: %.3f s\n", latencies[len(latencies)/2])
}

func setVerbosity(verbosity int) {
	var level core.LogLevel
	var logger core.Logger

	if verbosity == VERBOSITY_SILENT {
		level = core.LOG_SILENT
	} else if verbosity == VERBOSITY_FATAL {
		level = core.LOG_FATAL
	} else if verbosity == VERBOSITY_ERROR {
		level = core.LOG_ERROR
	} else if verbosity == VERBOSITY_WARNING {
		level = core.LOG_WARN
	} else if verbosity == VERBOSITY_INFO {
		level = core.LOG_INFO
	} else if verbosity == VERBOSITY_DEBUG {
		level = core.LOG_DEBUG
	} else {
		level = core.LOG_TRACE
	}

	logger = core.NewPrintLogger(os.Stderr, "", level)

	core.SetLogger(logger)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "diablo: ")
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintf(os.Stderr, "\nPlease type '%s --help' for more " +
		"information\n", os.Args[0])
	os.Exit(1)
}

func handleHelp(string) error {
	fmt.Printf("Usage: %s [--help | --version]                        " +
		"             (1)\n", os.Args[0])
	fmt.Printf("       %s primary [<options...>] <nsecondary> <setup> " +
		"<benchmark>  (2)\n", os.Args[0])
	fmt.Printf("       %s secondary [<options...>] <primary>          " +
		"             (3)\n", os.Args[0])
	fmt.Printf(`
(1) Print program information either help message or version information.

(2) Launch a Diablo primary node to run the benchmark specified by the given
    <benchmark> configuration file on the setup specified by the <setup> file
    using a number <nsecondary> of Diablo secondary node.

(3) Launch a Diablo secondary node to run the benchmark following the
    directives of the Diablo primary node at the specified <primary> address.


General Options:

  -e <str>, --env=<str>       Give the string <str> as an environment value to
                              the blockchain interface.

  -h, --help                  Print this message and exit.

  -v, --verbose=<lvl>         Increment or set verbosity to <lvl> (fatal=1,
                              error=2, warning=3, info=4, debug=5, trace=6).

  -V, --version               Print version information and exit.


Primary Options:

  --compress                  Compress output with gzip. Add a '.gz' suffix to
                              the output path is not already present.

  -d <sec>, --max-delay=<sec> Warn if secondaries submit interactions more than
                              <sec> seconds (and milliseconds) after schedule.

  -o <path>, --output=<path>  Write results in <path> instead of printing on
                              standard output.

  -p <int>, --port=<int>      Listen for Diablo secondary nodes on port <int>.

  -s <str>, --seed=<str>      Use <str> as the master random seed instead of
                              current time.

  -S <sec>, --max-skew=<sec>  Warn if secondaries schedule interactions more
                              than <sec> seconds (and milliseconds) late.

  --stat                      Print result statistics on standard output.


Secondary Options:

  -p <int>, --port=<int>      Connect to the Diablo primary node on port <int>.

  -t <str>, --tag=<str>       Attach the given <tag> to this node.

`)

	os.Exit(0)

	return nil
}

func handleVerbose(verbosity *int) {
	*verbosity++
}

func handleVerboseLevel(verbosity *int, level string) error {
	var levels = map[string]int{ "fatal": 1, "error": 2, "warning": 3,
		"information": 4, "debug": 5, "trace": 6 }
	var lowLevel, key string
	var intLevel int
	var err error

	if len(level) == 0 {
		return fmt.Errorf("invalid verbosity name '%s'", level)
	}

	intLevel, err = strconv.Atoi(level)
	if err == nil {
		if intLevel < 0 {
			return fmt.Errorf("invalid verbosity level %d",
				intLevel)
		}

		*verbosity = intLevel
		return nil
	}

	lowLevel = strings.ToLower(level)

	for key = range levels {
		if strings.HasPrefix(key, lowLevel) {
			*verbosity = levels[key]
			return nil
		}
	}

	return fmt.Errorf("invalid verbosity name '%s'", level)
}

func handleVersion(string) error {
	fmt.Printf("diablo 0.0.1\n")
	fmt.Printf("Christopher Natoli\n")
	fmt.Printf("chrisnatoli.research@gmail.com\n")
	os.Exit(0)

	return nil
}

func handlePort(port *int, value string) error {
	var intValue int
	var err error

	intValue, err = strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid port '%s'", value)
	}

	if (intValue <= 0) || (intValue > 65535) {
		return fmt.Errorf("invalid port %d", intValue)
	}

	*port = intValue

	return nil
}

func handleSeed(seed *int64, value string) error {
	var hash hash.Hash64
	var intValue int
	var err error

	intValue, err = strconv.Atoi(value)
	if err == nil {
		*seed = int64(intValue)
		return nil
	}

	hash = fnv.New64()
	hash.Write([]byte(value))

	*seed = int64(hash.Sum64())
	return nil
}

func handleSeconds(seconds *float64, value string) error {
	var floatValue float64
	var err error

	floatValue, err = strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	*seconds = floatValue
	return nil
}

func main() {
	var shorts []shortOption = make([]shortOption, 0)
	var longs []longOption = make([]longOption, 0)
	var env []string = make([]string, 0)
	var index, verbosity int
	var err error

	shorts = append(shorts, shortOption{'h', false, handleHelp})
	longs = append(longs, longOption{"help", false, handleHelp})

	verbosity = VERBOSITY_WARNING

	shorts = append(shorts, shortOption{'e', true, func(l string) error {
		env = append(env, l) ; return nil
	}})
	longs = append(longs, longOption{"env", true, func(l string) error{
		env = append(env, l) ; return nil
	}})

	shorts = append(shorts, shortOption{'v', false, func(string) error {
		handleVerbose(&verbosity) ; return nil
	}})
	longs = append(longs, longOption{"verbose", true, func(v string)error{
		return handleVerboseLevel(&verbosity, v)
	}})

	shorts = append(shorts, shortOption{'V', false, handleVersion})
	longs = append(longs, longOption{"version", false, handleVersion})

	index, err = parseOptions(os.Args[1:], shorts, longs)
	if err != nil {
		fatal("%s", err.Error())
	}

	index += 1

	if index >= len(os.Args) {
		fatal("missing either 'primary' or 'secondary'")
	}

	if os.Args[index] == "primary" {
		mainPrimary(verbosity, env, os.Args[(index+1):])
		return
	}

	if os.Args[index] == "secondary" {
		mainSecondary(verbosity, env, os.Args[(index+1):])
		return
	}

	fatal("unknown role '%s'", os.Args[index])
}

func mainPrimary(verbosity int, env []string, args []string) {
	var maxDelayClosure, portClosure, seedClosure func(string) error
	var maxSkewClosure, outputPathClosure func(string) error
	var maxDelayDefined, portDefined, seedDefined, maxSkewDefined bool
	var outputPathDefined, statDefined, compressDefined bool
	var shorts []shortOption = make([]shortOption, 0)
	var longs []longOption = make([]longOption, 0)
	var output io.WriteCloser
	var primary core.Nprimary
	var nsecondaryStr string
	var result *core.Result
	var outputPath string
	var index int
	var err error

	compressDefined = false
	longs = append(longs, longOption{"compress", false,func(string) error {
		if compressDefined {
			return fmt.Errorf("option specified twice")
		}

		compressDefined = true
		return nil
	}})

	primary.MaxDelay = MAX_DELAY_DEFAULT
	maxDelayDefined = false
	maxDelayClosure = func(l string) error {
		if maxDelayDefined {
			return fmt.Errorf("option specified twice")
		}

		maxDelayDefined = true

		return handleSeconds(&primary.MaxDelay, l)
	}
	shorts = append(shorts, shortOption{'d', true, maxDelayClosure})
	longs = append(longs, longOption{"max-delay", true, maxDelayClosure})

	shorts = append(shorts, shortOption{'h', false, handleHelp})
	longs = append(longs, longOption{"help", false, handleHelp})

	shorts = append(shorts, shortOption{'e', true, func(l string) error {
		env = append(env, l) ; return nil
	}})
	longs = append(longs, longOption{"env", true, func(l string) error{
		env = append(env, l) ; return nil
	}})

	outputPathDefined = false
	outputPathClosure = func(l string) error {
		if outputPathDefined {
			return fmt.Errorf("option specified twice")
		}

		outputPathDefined = true
		outputPath = l

		return nil
	}

	shorts = append(shorts, shortOption{'o', true, outputPathClosure})
	longs = append(longs, longOption{"output", true, outputPathClosure})

	primary.ListenPort = PORT_DEFAULT
	portDefined = false
	portClosure = func(l string) error {
		if portDefined {
			return fmt.Errorf("option specified twice")
		}

		portDefined = true

		return handlePort(&primary.ListenPort, l)
	}

	shorts = append(shorts, shortOption{'p', true, portClosure})
	longs = append(longs, longOption{"port", true, portClosure})

	primary.MasterSeed = int64(time.Now().Nanosecond())
	seedDefined = false
	seedClosure = func(l string) error {
		if seedDefined {
			return fmt.Errorf("option specified twice")
		}

		seedDefined = true

		return handleSeed(&primary.MasterSeed, l)
	}
	shorts = append(shorts, shortOption{'s', true, seedClosure})
	longs = append(longs, longOption{"seed", true, seedClosure})

	primary.MaxSkew = MAX_SKEW_DEFAULT
	maxSkewDefined = false
	maxSkewClosure = func(l string) error {
		if maxSkewDefined {
			return fmt.Errorf("option specified twice")
		}

		maxSkewDefined = true

		return handleSeconds(&primary.MaxSkew, l)
	}
	shorts = append(shorts, shortOption{'S', true, maxSkewClosure})
	longs = append(longs, longOption{"max-skew", true, maxSkewClosure})

	statDefined = false
	longs = append(longs, longOption{"stat", false, func(string) error {
		if statDefined {
			return fmt.Errorf("option specified twice")
		}

		statDefined = true
		return nil
	}})

	shorts = append(shorts, shortOption{'v', false, func(string) error {
		handleVerbose(&verbosity) ; return nil
	}})
	longs = append(longs, longOption{"verbose", true, func(v string)error{
		return handleVerboseLevel(&verbosity, v)
	}})

	index, err = parseOptions(args, shorts, longs)
	if err != nil {
		fatal("%s", err.Error())
	}

	if index >= len(args) {
		fatal("missing nsecondary operand")
	} else if (index + 1) >= len(args) {
		fatal("missing setup operand")
	} else if (index + 2) >= len(args) {
		fatal("missing benchmark operand")
	} else if (index + 3) < len(args) {
		fatal("unexpected operand '%s'", args[index + 3])
	}

	nsecondaryStr = args[index]
	primary.SetupPath = args[index + 1]
	primary.BenchmarkPath = args[index + 2]

	primary.NumSecondary, err = strconv.Atoi(nsecondaryStr)
	if err != nil {
		fatal("invalid nsecondary operand '%s'", nsecondaryStr)
	} else if primary.NumSecondary <= 0 {
		fatal("invalid nsecondary operand %d", primary.NumSecondary)
	}

	primary.SystemMap = buildSystemMap()
	primary.Env = env

	if outputPathDefined {
		if compressDefined && !strings.HasSuffix(outputPath, ".gz") {
			outputPath += ".gz"
		}

		output, err = os.Create(outputPath)
		if err != nil {
			fatal("cannot open output path '%s': %s", outputPath,
				err.Error())
		}

		defer output.Close()
	} else {
		output = os.Stdout
	}

	if compressDefined {
		output = gzip.NewWriter(output)
		defer output.Close()
	}

	setVerbosity(verbosity)

	result, err = primary.Run()
	if err != nil {
		fatal("%s", err.Error())
	}

	printResult(output, result)

	if statDefined {
		printStat(result)
	}
}

func mainSecondary(verbosity int, env []string, args []string) {
	var shorts []shortOption = make([]shortOption, 0)
	var longs []longOption = make([]longOption, 0)
	var portClosure, tagClosure func(string) error
	var tags []string = make([]string, 0)
	var secondary *core.Nsecondary
	var portDefined bool
	var index, port int
	var primary string
	var err error

	shorts = append(shorts, shortOption{'e', true, func(l string) error {
		env = append(env, l) ; return nil
	}})
	longs = append(longs, longOption{"env", true, func(l string) error{
		env = append(env, l) ; return nil
	}})

	shorts = append(shorts, shortOption{'h', false, handleHelp})
	longs = append(longs, longOption{"help", false, handleHelp})

	port = PORT_DEFAULT
	portDefined = false
	portClosure = func(l string) error {
		if portDefined {
			return fmt.Errorf("option specified twice")
		}

		portDefined = true

		return handlePort(&port, l)
	}

	shorts = append(shorts, shortOption{'p', true, portClosure})
	longs = append(longs, longOption{"port", true, portClosure})

	tagClosure = func(l string) error {
		tags = append(tags, l)
		return nil
	}

	shorts = append(shorts, shortOption{'t', true, tagClosure})
	longs = append(longs, longOption{"tag", true, tagClosure})

	shorts = append(shorts, shortOption{'v', false, func(string) error {
		handleVerbose(&verbosity) ; return nil
	}})
	longs = append(longs, longOption{"verbose", true, func(v string)error{
		return handleVerboseLevel(&verbosity, v)
	}})

	index, err = parseOptions(args, shorts, longs)
	if err != nil {
		fatal("%s", err.Error())
	}

	if index >= len(args) {
		fatal("missing primary operand")
	} else if (index + 1) < len(args) {
		fatal("unexpected operand '%s'", args[index + 1])
	}

	primary = args[index]

	setVerbosity(verbosity)

	secondary = core.NewNsecondary(primary, port, env, tags,
		buildSystemMap())

	err = secondary.Run()
	if err != nil {
		fatal("%s", err.Error())
	}
}
