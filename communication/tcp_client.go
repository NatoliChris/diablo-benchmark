package communication

import (
	"encoding/binary"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
)

// ConnClient provides an active connection from the secondary to
// the primary. The main action of the connection is to receive commands and to
// reply with OK or errors and results.
type ConnClient struct {
	Conn net.Conn // Active connection to the primary
}

// READLENGTH is the amount to read on first read
// byte 0 : cmd
// byte 1 : aux / len
// byte 2-9 will be the uint64
// 3 byte (16 bit number to represent size to read of payload)
const READLENGTH int = 9

// MAXREAD is the maximum value to read in a single read.
// If the size we need to read is greater, then we need to read split amounts
// and iterate through the reading.
const MAXREAD uint64 = 65500

// SetupSecondaryTCP connects to the master TCP address and return the connected client
func SetupSecondaryTCP(addr string) (*ConnClient, error) {
	// Dial the address, return the error if we cannot
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return &ConnClient{Conn: conn}, nil
}

//////////////////////////
// Writing Response
//////////////////////////

// ReplyOK replies with an OK, just an ACK to say we got the message and all is well
func (c *ConnClient) ReplyOK() {
	_, err := c.Conn.Write(MsgOk)
	if err != nil {
		fmt.Println(err)
		_ = c.Conn.Close()
		return
	}
}

// ReplyERR replies with an error: We tried the command, but something went wrong
func (c *ConnClient) ReplyERR(msg string) {
	errmsg := append(MsgErr, []byte(msg)...)
	_, err := c.Conn.Write(errmsg)
	if err != nil {
		fmt.Println(err)
		_ = c.Conn.Close()
		return
	}
}

// SendDataOK will send OK + DATA to the Primary
func (c *ConnClient) SendDataOK(data []byte) {
	// msg OK
	payload := MsgOk

	// Length = uint64
	dataLen := make([]byte, 8)
	binary.BigEndian.PutUint64(dataLen, uint64(len(data)))

	payload = append(payload, dataLen...)
	payload = append(payload, data...)

	zap.L().Debug("Sending data to primary",
		zap.Int("dataLen", len(data)))

	fmt.Println(dataLen)

	_, err := c.Conn.Write(payload)

	if err != nil {
		fmt.Println(err)
		_ = c.Conn.Close()
		return
	}
}

//////////////////////////
// Reading
//////////////////////////

// InitialRead performs a first read, always reads 4 bytes long
// gets command, length to read or aux value
func (c *ConnClient) InitialRead() ([]byte, error) {
	buf := make([]byte, READLENGTH)

	_, err := c.Conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// ReadSplit reads information over a number of "reads".
// This should happen if the size of the object to read is larger than the
// size of a full read in go.
func (c *ConnClient) ReadSplit(totalSize uint64) ([]byte, error) {
	fullData := make([]byte, 0)
	buffer := make([]byte, 1024)
	readLen := 0

	// Loop through, iteratively reading until the EOF is reached.
	for {
		numRead, err := c.Conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}

		fullData = append(fullData, buffer[:numRead]...)
		readLen += numRead

		if uint64(readLen) >= totalSize {
			break
		}
	}

	return fullData, nil
}

// ReadSize read to the given size
func (c *ConnClient) ReadSize(size uint64) ([]byte, error) {

	if size > MAXREAD {
		// split read
		return c.ReadSplit(size)
	}
	buf := make([]byte, size)

	n, err := c.Conn.Read(buf)
	if err != nil {
		return nil, err
	}

	zap.L().Debug("Read bytes",
		zap.Int("number", n))

	return buf, nil
}

// CloseConn closes the connection to the primary server
func (c *ConnClient) CloseConn() {
	_ = c.Conn.Close()
}
