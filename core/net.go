package core


import (
	"bufio"
	"net"
)


type primaryConn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
}

func newPrimaryConn(conn net.Conn) *primaryConn {
	return &primaryConn{
		conn: conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}
}

func (this *primaryConn) init(fromSecondary *msgSecondaryParameters) (*msgPrimaryParameters, error) {
	var fromPrimary *msgPrimaryParameters
	var err error

	fromPrimary, err = decodeMsgPrimaryParameters(this.reader)
	if err != nil {
		return nil, err
	}

	err = fromSecondary.encode(this.writer)
	if err != nil {
		return nil, err
	}

	err = this.writer.Flush()
	if err != nil {
		return nil, err
	}

	return fromPrimary, nil
}

func (this *primaryConn) waitPrepare() (msgPrepare, error) {
	return decodeMsgPrepare(this.reader)
}

func (this *primaryConn) syncReady() error {
	var err error

	err = (&msgPrepareDone{}).encode(this.writer)
	if err != nil {
		return err
	}

	return this.writer.Flush()
}

func (this *primaryConn) waitStart() (*msgStart, error) {
	return decodeMsgStart(this.reader)
}

func (this *primaryConn) pushResult(fromSecondary msgResult) error {
	var err error
	var ok bool

	err = fromSecondary.encode(this.writer)
	if err != nil {
		return err
	}

	_, ok = fromSecondary.(*msgResultDone)
	if ok {
		err = this.writer.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *primaryConn) Close() error {
	return this.conn.Close()
}


type secondaryConn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
}

func newSecondaryConn(conn net.Conn) *secondaryConn {
	return &secondaryConn{
		conn: conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}
}

func (this *secondaryConn) init(fromPrimary *msgPrimaryParameters) (*msgSecondaryParameters, error) {
	var err error

	err = fromPrimary.encode(this.writer)
	if err != nil {
		return nil, err
	}

	err = this.writer.Flush()
	if err != nil {
		return nil, err
	}

	return decodeMsgSecondaryParameters(this.reader)
}

func (this *secondaryConn) sendPrepare(fromPrimary msgPrepare) error {
	return fromPrimary.encode(this.writer)
}

func (this *secondaryConn) syncReady() error {
	var err error

	err = (&msgPrepareDone{}).encode(this.writer)
	if err != nil {
		return err
	}

	err = this.writer.Flush()
	if err != nil {
		return err
	}

	_, err = decodeMsgPrepare(this.reader)
	if err != nil {
		return err
	}

	return nil
}

func (this *secondaryConn) sendStart(fromPrimary *msgStart) error {
	var err error

	err = fromPrimary.encode(this.writer)
	if err != nil {
		return err
	}

	return this.writer.Flush()
}

func (this *secondaryConn) pullResult() (msgResult, error) {
	return decodeMsgResult(this.reader)
}

func (this *secondaryConn) addr() string {
	return this.conn.RemoteAddr().String()
}

func (this *secondaryConn) Close() error {
	return this.conn.Close()
}
