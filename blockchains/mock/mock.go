package mock


import (
	"bytes"
	"diablo-benchmark/core"
	"diablo-benchmark/util"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
)


type BlockchainInterface struct {
}

func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	var key, value, addr string

	logger.Debugf("new builder:")

	logger.Debugf("  chain parameters:")
	for key, value = range params {
		logger.Debugf("    %s: %s", key, value)
	}

	logger.Debugf("  environment:")
	for _, value = range env {
		logger.Debugf("    %s", value)
	}

	logger.Debugf("  endpoints:")
	for addr = range endpoints {
		logger.Debugf("    %s:", addr)
		for _, value = range endpoints[addr] {
			logger.Debugf("      %s", value)
		}
	}

	return &BlockchainBuilder{ logger, 0, 0 }, nil
}

func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	var key, value string
	var delay float64
	var presign bool
	var err error
	var ok bool

	logger.Debugf("new client:")

	logger.Debugf("  chain parameters:")
	for key, value = range params {
		logger.Debugf("    %s: %s", key, value)
	}

	logger.Debugf("  environment:")
	for _, value = range env {
		logger.Debugf("    %s", value)
	}

	logger.Debugf("  endpoints:")
	for _, value = range view {
		logger.Debugf("    %s", value)
	}

	value, ok = params["delay"]
	if !ok {
		delay = 1.0
	} else {
		delay, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid delay parameter: '%s'",
				value)
		}

		if delay < 0 {
			return nil, fmt.Errorf("invalid delay parameter: %f",
				delay)
		}
	}

	value, ok = params["pre-sign"]
	if !ok {
		presign = false
	} else {
		presign, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid pre-sign parameter: " +
				"'%s'", value)
		}
	}

	return &BlockchainClient{ delay, presign, logger }, nil
}


type BlockchainBuilder struct {
	logger        core.Logger
	nextAccount   int
	nextContract  int
}

func (this *BlockchainBuilder) CreateAccount(stake int) (interface{}, error) {
	var account int = this.nextAccount

	this.logger.Tracef("mint new account %d with stake %d", account, stake)
	this.nextAccount += 1

	return account, nil
}

func (this *BlockchainBuilder) CreateContract(name string) (interface{}, error) {
	var contract int = this.nextContract

	this.logger.Tracef("upload new contract '%s' with id %d", name,
		contract)
	this.nextContract += 1

	return contract, nil
}

func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(stake int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	var buf bytes.Buffer

	util.NewMonadOutputWriter(&buf).
		SetOrder(binary.LittleEndian).
		Write(uint64(stake)).
		Write(uint64(from.(int))).
		Write(uint64(to.(int))).
		Trust()

	return buf.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInvoke(from, contract interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	var buf bytes.Buffer

	util.NewMonadOutputWriter(&buf).
		SetOrder(binary.LittleEndian).
		Write(uint64(from.(int))).
		Write(uint64(contract.(int))).
		Trust()

	return buf.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return nil, fmt.Errorf("unknown interaction type '%s'", itype)
}


type BlockchainClient struct {
	delay    float64
	presign  bool
	logger   core.Logger
}

func (this *BlockchainClient) DecodePayload(encoded []byte) (interface{}, error) {
	var buf *bytes.Buffer = bytes.NewBuffer(encoded)
	var contract, from, to, stake int
	var tx string

	if len(encoded) == 16 {
		util.NewMonadInputReader(buf).
			SetOrder(binary.LittleEndian).
			ReadUint64(&from).
			ReadUint64(&contract).
			Trust()
		tx = fmt.Sprintf("invoke(%d -> %d)", from, contract)
	} else if len(encoded) == 24 {
		util.NewMonadInputReader(buf).
			SetOrder(binary.LittleEndian).
			ReadUint64(&stake).
			ReadUint64(&from).
			ReadUint64(&to).
			Trust()
		tx = fmt.Sprintf("transfer(%d : %d -> %d)", stake, from, to)
	} else {
		return nil, fmt.Errorf("invalid interaction payload %v",
			encoded)
	}

	if this.presign {
		this.logger.Tracef("sign interaction '%s'", tx)
		time.Sleep(1000 * time.Millisecond)
		tx = fmt.Sprintf("signed(%d : %s)", from, tx)
	}

	return tx, nil
}

func (this *BlockchainClient) TriggerInteraction(iact core.Interaction) error {
	var tx string = iact.Payload().(string)

	if !this.presign {
		this.logger.Tracef("sign interaction '%s'", tx)
		time.Sleep(1000 * time.Millisecond)
		tx = fmt.Sprintf("signed(%s)", tx)
	}

	this.logger.Tracef("submit interaction '%s'", tx)
	iact.ReportSubmit()

	time.Sleep(time.Duration(this.delay * float64(time.Second)))

	this.logger.Tracef("commit interaction '%s'", tx)
	iact.ReportCommit()

	return nil
}
