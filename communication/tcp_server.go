package communication

import (
	"bytes"
	"diablo-benchmark/blockchains"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/results"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
)

// Master server struct that contains the listener and the
// list of all the clients.
type MasterServer struct {
	Listener        net.Listener // TCP listener listening for incoming clients
	Clients         []net.Conn   // Any connected clients so that they can communicate with the master
	ExpectedClients int          // The number of expected clients to connect
}

type ClientReplyErrors []string

// Generates a new "Listener" by creating the TCP server.
func SetupMasterTCP(addr string, expectedClients int) (*MasterServer, error) {
	listener, err := net.Listen("tcp", addr)

	// If we can't make a listener, we
	// should fail gracefully but immediately.
	if err != nil {
		return nil, err
	}

	zap.L().Info("Server Started",
		zap.String("Addr", addr),
		zap.Int("Expected Clients", expectedClients))

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

		zap.L().Info(fmt.Sprintf("Client %d connected", len(s.Clients)),
			zap.String("Addr:", c.RemoteAddr().String()))

		s.Clients = append(s.Clients, c)

		if len(s.Clients) == s.ExpectedClients {
			readyChannel <- true
			break
		}
	}
}

// // This function is used to send and wait for the OK byte to be
// // received. This takes a channel and replies on the channel once OK or err is received.
func (s *MasterServer) sendAndWaitOKAsync(data []byte, client net.Conn, doneCh chan int, errCh chan error) {
	if _, err := client.Write(data); err != nil {
		// TODO: Log that we can't communicate with client
		errCh <- err
		doneCh <- 1
	}

	reply := make([]byte, 1)

	_, err := client.Read(reply)

	if err != nil {
		errCh <- err
		doneCh <- 1
	}

	fmt.Printf("GOT REPLY FROM %s\n", client.RemoteAddr().String())

	// If we got an error reply - it means
	// something failed on the client machine
	if bytes.Equal(MsgErr, reply) {
		// TODO: Add a "get X bytes for the error reason"
		errCh <- errors.New(fmt.Sprintf("failed to communicate with client %s", client.RemoteAddr().String()))
		doneCh <- 1
	}

	doneCh <- 0
	return
}

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

	reply := make([]byte, 1024)

	n, err := client.Read(reply)

	if err != nil {
		// TODO: Log client got an error
		return &ClientCommError{
			ClientInfo: client.RemoteAddr().String(),
			Err:        err,
		}
	}

	fmt.Printf("GOT REPLY FROM %s\n", client.RemoteAddr().String())

	// If we got an error reply - it means
	// something failed on the client machine
	if reply[0] == MsgErr[0] {
		// TODO: Add a "get X bytes for the error reason"
		return &ClientErrorReply{
			Info: client.RemoteAddr().String(),
			Err:  fmt.Errorf("error from client %s", string(reply[1:n])),
		}
	}

	return nil
}

// Send a message to a client and wait for the OK and data, or errors
func (s *MasterServer) sendAndWaitData(data []byte, client net.Conn) (*results.Results, error) {
	if _, err := client.Write(data); err != nil {
		// TODO log that we can't communicate with client
		return nil, &ClientCommError{
			ClientInfo: client.RemoteAddr().String(),
			Err:        err,
		}
	}

	// Read the reply AND response error (if it's an error, 1024 is to
	// encapsulate any error string passed with the data).
	initialReply := make([]byte, 1024)

	n, err := client.Read(initialReply)

	zap.L().Debug("Got client reply from RES message")

	if err != nil {
		// TODO: Log client got an error
		return nil, &ClientCommError{
			ClientInfo: client.RemoteAddr().String(),
			Err:        err,
		}
	}

	// If we got an error reply - it means
	// something failed on the client machine
	if initialReply[0] == MsgErr[0] {
		// TODO: Add a "get X bytes for the error reason"
		return nil, &ClientErrorReply{
			Info: client.RemoteAddr().String(),
			Err:  fmt.Errorf("error from client %s", string(initialReply[1:n])),
		}
	}

	// Now we have to read through the data until we end.
	// Get the length
	dataLen := binary.BigEndian.Uint64(initialReply[1:9])
	fullReply := initialReply[9:]
	fmt.Println("Got ", dataLen, fullReply)
	buffer := make([]byte, 1024)
	readLen := n - 9

	for {
		n, err := client.Read(buffer)
		zap.L().Debug(fmt.Sprintf("Client %s read %d, total %d", client.RemoteAddr().String(), n, readLen))
		if err != nil {
			if err != io.EOF {
				return nil, &ClientCommError{
					ClientInfo: client.RemoteAddr().String(),
					Err:        err,
				}
			}
			break
		}

		fullReply = append(fullReply, buffer[:n]...)
		readLen += n

		if uint64(readLen) >= dataLen {
			break
		}
	}

	zap.L().Info("Read client reply",
		zap.String("client", client.RemoteAddr().String()),
		zap.Int("numbytes", readLen))

	var res results.Results
	err = json.Unmarshal(fullReply[:dataLen], &res)

	if err != nil {
		zap.L().Error("failed to unmarshal bytes of result reply from client",
			zap.Error(err))
		return nil, err
	}
	return &res, nil
}

func (s *MasterServer) PrepareBenchmarkClients(numThreads uint32) ClientReplyErrors {

	var errorList []string

	threadBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(threadBytes, numThreads)

	for i, c := range s.Clients {
		clientID := make([]byte, 4)
		binary.BigEndian.PutUint32(clientID, uint32(i))
		payload := append(MsgPrepare, clientID...)
		payload = append(payload, threadBytes...)
		err := s.SendAndWaitOKSync(payload, c)
		if err != nil {
			zap.L().Warn("Got an error from client",
				zap.String("client", c.RemoteAddr().String()))
			errorList = append(errorList, err.Error())
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	return errorList
}

func (s *MasterServer) SendBlockchainType(bcType blockchains.BlockchainTypeMessage) ClientReplyErrors {
	// Send the blockchain type message
	var errorList []string

	fullMessage := append(MsgBc, byte(bcType))
	for _, c := range s.Clients {
		err := s.SendAndWaitOKSync(fullMessage, c)
		if err != nil {
			zap.L().Warn("error from client",
				zap.String("client", c.RemoteAddr().String()))
			errorList = append(errorList, err.Error())
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	return errorList
}

func (s *MasterServer) SendWorkload(workloads workloadgenerators.Workload) ClientReplyErrors {
	var errorList []error

	for i, c := range s.Clients {
		data := MsgWorkload

		payload, err := EncodeWorkload(workloads[i])
		if err != nil {
			errorList = append(errorList, err)
			continue
		}

		// format: cmd, len, payload
		payloadLen := uint64(len(payload))
		payloadLenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(payloadLenBytes, payloadLen)

		data = append(data, payloadLenBytes...)
		data = append(data, payload...)
		err = s.SendAndWaitOKSync(data, c)
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	return nil
}

func (s *MasterServer) RunBenchmark() ClientReplyErrors {
	zap.L().Info("\n------------\nStarting Benchmark\n------------\n")

	var errorList ClientReplyErrors

	// Channels for goroutine comms
	okCh := make(chan int, len(s.Clients))
	errCh := make(chan error, len(s.Clients))

	for _, c := range s.Clients {
		go s.sendAndWaitOKAsync(MsgRun, c, okCh, errCh)
	}

	numberDone := 0
	numberOfErrors := 0
	for {
		select {
		case clientDone := <-okCh:
			zap.L().Info("Client Done")
			numberDone++
			numberOfErrors += clientDone
			if numberDone == len(s.Clients) {
				break
			}
		}
		if numberDone == len(s.Clients) {
			break
		}
	}

	var errList ClientReplyErrors
	if numberOfErrors > 0 {
		// Check the errors and report back
		counter := 0
		for {
			select {
			case err := <-errCh:
				errList = append(errList, err.Error())
				counter++
				if counter >= numberOfErrors {
					break
				}
			}
			if counter >= numberOfErrors {
				break
			}
		}
	}

	if len(errList) == 0 {
		return nil
	}

	return errorList
}

func (s *MasterServer) GetResults() ([]results.Results, ClientReplyErrors) {
	var allResults []results.Results
	var errs ClientReplyErrors

	for _, c := range s.Clients {
		// Send the RES command, wait for the results to come back
		clientRes, err := s.sendAndWaitData(MsgResults, c)

		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		allResults = append(allResults, *clientRes)
	}

	return allResults, errs
}

// Send the final GOODBYE message and then close the connection
func (s *MasterServer) SendFin() {
	for _, c := range s.Clients {
		_ = s.SendAndWaitOKSync(MsgFin, c)
	}
}

// Master method to close all things
func (s *MasterServer) CloseAll() {
	s.CloseClients()
	s.CloseAll()
}

// Close the client connections
func (s *MasterServer) CloseClients() {
	for i, c := range s.Clients {
		zap.L().Info(fmt.Sprintf("Closing Client %d @ %s", i, c.RemoteAddr().String()))
		c.Close()
	}
}

// Close the listener and exit
func (s *MasterServer) Close() {
	s.Listener.Close()
}
