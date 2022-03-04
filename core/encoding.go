package core


import (
	"encoding/binary"
	"fmt"
	"io"
)


type msgPrimaryParameters struct {
	sysname      string
	chainParams  map[string]string
	maxDelay     float64
	maxSkew      float64
}

func decodeMsgPrimaryParameters(src io.Reader) (*msgPrimaryParameters, error) {
	var buf []byte = make([]byte, 255)
	var this msgPrimaryParameters
	var key, value string
	var err error
	var i, n, k, v int

	_, err = io.ReadFull(src, buf[:1])
	if err != nil {
		return nil, err
	}

	n = int(buf[0])

	_, err = io.ReadFull(src, buf[:n])
	if err != nil {
		return nil, err
	}

	this.sysname = string(buf[:n])

	_, err = io.ReadFull(src, buf[:1])
	if err != nil {
		return nil, err
	}

	n = int(buf[0])

	this.chainParams = make(map[string]string, n)

	for i = 0; i < n; i++ {
		_, err = io.ReadFull(src, buf[:2])
		if err != nil {
			return nil, err
		}

		k = int(buf[0])
		v = int(buf[1])

		_, err = io.ReadFull(src, buf[:k])
		if err != nil {
			return nil, err
		}

		key = string(buf[:k])

		_, err = io.ReadFull(src, buf[:v])
		if err != nil {
			return nil, err
		}

		value = string(buf[:v])

		this.chainParams[key] = value
	}

	err = binary.Read(src, binary.LittleEndian, &this.maxDelay)
	if err != nil {
		return nil, err
	}

	err = binary.Read(src, binary.LittleEndian, &this.maxSkew)
	if err != nil {
		return nil, err
	}

	return &this, nil
}

func (this *msgPrimaryParameters) encode(dest io.Writer) error {
	var key, value string
	var buf []byte
	var err error

	if len(this.sysname) > 255 {
		return fmt.Errorf("interface name '%s' too long (%d bytes)",
			this.sysname, len(this.sysname))
	}

	if len(this.chainParams) > 255 {
		return fmt.Errorf("too many chain parameters (%d)",
			len(this.chainParams))
	}

	for key, value = range this.chainParams {
		if len(key) > 255 {
			return fmt.Errorf("chain parameter name '%s' too " +
				"long (%d bytes)", key, len(key))
		}

		if len(value) > 255 {
			return fmt.Errorf("chain parameter value '%s' too " +
				"long (%d bytes)", value, len(value))
		}
	}

	buf = make([]byte, 2)

	buf[0] = uint8(len(this.sysname))
	_, err = dest.Write(buf[:1])
	if err != nil {
		return err
	}

	_, err = io.WriteString(dest, this.sysname)
	if err != nil {
		return err
	}

	buf[0] = uint8(len(this.chainParams))
	_, err = dest.Write(buf[:1])
	if err != nil {
		return err
	}

	for key, value = range this.chainParams {
		buf[0] = uint8(len(key))
		buf[1] = uint8(len(value))
		_, err = dest.Write(buf)
		if err != nil {
			return err
		}

		_, err = io.WriteString(dest, key)
		if err != nil {
			return err
		}

		_, err = io.WriteString(dest, value)
		if err != nil {
			return err
		}
	}

	err = binary.Write(dest, binary.LittleEndian, this.maxDelay)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, this.maxSkew)
	if err != nil {
		return err
	}

	return nil
}


type msgSecondaryParameters struct {
	tags  []string
}

func decodeMsgSecondaryParameters(src io.Reader) (*msgSecondaryParameters, error) {
	var buf []byte = make([]byte, 255)
	var this msgSecondaryParameters
	var err error
	var i, n int

	_, err = io.ReadFull(src, buf[:1])
	if err != nil {
		return nil, err
	}

	this.tags = make([]string, int(buf[0]))

	for i = range this.tags {
		_, err = io.ReadFull(src, buf[:1])
		if err != nil {
			return nil, err
		}

		n = int(buf[0])

		_, err = io.ReadFull(src, buf[:n])
		if err != nil {
			return nil, err
		}

		this.tags[i] = string(buf[:n])
	}

	return &this, nil
}

func (this *msgSecondaryParameters) encode(dest io.Writer) error {
	var buf []byte
	var tag string
	var err error

	if len(this.tags) > 255 {
		return fmt.Errorf("too many tags (%d)", len(this.tags))
	}

	for _, tag = range this.tags {
		if len(tag) > 255 {
			return fmt.Errorf("tag '%s' too long (%d bytes)", tag,
				len(tag))
		}
	}

	buf = make([]byte, 1)

	buf[0] = uint8(len(this.tags))
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	for _, tag = range this.tags {
		buf[0] = uint8(len(tag))
		_, err = dest.Write(buf)
		if err != nil {
			return err
		}

		_, err = io.WriteString(dest, tag)
		if err != nil {
			return err
		}
	}

	return nil
}


type msgPrepareType = uint8

type msgPrepare interface {
	encode(dest io.Writer) error
}

const (
	MSG_PREPARE_TYPE_DONE         msgPrepareType = 0
	MSG_PREPARE_TYPE_CLIENT       msgPrepareType = 1
	MSG_PREPARE_TYPE_INTERACTION  msgPrepareType = 2
)

func decodeMsgPrepare(src io.Reader) (msgPrepare, error) {
	var mtype []byte = make([]byte, 1)
	var err error
	
	_, err = io.ReadFull(src, mtype)
	if err != nil {
		return nil, err
	}

	switch (mtype[0]) {
	case MSG_PREPARE_TYPE_DONE:
		return decodeMsgPrepareDone(src)
	case MSG_PREPARE_TYPE_CLIENT:
		return decodeMsgPrepareClient(src)
	case MSG_PREPARE_TYPE_INTERACTION:
		return decodeMsgPrepareInteraction(src)
	default:
		return nil, fmt.Errorf("unknown prepare message type %d",
			mtype[0])
	}
}


type msgPrepareDone struct {
}

func decodeMsgPrepareDone(io.Reader) (msgPrepare, error) {
	return &msgPrepareDone{}, nil
}

func (this *msgPrepareDone) encode(dest io.Writer) error {
	var header []byte = make([]byte, 1)
	var err error

	header[0] = MSG_PREPARE_TYPE_DONE
	_, err = dest.Write(header)

	return err
}


type msgPrepareClient struct {
	view   []string
	index  int
}

func decodeMsgPrepareClient(src io.Reader) (msgPrepare, error) {
	var buf []byte = make([]byte, 255)
	var this msgPrepareClient
	var nview uint16
	var index uint32
	var err error
	var i, n int

	err = binary.Read(src, binary.LittleEndian, &nview)
	if err != nil {
		return nil, err
	}

	this.view = make([]string, int(nview))

	for i = range this.view {
		_, err = io.ReadFull(src, buf[:1])
		if err != nil {
			return nil, err
		}

		n = int(buf[0])

		_, err = io.ReadFull(src, buf[:n])
		if err != nil {
			return nil, err
		}

		this.view[i] = string(buf[:n])
	}

	err = binary.Read(src, binary.LittleEndian, &index)
	if err != nil {
		return nil, err
	}

	this.index = int(index)

	return &this, nil
}

func (this *msgPrepareClient) encode(dest io.Writer) error {
	var buf []byte = make([]byte, 1)
	var addr string
	var err error

	if len(this.view) > 65535 {
		return fmt.Errorf("too many addresses (%d)", len(this.view))
	}

	for _, addr = range this.view {
		if len(addr) > 255 {
			return fmt.Errorf("address '%s' too long (%d bytes)",
				addr, len(addr))
		}
	}

	if this.index >= (1 << 32) {
		return fmt.Errorf("client index too large (%d)", this.index)
	}

	buf = make([]byte, 1)
	buf[0] = MSG_PREPARE_TYPE_CLIENT
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, uint16(len(this.view)))
	if err != nil {
		return err
	}

	for _, addr = range this.view {
		buf[0] = uint8(len(addr))
		_, err = dest.Write(buf)
		if err != nil {
			return err
		}

		_, err = io.WriteString(dest, addr)
		if err != nil {
			return err
		}
	}

	err = binary.Write(dest, binary.LittleEndian, uint32(this.index))
	if err != nil {
		return err
	}

	return nil
}


type msgPrepareInteraction struct {
	index    int
	ikind    int
	time     float64
	payload  []byte
}

func decodeMsgPrepareInteraction(src io.Reader) (msgPrepare, error) {
	var buf []byte = make([]byte, 1)
	var this msgPrepareInteraction
	var lpayload uint16
	var index uint32
	var err error

	err = binary.Read(src, binary.LittleEndian, &index)
	if err != nil {
		return nil, err
	}

	this.index = int(uint8(index))

	_, err = io.ReadFull(src, buf)
	if err != nil {
		return nil, err
	}

	this.ikind = int(buf[0])

	err = binary.Read(src, binary.LittleEndian, &this.time)
	if err != nil {
		return nil, err
	}

	err = binary.Read(src, binary.LittleEndian, &lpayload)
	if err != nil {
		return nil, err
	}

	this.payload = make([]byte, int(lpayload))

	_, err = io.ReadFull(src, this.payload)
	if err != nil {
		return nil, err
	}

	return &this, nil
}

func (this *msgPrepareInteraction) encode(dest io.Writer) error {
	var buf []byte = make([]byte, 1)
	var err error

	if this.index >= (1 << 32) {
		return fmt.Errorf("client index too large (%d)", this.index)
	}

	if this.ikind > 255 {
		return fmt.Errorf("interaction kind too large (%d)",this.ikind)
	}

	// If you find yourself stuck by this limit then wait a moment before
	// to change it.
	// There is a single Diablo primary sending tons of these messages to
	// Diablo secondaries. It is a good idea to limit the work this primary
	// has to do and instead, recompute things on secondaries side when
	// possible.
	//
	if len(this.payload) > 65535 {
		return fmt.Errorf("payload too large (%d bytes)",
			len(this.payload))
	}

	buf = make([]byte, 1)
	buf[0] = MSG_PREPARE_TYPE_INTERACTION
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, uint32(this.index))
	if err != nil {
		return err
	}

	buf[0] = uint8(this.ikind)
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, this.time)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian,uint16(len(this.payload)))
	if err != nil {
		return err
	}

	_, err = dest.Write(this.payload)
	if err != nil {
		return err
	}

	return nil
}


type msgStart struct {
	duration  float64
}

func decodeMsgStart(src io.Reader) (*msgStart, error) {
	var this msgStart
	var err error

	err = binary.Read(src, binary.LittleEndian, &this.duration)
	if err != nil {
		return nil, err
	}

	return &this, nil
}

func (this *msgStart) encode(dest io.Writer) error {
	return binary.Write(dest, binary.LittleEndian, this.duration)
}


type msgResultType = uint8

type msgResult interface {
	encode(dest io.Writer) error
}

const (
	MSG_RESULT_TYPE_DONE         msgResultType = 0
	MSG_RESULT_TYPE_INTERACTION  msgResultType = 1
)

func decodeMsgResult(src io.Reader) (msgResult, error) {
	var mtype []byte = make([]byte, 1)
	var err error
	
	_, err = io.ReadFull(src, mtype)
	if err != nil {
		return nil, err
	}

	switch (mtype[0]) {
	case MSG_RESULT_TYPE_DONE:
		return decodeMsgResultDone(src)
	case MSG_RESULT_TYPE_INTERACTION:
		return decodeMsgResultInteraction(src)
	default:
		return nil, fmt.Errorf("unknown result message type %d",
			mtype[0])
	}
}


type msgResultDone struct {
}

func decodeMsgResultDone(io.Reader) (msgResult, error) {
	return &msgResultDone{}, nil
}

func (this *msgResultDone) encode(dest io.Writer) error {
	var buf []byte = make([]byte, 1)
	var err error

	buf[0] = MSG_RESULT_TYPE_DONE
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	return nil
}


type msgResultInteraction struct {
	index       int      // client index
	ikind       int      // interaction kind index
	submitTime  float64  // negative if not submitted
	commitTime  float64  // negative if not committed
	abortTime   float64  // negative if not aborted
	hasError    bool
}

func decodeMsgResultInteraction(src io.Reader) (msgResult, error) {
	var buf []byte = make([]byte, 1)
	var this msgResultInteraction
	var index uint32
	var err error

	err = binary.Read(src, binary.LittleEndian, &index)
	if err != nil {
		return nil, err
	}

	this.index = int(index)

	_, err = io.ReadFull(src, buf)
	if err != nil {
		return nil, err
	}

	this.ikind = int(uint8(buf[0]))

	err = binary.Read(src, binary.LittleEndian, &this.submitTime)
	if err != nil {
		return nil, err
	}

	err = binary.Read(src, binary.LittleEndian, &this.commitTime)
	if err != nil {
		return nil, err
	}

	err = binary.Read(src, binary.LittleEndian, &this.abortTime)
	if err != nil {
		return nil, err
	}

	_, err = io.ReadFull(src, buf)
	if err != nil {
		return nil, err
	}

	this.hasError = (buf[0] == 1)

	return &this, nil
}

func (this *msgResultInteraction) encode(dest io.Writer) error {
	var buf []byte
	var err error

	if this.index >= (1 << 32) {
		return fmt.Errorf("client index too large (%d)", this.index)
	}

	if this.ikind > 255 {
		return fmt.Errorf("interaction kind too large (%d)",this.ikind)
	}

	buf = make([]byte, 1)

	buf[0] = MSG_RESULT_TYPE_INTERACTION
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, uint32(this.index))
	if err != nil {
		return err
	}

	buf[0] = uint8(this.ikind)
	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, this.submitTime)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, this.commitTime)
	if err != nil {
		return err
	}

	err = binary.Write(dest, binary.LittleEndian, this.abortTime)
	if err != nil {
		return err
	}

	if this.hasError {
		buf[0] = 1
	} else {
		buf[0] = 0
	}

	_, err = dest.Write(buf)
	if err != nil {
		return err
	}

	return nil
}
