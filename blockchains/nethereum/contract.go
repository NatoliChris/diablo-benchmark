package nethereum


import (
	"bufio"
	"diablo-benchmark/core"
	"diablo-benchmark/util"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common/compiler"
)


type solidityCompiler struct {
	logger  core.Logger
	base    string
}

func newSolidityCompiler(logger core.Logger, base string) *solidityCompiler {
	return &solidityCompiler{
		logger: logger,
		base: base,
	}
}

func (this *solidityCompiler) compile(name string) (*application, error) {
	var contracts map[string]*compiler.Contract
	var contract *compiler.Contract
	var parser *util.ServiceProcess
	var entries map[string][]byte
	var path, fname, hash string
	var text []byte
	var err error

	this.logger.Debugf("compile contract '%s'", name)

	path = this.base + "/" + name + "/contract.sol"

	this.logger.Tracef("compile contract source in '%s'", path)

	contracts, err = compiler.CompileSolidity("", path)
	if err != nil {
		return nil, err
	} else if len(contracts) < 1 {
		return nil, fmt.Errorf("no contract in '%s'", path)
	} else if len(contracts) > 1 {
		return nil, fmt.Errorf("more than one contract in '%s'", path)
	}

	for _, contract = range contracts {
		break
	}

	text, err = hex.DecodeString(contract.Code[2:])
	if err != nil {
		return nil, err
	}

	entries = make(map[string][]byte)
	for fname, hash = range contract.Hashes {
		entries[fname], err = hex.DecodeString(hash)
		if err != nil {
			return nil, err
		}

		this.logger.Tracef("  has function %s", fname)
	}

	path = this.base + "/" + name + "/arguments"

	if !strings.HasPrefix(path, "/") {
		path = "./" + path
	}

	parser, err = util.StartServiceProcess(path)
	if err != nil {
		return nil, err
	}

	return newApplication(this.logger, text, entries, parser), nil
}

type application struct {
	logger   core.Logger
	text     []byte
	entries  map[string][]byte
	parser   *util.ServiceProcess
	scanner  *bufio.Scanner
}

func newApplication(logger core.Logger, text []byte, entries map[string][]byte, parser *util.ServiceProcess) *application {
	return &application{
		logger: logger,
		text: text,
		entries: entries,
		parser: parser,
		scanner: bufio.NewScanner(parser),
	}
}

func (this *application) arguments(function string) ([]byte, error) {
	var entry, payload []byte
	var fname string
	var found bool
	var err error

	_, err = io.WriteString(this.parser, function + "\n")
	if err != nil {
		return nil, err
	}

	if this.scanner.Scan() {
		fname = this.scanner.Text()

		if this.scanner.Scan() {
			payload, err = base64.StdEncoding.
				DecodeString(this.scanner.Text())
			if err != nil {
				return nil, err
			}
		}
	}

	if payload == nil {
		err = this.scanner.Err()
		if err == nil {
			err = fmt.Errorf("EOF")
		}
		return nil, err
	}

	entry, found = this.entries[fname]
	if !found {
		return nil, fmt.Errorf("unknown function '%s'", fname)
	}

	return append(entry, payload...), nil
}
