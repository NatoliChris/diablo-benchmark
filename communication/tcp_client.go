package communication

import (
	"fmt"
	"net"
)

type ConnClient struct {
	Conn net.Conn
}

func SetupClientTCP(addr string) (*ConnClient, error) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return &ConnClient{Conn: conn}, nil
}

func (c *ConnClient) HandleCommands() {
	for {
		cmd := make([]byte, 1)

		_, err := c.Conn.Read(cmd)

		if err != nil {
			fmt.Println(err)
			c.Conn.Close()
			return
		}

		fmt.Println(cmd)

		// Send the reply
		_, err = c.Conn.Write(MsgOk)
		if err != nil {
			fmt.Println(err)
			c.Conn.Close()
			return
		}
	}
}

func (c *ConnClient) CloseConn() {
	c.Conn.Close()
}
