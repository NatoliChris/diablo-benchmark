package diem


import (
	"encoding/binary"
	"errors"
)


type transaction struct {
	from      int
	to        int
	amount    int
	sequence  int
	endpoint  int
}


func newTransaction(from, to, amount, sequence, endpoint int) *transaction {
	return &transaction{
		from:      from,
		to:        to,
		amount:    amount,
		sequence:  sequence,
		endpoint:  endpoint,
	}
}

func decodeTransaction(encoded []byte) (*transaction, error)  {
	if len(encoded) < 14 {
		return nil, errors.New("corrupted transaction")
	}

	return &transaction{
		from:      int(binary.LittleEndian.Uint16(encoded[0:])),
		to:        int(binary.LittleEndian.Uint16(encoded[2:])),
		amount:    int(binary.LittleEndian.Uint32(encoded[4:])),
		sequence:  int(binary.LittleEndian.Uint32(encoded[8:])),
		endpoint:  int(binary.LittleEndian.Uint16(encoded[12:])),
	}, nil
}

func (this *transaction) encode() []byte {
	var bytes []byte = make([]byte, 14)

	binary.LittleEndian.PutUint16(bytes[0:], uint16(this.from))
	binary.LittleEndian.PutUint16(bytes[2:], uint16(this.to))
	binary.LittleEndian.PutUint32(bytes[4:], uint32(this.amount))
	binary.LittleEndian.PutUint32(bytes[8:], uint32(this.sequence))
	binary.LittleEndian.PutUint16(bytes[12:], uint16(this.endpoint))

	return bytes
}
