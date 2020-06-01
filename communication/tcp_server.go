package communication

import (
	"bytes"
	"errors"
	"fmt"
	"net"
)

// Master server struct that contains the listener and the
// list of all the clients.
type MasterServer struct {
	Listener        net.Listener // TCP listener listening for incoming clients
	Clients         []net.Conn   // Any connected clients so that they can communicate with the master
	ExpectedClients int          // The number of expected clients to connect
}

// Generates a new "Listener" by creating the TCP server.
func SetupMasterTCP(addr string, expectedClients int) (*MasterServer, error) {
	listener, err := net.Listen("tcp", addr)

	// If we can't make a listener, we
	// should fail graciously but immediately.
	if err != nil {
		return nil, err
	}

	return &MasterServer{Listener: listener, ExpectedClients: expectedClients}, nil
}

// A listener that will run in a thread to
// handle any client connections.
func (s *MasterServer) HandleClients(readyChannel chan bool) {

	for {
		c, err := s.Listener.Accept()

		if err != nil {
			// Log the error here
			fmt.Println(err)
		}

		s.Clients = append(s.Clients, c)

		if len(s.Clients) == s.ExpectedClients {
			readyChannel <- true
			break
		}
	}
}

// // This function is used to send and wait for the OK byte to be
// // received. This takes a channel and replies on the channel once OK or err is received.
// func (s *MasterServer) sendAndWaitOKAsync(data []byte, client net.Conn, ch chan int) []error {
//
// }

// Send a message to a client and wait for the okay without
// the use of a channel (synchronous sending).
func (s *MasterServer) SendAndWaitOKSync(data []byte, client net.Conn) error {
	if _, err := client.Write(data); err != nil {
		// TODO: Log that we can't communicate with client
		return &ClientCommError{
			ClientInfo: client.RemoteAddr().String(),
			Err:        err,
		}
	}

	reply := make([]byte, 1)

	_, err := client.Read(reply)

	if err != nil {
		// TODO: Log client got an error
		return &ClientCommError{
			ClientInfo: client.RemoteAddr().String(),
			Err:        err,
		}
	}

	fmt.Println("GOT REPLY FROM %s", client.RemoteAddr().String())

	// If we got an error reply - it means
	// something failed on the client machine
	if bytes.Equal(MsgErr, reply) {
		// TODO: Add a "get X bytes for the error reason"
		return &ClientErrorReply{
			Info: client.RemoteAddr().String(),
			Err:  errors.New("client error received"),
		}
	}

	return nil
}

func (s *MasterServer) PrepareBenchmarkClients() []error {

	errorList := make([]error, 0)

	for _, c := range s.Clients {
		err := s.SendAndWaitOKSync(MsgPrepare, c)
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	return errorList
}

func (s *MasterServer) SendBlockchainType() error {
	return nil
}

func (s *MasterServer) SendWorkload() error {

	return nil
}

func (s *MasterServer) RunBenchmark() []error {

	errorList := make([]error, 0)

	for _, c := range s.Clients {
		err := s.SendAndWaitOKSync(MsgRun, c)
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	return errorList
}

func (s *MasterServer) GetResults() error {

	return nil
}

// Close the client connections
func (s *MasterServer) CloseClients() {
	for i, c := range s.Clients {
		fmt.Println(fmt.Sprintf("Closing Client %d @ %s", i, c.RemoteAddr().String()))
		c.Close()
	}
}

// Close the listener and exit
func (s *MasterServer) Close() {
	s.Listener.Close()
}
