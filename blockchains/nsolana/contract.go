package nsolana

import (
	"bufio"
	"bytes"
	"diablo-benchmark/core"
	"diablo-benchmark/util"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type solidityCompiler struct {
	logger core.Logger
	base   string
}

func newSolidityCompiler(logger core.Logger, base string) *solidityCompiler {
	return &solidityCompiler{
		logger: logger,
		base:   base,
	}
}

type solangExecutable struct {
	path, version, fullVersion string
	major, minor, patch        int
}

type solangContract struct {
	name          string
	requiredSpace uint64
	data          []byte
	abi           abi.ABI
	hashes        map[string][]byte // method signature => hash
}

var (
	versionRegexp      = regexp.MustCompile(`([0-9]+)\.([0-9]+)\.([0-9]+)`)
	contractNameRegexp = regexp.MustCompile(`found contract ‘(.*)’`)
	dataUsageRegexp    = regexp.MustCompile(`least (.*) bytes`)
	binaryPathRegexp   = regexp.MustCompile(`binary (.*) for`)
	abiPathRegexp      = regexp.MustCompile(`ABI (.*) for`)
)

func solangVersion(solang string) (*solangExecutable, error) {
	if solang == "" {
		solang = "solang"
	}
	var out bytes.Buffer
	cmd := exec.Command(solang, "--version")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	matches := versionRegexp.FindStringSubmatch(out.String())
	if len(matches) != 4 {
		return nil, fmt.Errorf("can't parse solang version %q", out.String())
	}
	s := &solangExecutable{path: cmd.Path, fullVersion: out.String(), version: matches[0]}
	if s.major, err = strconv.Atoi(matches[1]); err != nil {
		return nil, err
	}
	if s.minor, err = strconv.Atoi(matches[2]); err != nil {
		return nil, err
	}
	if s.patch, err = strconv.Atoi(matches[3]); err != nil {
		return nil, err
	}
	return s, nil
}

func compileSolidity(solangPath, contractPath string) (contract *solangContract, err error) {
	dir, err := ioutil.TempDir("", "diablo-solang")
	if err != nil {
		return
	}
	defer func() {
		tmpErr := os.RemoveAll(dir)
		if tmpErr != nil {
			err = tmpErr
		}
	}()
	solang, err := solangVersion(solangPath)
	if err != nil {
		return
	}
	args := []string{
		"--verbose",
		"--output", dir,
		"--target", "solana",
		contractPath,
	}
	cmd := exec.Command(solang.path, args...)
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("solang: %v\n%s", err, stderr.Bytes())
	}
	contract = &solangContract{}
	contractNameMatches := contractNameRegexp.FindStringSubmatch(stderr.String())
	if len(contractNameMatches) != 2 {
		return nil, fmt.Errorf("can't parse contract name %q", stderr.String())
	}
	contract.name = contractNameMatches[1]
	dataUsageMatches := dataUsageRegexp.FindStringSubmatch(stderr.String())
	if len(dataUsageMatches) != 2 {
		return nil, fmt.Errorf("can't parse data usage %q", stderr.String())
	}
	dataUsage, err := strconv.Atoi(dataUsageMatches[1])
	if err != nil {
		return nil, err
	}
	contract.requiredSpace = uint64(dataUsage)
	binaryPathMatches := binaryPathRegexp.FindStringSubmatch(stderr.String())
	if len(binaryPathMatches) != 2 {
		return nil, fmt.Errorf("can't parse binary path %q", stderr.String())
	}
	if contract.data, err = ioutil.ReadFile(binaryPathMatches[1]); err != nil {
		return nil, err
	}
	abiPathMatches := abiPathRegexp.FindStringSubmatch(stderr.String())
	if len(abiPathMatches) != 2 {
		return nil, fmt.Errorf("can't parse ABI path %q", stderr.String())
	}
	abiData, err := ioutil.ReadFile(abiPathMatches[1])
	if err != nil {
		return nil, err
	}
	if contract.abi, err = abi.JSON(bytes.NewReader(abiData)); err != nil {
		return nil, err
	}
	contract.hashes = make(map[string][]byte)
	for _, method := range contract.abi.Methods {
		contract.hashes[method.Sig] = method.ID
	}
	return
}

func (this *solidityCompiler) compile(name string) (*application, error) {
	this.logger.Debugf("compile contract '%s'", name)

	path := this.base + "/" + name + "/contract.sol"

	this.logger.Tracef("compile contract source in '%s'", path)

	contract, err := compileSolidity("", path)
	if err != nil {
		return nil, err
	}

	for fname := range contract.hashes {
		this.logger.Tracef("  has function %s", fname)
	}

	path = this.base + "/" + name + "/arguments"

	if !strings.HasPrefix(path, "/") {
		path = "./" + path
	}

	parser, err := util.StartServiceProcess(path)
	if err != nil {
		return nil, err
	}

	return newApplication(this.logger, contract.name, contract.abi, contract.data, contract.hashes, parser), nil
}

type application struct {
	logger  core.Logger
	name    string
	abi     abi.ABI
	text    []byte
	entries map[string][]byte
	parser  *util.ServiceProcess
	scanner *bufio.Scanner
}

func newApplication(logger core.Logger, name string, abi abi.ABI, text []byte, entries map[string][]byte, parser *util.ServiceProcess) *application {
	return &application{
		logger:  logger,
		name:    name,
		abi:     abi,
		text:    text,
		entries: entries,
		parser:  parser,
		scanner: bufio.NewScanner(parser),
	}
}

func (this *application) arguments(function string) ([]byte, error) {
	_, err := io.WriteString(this.parser, function+"\n")
	if err != nil {
		return nil, err
	}

	var fname string
	var payload []byte
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

	entry, found := this.entries[fname]
	if !found {
		return nil, fmt.Errorf("unknown function '%s'", fname)
	}

	return append(entry, payload...), nil
}
