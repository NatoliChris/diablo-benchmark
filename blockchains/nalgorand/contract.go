package nalgorand


import (
	"bufio"
	"bytes"
	"context"
	"diablo-benchmark/core"
	"diablo-benchmark/util"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/types"

	"gopkg.in/yaml.v3"
)


type tealCompiler struct {
	logger  core.Logger
	base    string
	client  *algod.Client
	ctx     context.Context
}

const clearContractSource = "#pragma version 5\nint 1\nreturn\n"

func newTealCompiler(logger core.Logger, base string, client *algod.Client, ctx context.Context) *tealCompiler {
	return &tealCompiler{
		logger: logger,
		base: base,
		client: client,
		ctx: ctx,
	}
}

func (this *tealCompiler) compile(name string) (*application, error) {
	var local, global *types.StateSchema
	var approvalCode, clearCode []byte
	var parser *util.ServiceProcess
	var path string
	var err error

	this.logger.Debugf("compile contract '%s'", name)

	approvalCode, err = this.getApprovalCode(name)
	if err != nil {
		return nil, err
	}

	this.logger.Debugf("compile approval code (%d bytes)",
		len(approvalCode))
	approvalCode, err = this.compileSource(approvalCode)
	if err != nil {
		return nil, err
	}

	clearCode, err = this.getClearCode(name)
	if err != nil {
		return nil, err
	}

	this.logger.Debugf("compile clear code (%d bytes)", len(clearCode))
	clearCode, err = this.compileSource(clearCode)
	if err != nil {
		return nil, err
	}

	local, global, err = this.getSchemas(name)
	if err != nil {
		return nil, err
	}

	this.logger.Debugf("schemas for contract '%s':", name)
	this.logger.Debugf("  local.ints   = %d", local.NumUint)
	this.logger.Debugf("  local.bytes  = %d", local.NumByteSlice)
	this.logger.Debugf("  global.ints  = %d", global.NumUint)
	this.logger.Debugf("  global.bytes = %d", global.NumByteSlice)

	path = this.base + "/" + name + "/arguments"
	if !strings.HasPrefix(path, "/") {
		path = "./" + path
	}

	parser, err = util.StartServiceProcess(path)
	if err != nil {
		return nil, err
	}

	return &application{
		logger: this.logger,
		approvalCode: approvalCode,
		clearCode: clearCode,
		localSchema: *local,
		globalSchema: *global,
		parser: parser,
		scanner: bufio.NewScanner(parser),
	}, nil
}

func (this *tealCompiler) getApprovalCode(name string) ([]byte, error) {
	var code []byte
	var path string
	var err error

	path = this.base + "/" + name + "/approval.py"
	this.logger.Debugf("try fetch approval code in '%s'", path)
	code, err = this.getPytealSource(path)
	if err == nil {
		return code, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	path = this.base + "/" + name + "/approval.teal"
	this.logger.Debugf("try fetch approval code in '%s'", path)
	code, err = this.getTealSource(path)
	if err == nil {
		return code, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	
	return nil, fmt.Errorf("cannot find approval code for '%s' in '%s'",
		name, this.base)
}

func (this *tealCompiler) getClearCode(name string) ([]byte, error) {
	var code []byte
	var path string
	var err error

	path = this.base + "/" + name + "/clear.py"
	this.logger.Debugf("try fetch clear code in '%s'", path)
	code, err = this.getPytealSource(path)
	if err == nil {
		return code, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	path = this.base + "/" + name + "/clear.teal"
	this.logger.Debugf("try fetch clear code in '%s'", path)
	code, err = this.getTealSource(path)
	if err == nil {
		return code, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	this.logger.Debugf("rely on default clear code")
	return []byte(clearContractSource), nil
}

func (this *tealCompiler) getPytealSource(path string) ([]byte, error) {
	var buffer bytes.Buffer
	var cmd *exec.Cmd
	var err error

	if !strings.HasPrefix(path, "/") {
		path = "./" + path
	}

	cmd = exec.Command(path)
	cmd.Stdout = &buffer

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (this *tealCompiler) getTealSource(path string) ([]byte, error) {
	var buffer []byte
	var file *os.File
	var err error

	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buffer, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func (this *tealCompiler) compileSource(source []byte) ([]byte, error) {
	var compilation models.CompileResponse
	var err error

	compilation, err = this.client.TealCompile(source).Do(this.ctx)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(compilation.Result)
}

type yamlSchemas struct {
	Local   yamlSchema  `yaml:"local"`
	Global  yamlSchema  `yaml:"global"`
}

type yamlSchema struct {
	Ints   int  `yaml:"ints"`
	Bytes  int  `yaml:"bytes"`
}

func (this *tealCompiler) getSchemas(name string) (*types.StateSchema, *types.StateSchema, error) {
	var decoder *yaml.Decoder
	var schemas yamlSchemas
	var file *os.File
	var path string
	var err error

	path = this.base + "/" + name + "/schemas.yaml"
	this.logger.Debugf("try fetch schemas in '%s'", path)
	file, err = os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, err
		}

		return &types.StateSchema{
			NumUint: uint64(1),
			NumByteSlice: uint64(1),
		}, &types.StateSchema{
			NumUint: uint64(1),
			NumByteSlice: uint64(1),
		}, nil
	}

	decoder = yaml.NewDecoder(file)
	err = decoder.Decode(&schemas)

	file.Close()

	if err != nil {
		return nil, nil, err
	}

	return &types.StateSchema{
		NumUint: uint64(schemas.Local.Ints),
		NumByteSlice: uint64(schemas.Local.Bytes),
	}, &types.StateSchema{
		NumUint: uint64(schemas.Global.Ints),
		NumByteSlice: uint64(schemas.Global.Bytes),
	}, nil
}


type application struct {
	logger        core.Logger
	approvalCode  []byte
	clearCode     []byte
	localSchema   types.StateSchema
	globalSchema  types.StateSchema
	parser        *util.ServiceProcess
	scanner       *bufio.Scanner
}

func (this *application) arguments(function string) ([][]byte, error) {
	var args [][]byte = make([][]byte, 0)
	var line string
	var arg []byte
	var err error

	_, err = io.WriteString(this.parser, function + "\n")
	if err != nil {
		return nil, err
	}

	for this.scanner.Scan() {
		line = this.scanner.Text()
		if line == "" {
			break
		}

		arg, err = base64.StdEncoding.DecodeString(line)
		if err != nil {
			return nil, err
		}

		args = append(args, arg)
	}

	return args, nil
}
