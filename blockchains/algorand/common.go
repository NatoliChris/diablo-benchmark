package algorand


import (
	"encoding/binary"
	"errors"

	"diablo-benchmark/core/configs"
)


type transaction struct {
	endpoint  int       // blockchain node to which to send the transaction
	uid       int              // unique identifier (whole benchmark scope)
	raw       []byte                  // bytes to send to the endpoint node
}


func makeTransactionNote(uid int) []byte {
	var note []byte = make([]byte, 4)

	binary.LittleEndian.PutUint32(note, uint32(uid))

	return note
}

func getUidFromTransactionNote(note []byte) int {
	return int(binary.LittleEndian.Uint32(note))
}


func newTransaction(endpoint, uid int, raw []byte) transaction {
	return transaction{
		endpoint:  endpoint,
		uid:       uid,
		raw:       raw,
	}
}

func decodeTransaction(encoded []byte) (ret transaction, err error) {
	if len(encoded) < 8 {
		err = errors.New("corrupted transaction")
	} else {
		ret.endpoint = int(binary.LittleEndian.Uint32(encoded[0:]))
		ret.uid = int(binary.LittleEndian.Uint32(encoded[4:]))
		ret.raw = encoded[8:]
	}

	return
}

func (this transaction) encode() []byte {
	var header []byte = make([]byte, 8)

	binary.LittleEndian.PutUint32(header[0:], uint32(this.endpoint))
	binary.LittleEndian.PutUint32(header[4:], uint32(this.uid))

	return append(header, this.raw...)
}


const benchmarkToken =
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func parseConfig(c *configs.ChainConfig) (*Config, error) {
	var iextra, iaccount map[string]interface{}
	var ret *Config = NewConfig()
	var iaccounts []interface{}
	var i interface{}
	var node string

	for _, node = range c.Nodes {
		ret.AddNode(node, benchmarkToken)
	}

	// FIXME: why defining chainConfig.Extra as a slice? (wtf)
	iextra = c.Extra[0].(map[string]interface{})
	iaccounts = iextra["accounts"].([]interface{})

	for _, i = range iaccounts {
		iaccount = i.(map[string]interface{})
		ret.AddAccount(iaccount["address"].(string),
			iaccount["mnemonic"].(string))
	}

	return ret, nil
}
