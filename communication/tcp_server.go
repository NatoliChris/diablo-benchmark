package communication

import (
	"bytes"
	"diablo-benchmark/blockchains"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/results"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"go.uber.org/zap"
)

// PrimaryServer provides the listening server to communicate with the secondaries
// as well as a connection to the active secondaries.
type PrimaryServer struct {
	Listener            net.Listener // TCP listener listening for incoming secondaries
	Secondaries         []net.Conn   // Any connected secondaries so that they can communicate with the Primary
	ExpectedSecondaries int          // The number of expected secondaries to connect
}

// SecondaryReplyErrors stores the errors returned by the secondaries to be printed out
type SecondaryReplyErrors []string

// SetupPrimaryTCP generates a new "Listener" by creating the TCP server.
func SetupPrimaryTCP(addr string, expectedSecondaries int) (*PrimaryServer, error) {
	listener, err := net.Listen("tcp", addr)

	// If we can't make a listener, we
	// should fail gracefully but immediately.
	if err != nil {
		return nil, err
	}

	zap.L().Info("Server Started",
		zap.String("Addr", addr),
		zap.Int("Expected Secondaries", expectedSecondaries))

	return &PrimaryServer{Listener: listener, ExpectedSecondaries: expectedSecondaries}, nil
}

// HandleSecondaries starts a listener that will run in a thread to
// handle any secondary connections.
func (s *PrimaryServer) HandleSecondaries(readyChannel chan bool) {

	for {
		c, err := s.Listener.Accept()

		if err != nil {
			// Log the error here
			zap.L().Error("Error from listen",
				zap.Error(err))
		}

		zap.L().Info(fmt.Sprintf("Secondary %d / %d connected", len(s.Secondaries), s.ExpectedSecondaries),
			zap.String("Addr:", c.RemoteAddr().String()))

		s.Secondaries = append(s.Secondaries, c)

		if len(s.Secondaries) == s.ExpectedSecondaries {
			readyChannel <- true
			break
		}
	}
}

// sendAndWaitOKAsync is used to send and wait for the OK byte to be
// received. This takes a channel and replies on the channel once OK or err is received.
func (s *PrimaryServer) sendAndWaitOKAsync(data []byte, secondary net.Conn, doneCh chan int, errCh chan error) {
	if _, err := secondary.Write(data); err != nil {
		// TODO: Log that we can't communicate with secondary
		errCh <- err
		doneCh <- 1
	}

	reply := make([]byte, 1)

	_, err := secondary.Read(reply)

	if err != nil {
		errCh <- err
		doneCh <- 1
	}

	zap.L().Debug(fmt.Sprintf("GOT REPLY FROM %s\n", secondary.RemoteAddr().String()))

	// If we got an error reply - it means
	// something failed on the secondary machine
	if bytes.Equal(MsgErr, reply) {
		// TODO: Add a "get X bytes for the error reason"
		errCh <- fmt.Errorf("failed to communicate with secondary %s", secondary.RemoteAddr().String())
		doneCh <- 1
	}

	doneCh <- 0
	return
}

// SendAndWaitOKSync send a message to a secondary and wait for the okay without
// the use of a channel (synchronous sending).
func (s *PrimaryServer) SendAndWaitOKSync(data []byte, secondary net.Conn) error {
	if _, err := secondary.Write(data); err != nil {
		// TODO: Log that we can't communicate with secondary
		return &SecondaryCommError{
			SecondaryInfo: secondary.RemoteAddr().String(),
			Err:           err,
		}
	}

	reply := make([]byte, 1024)

	n, err := secondary.Read(reply)

	if err != nil {
		// TODO: Log secondary got an error
		return &SecondaryCommError{
			SecondaryInfo: secondary.RemoteAddr().String(),
			Err:           err,
		}
	}

	zap.L().Debug(fmt.Sprintf("GOT REPLY FROM %s\n", secondary.RemoteAddr().String()))

	// If we got an error reply - it means
	// something failed on the secondary machine
	if reply[0] == MsgErr[0] {
		// TODO: Add a "get X bytes for the error reason"
		return &SecondaryErrorReply{
			Info: secondary.RemoteAddr().String(),
			Err:  fmt.Errorf("error from secondary %s", string(reply[1:n])),
		}
	}

	return nil
}

// sendAndWaitData sends a message to a secondary and waits for the OK and data, or errors
func (s *PrimaryServer) sendAndWaitData(data []byte, secondary net.Conn) ([]results.Results, error) {
	if _, err := secondary.Write(data); err != nil {
		// TODO log that we can't communicate with secondary
		return nil, &SecondaryCommError{
			SecondaryInfo: secondary.RemoteAddr().String(),
			Err:           err,
		}
	}

	// Read the reply AND response error (if it's an error, 1024 is to
	// encapsulate any error string passed with the data).
	initialReply := make([]byte, 512)

	n, err := secondary.Read(initialReply)

	zap.L().Debug("Got secondary reply from RES message")

	if err != nil {
		// TODO: Log secondary got an error
		return nil, &SecondaryCommError{
			SecondaryInfo: secondary.RemoteAddr().String(),
			Err:           err,
		}
	}

	// If we got an error reply - it means
	// something failed on the secondary machine
	if initialReply[0] == MsgErr[0] {
		// TODO: Add a "get X bytes for the error reason"
		return nil, &SecondaryErrorReply{
			Info: secondary.RemoteAddr().String(),
			Err:  fmt.Errorf("error from secondary %s", string(initialReply[1:n])),
		}
	}

	// Now we have to read through the data until we end.
	// Get the length
	dataLen := binary.BigEndian.Uint64(initialReply[1:9])

	if dataLen == 0 {
		return []results.Results{results.Results{
			AverageLatency: 0,
			Throughput:     0,
			TxLatencies:    []float64{},
		}}, nil
	}

	fullReply := initialReply[9:]
	zap.L().Debug("Reply from client",
		zap.Uint64("length", dataLen),
		zap.ByteString("reply", fullReply),
	)
	buffer := make([]byte, 1024)
	readLen := n - 9

	for {
		if uint64(readLen) >= dataLen {
			break
		}
		n, err := secondary.Read(buffer)
		zap.L().Debug(fmt.Sprintf("Secondary %s read %d, total %d", secondary.RemoteAddr().String(), n, readLen))
		if err != nil {
			if err != io.EOF {
				return nil, &SecondaryCommError{
					SecondaryInfo: secondary.RemoteAddr().String(),
					Err:           err,
				}
			}
			break
		}

		fullReply = append(fullReply, buffer[:n]...)
		readLen += n
	}

	zap.L().Debug("Read secondary reply",
		zap.String("secondary", secondary.RemoteAddr().String()),
		zap.Int("numbytes", readLen))

	var res []results.Results
	err = json.Unmarshal(fullReply[:dataLen], &res)

	if err != nil {
		zap.L().Error("failed to unmarshal bytes of result reply from secondary",
			zap.Error(err))
		return nil, err
	}
	return res, nil
}

// PrepareBenchmarkSecondaries sends the prepare message to the secondaires
func (s *PrimaryServer) PrepareBenchmarkSecondaries(numThreads uint32) SecondaryReplyErrors {

	var errorList []string

	threadBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(threadBytes, numThreads)

	for i, c := range s.Secondaries {
		secondaryID := make([]byte, 4)
		binary.BigEndian.PutUint32(secondaryID, uint32(i))
		payload := append(MsgPrepare, secondaryID...)
		payload = append(payload, threadBytes...)
		err := s.SendAndWaitOKSync(payload, c)
		if err != nil {
			zap.L().Warn("Got an error from secondary",
				zap.String("secondary", c.RemoteAddr().String()))
			errorList = append(errorList, err.Error())
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	return errorList
}

// SendBlockchainType [UNUSED/TODO] sends the blockchain type to the secondaries
// Note: this function was intended to let the same servers be used for multiple
// benchmarks of different chains without having to restart and spawn new secondaries.
// Moved to future work.
func (s *PrimaryServer) SendBlockchainType(bcType blockchains.BlockchainTypeMessage) SecondaryReplyErrors {
	// Send the blockchain type message
	var errorList []string

	fullMessage := append(MsgBc, byte(bcType))
	for _, c := range s.Secondaries {
		err := s.SendAndWaitOKSync(fullMessage, c)
		if err != nil {
			zap.L().Warn("error from secondary",
				zap.String("secondary", c.RemoteAddr().String()))
			errorList = append(errorList, err.Error())
		}
	}

	if len(errorList) == 0 {
		return nil
	}

	return errorList
}

// SendWorkload sends the workload to all secondaries. Encodes the workload in the
// chosen encoding in helpers.go and will send off the bytes to be read and processed
// by the secondary.
func (s *PrimaryServer) SendWorkload(workloads workloadgenerators.Workload) SecondaryReplyErrors {
	var errorList []error

	for i, c := range s.Secondaries {
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

// RunBenchmark sends the message to all secondaries to run the benchmark.
func (s *PrimaryServer) RunBenchmark() SecondaryReplyErrors {
	zap.L().Info("\n------------\nStarting Benchmark\n------------\n")

	var errorList SecondaryReplyErrors

	// Channels for goroutine comms
	okCh := make(chan int, len(s.Secondaries))
	errCh := make(chan error, len(s.Secondaries))

	for _, c := range s.Secondaries {
		go s.sendAndWaitOKAsync(MsgRun, c, okCh, errCh)
	}

	numberDone := 0
	numberOfErrors := 0
	for {
		select {
		case secondaryDone := <-okCh:
			zap.L().Debug("Secondary Done")
			numberDone++
			numberOfErrors += secondaryDone
			if numberDone == len(s.Secondaries) {
				break
			}
		}
		if numberDone == len(s.Secondaries) {
			break
		}
	}

	var errList SecondaryReplyErrors
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

// GetResults calls the secondaries to return the results.
// Will return the list of results as well as any errors that had been encountered
func (s *PrimaryServer) GetResults() ([][]results.Results, SecondaryReplyErrors) {
	var allResults [][]results.Results
	var errs SecondaryReplyErrors

	for _, c := range s.Secondaries {
		// Send the RES command, wait for the results to come back
		secondaryRes, err := s.sendAndWaitData(MsgResults, c)

		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		zap.L().Debug(fmt.Sprintf("Got %d results from secondary", len(secondaryRes)))

		allResults = append(allResults, secondaryRes)
	}

	zap.L().Debug(fmt.Sprintf("%d Results returned", len(allResults)))
	return allResults, errs
}

// SendFin sends the final GOODBYE message and then close the connection to the secondaries
func (s *PrimaryServer) SendFin() {
	for _, c := range s.Secondaries {
		_ = s.SendAndWaitOKSync(MsgFin, c)
	}
}

// CloseAll close all connections and threads
func (s *PrimaryServer) CloseAll() {
	s.CloseSecondaries()
	s.CloseAll()
}

// CloseSecondaries closes the secondary connections
func (s *PrimaryServer) CloseSecondaries() {
	for i, c := range s.Secondaries {
		zap.L().Debug(fmt.Sprintf("Closing Secondary %d @ %s", i, c.RemoteAddr().String()))
		_ = c.Close()
	}
}

// Close the listener and exit
func (s *PrimaryServer) Close() {
	_ = s.Listener.Close()
}
