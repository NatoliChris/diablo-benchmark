package communication

import (
	"fmt"
	"go.uber.org/zap"
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

		switch cmd[0] {
		case MsgPrepare[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "PREPARE"))
		case MsgWorkload[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "WORKLOAD"))
		case MsgRun[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "RUN"))
		case MsgResults[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "RESULTS"))
		}

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
