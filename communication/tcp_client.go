package communication

import (
	"encoding/binary"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
)

type ConnClient struct {
	Conn net.Conn
}

// The amount to read on first read
// byte 0 : cmd
// byte 1 : aux / len
// byte 2-9 will be the uint64
// 3 byte (16 bit number to represent size to read of payload)
const READ_LENGTH int = 9

func SetupClientTCP(addr string) (*ConnClient, error) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return &ConnClient{Conn: conn}, nil
}

//////////////////////////
// Writing Response
//////////////////////////

// Reply with an OK, just an ACK to say we got the message and all is well
func (c *ConnClient) ReplyOK() {
	_, err := c.Conn.Write(MsgOk)
	if err != nil {
		fmt.Println(err)
		_ = c.Conn.Close()
		return
	}
}

// Reply with an error: We tried the command, but something went wrong
func (c *ConnClient) ReplyERR(msg string) {
	errmsg := append(MsgErr, []byte(msg)...)
	_, err := c.Conn.Write(errmsg)
	if err != nil {
		fmt.Println(err)
		_ = c.Conn.Close()
		return
	}
}

// send; OK + DATA to the master
func (c *ConnClient) SendDataOK(data []byte) {
	// msg OK
	payload := MsgOk

	// Length = uint64
	dataLen := make([]byte, 8)
	binary.BigEndian.PutUint64(dataLen, uint64(len(data)))

	payload = append(payload, dataLen...)
	payload = append(payload, data...)

	zap.L().Debug("Sending data to master",
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

// Initial read, always reads 4 bytes long
// gets command, length or aux value
func (c *ConnClient) InitialRead() ([]byte, error) {
	buf := make([]byte, READ_LENGTH)

	_, err := c.Conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (c *ConnClient) ReadSplit(totalSize uint64) ([]byte, error) {
	fullData := make([]byte, 0)
	buffer := make([]byte, 1024)
	readLen := 0

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

// Read to the given size
func (c *ConnClient) ReadSize(size uint64) ([]byte, error) {

	if size > 65500 {
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

func (c *ConnClient) CloseConn() {
	_ = c.Conn.Close()
}
