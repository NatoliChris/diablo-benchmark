package clientinterfaces

import (
	"diablo-benchmark/blockchains/types"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// DiemInterface the Diem implementation of the clientinterface
// provide the means to communicate with the Diem blockchain
type DiemInterface struct {
	senderRefId	uint64							// reference id of the sender account which sends all transactions
	resultReceiver net.Listener 				// TCP Server to receive commit result (success or fail)
	commandSender *net.TCPAddr  				// TCP Client to send transaction
	throughputCommandSender *net.TCPAddr 		// TCP Client to send command to throughput server
	commitChannel chan *types.DiemCommitEvent 	// channel where we continuously listen to commit events to register throughput


	TransactionInfo  map[uint64][]time.Time // Transaction information (used for throughput calculation)
	StartTime        time.Time              // Start time of the benchmark
	ThroughputTicker *time.Ticker           // Ticker for throughput (1s)
	Throughputs      []float64              // Throughput over time with 1 second intervals
	GenericInterface
}

/**
	Initialise tcp client to query rust client
	Initialise tcp server to
 */
func (f *DiemInterface) Init(chainConfig *configs.ChainConfig) {
	f.Nodes = chainConfig.Nodes
	f.NumTxDone = 0
	f.TransactionInfo = make(map[uint64][]time.Time, 0)

	mapConfig := chainConfig.Extra[0].(map[string]interface{})
	// Configure result server
	urlResultServer := mapConfig["tcpServerAddress"].(string)
	l, err := net.Listen("tcp", urlResultServer)
	if err != nil {
		println("Fail to start a server")
		panic(err)
	}
	f.resultReceiver = l
	// Configure command sender Client
	tcpAddr, err := net.ResolveTCPAddr("tcp", mapConfig["tcpClientAddress"].(string))
	if err != nil {
		println("Address resolve failed")
		panic(err)
	}
	f.commandSender = tcpAddr

	throughputTcpAddr, err := net.ResolveTCPAddr("tcp", mapConfig["throughputServer"].(string))
	if err != nil {
		println("Address resolve failed")
		panic(err)
	}
	f.throughputCommandSender = throughputTcpAddr

}


// Invoke command on rust client to create actual signed transaction with sequence number for execution later
func (f *DiemInterface) createSignedTransactions(t *types.DiemTX) error {
	//senderAddress := f.accounts[t.SenderRefId]
	command := "d mt "+ strconv.FormatUint(t.SenderRefId, 10) +" "+ strconv.FormatUint(t.SequenceNumber, 10) +" " + t.ScriptPath
	for _, arg := range t.Args{
		command = command + " " + arg
	}

	conn, err := net.DialTCP("tcp", nil,f.commandSender)
	if err != nil{
		println("Failed to create connection createSignedTransaction")
		return err
	}
	defer conn.Close()
	_, err = conn.Write([]byte(command))
	if err != nil {
		return err
	}
	//f.accounts[t.SenderRefId].SequenceNumber = senderAddress.SequenceNumber+1
	return nil
}

func (f *DiemInterface) Cleanup() results.Results {
	// Stop the ticker
	f.ThroughputTicker.Stop()

	txLatencies := make([]float64, 0)
	var avgLatency float64

	var endTime time.Time

	for _, v := range f.TransactionInfo {
		if len(v) > 1 {
			txLatency := v[1].Sub(v[0]).Milliseconds()
			txLatencies = append(txLatencies, float64(txLatency))
			avgLatency += float64(txLatency)
			if v[1].After(endTime) {
				endTime = v[1]
			}
		}
	}

	success := uint(f.Success)
	fails := uint(f.Fail)

	zap.L().Debug("Statistics being returned",
		zap.Uint("success", success),
		zap.Uint("fail", fails))

	var throughput float64

	if len(txLatencies) > 0 {
		throughput = float64(f.NumTxDone) - float64(f.Fail)/(endTime.Sub(f.StartTime).Seconds())
		avgLatency = avgLatency / float64(len(txLatencies))
	} else {
		avgLatency = 0
		throughput = 0
	}

	averageThroughput := float64(0)
	var calculatedThroughputSeconds = []float64{f.Throughputs[0]}
	for i := 1; i < len(f.Throughputs); i++ {
		calculatedThroughputSeconds = append(calculatedThroughputSeconds, float64(f.Throughputs[i]-f.Throughputs[i-1]))
		averageThroughput += float64(f.Throughputs[i] - f.Throughputs[i-1])
	}

	averageThroughput = averageThroughput / float64(len(f.Throughputs))

	zap.L().Debug("Results being returned",
		zap.Float64("avg throughput", averageThroughput),
		zap.Float64("throughput (as is)", throughput),
		zap.Float64("latency", avgLatency),
		zap.String("ThroughputWindow", fmt.Sprintf("%v", calculatedThroughputSeconds)),
	)

	return results.Results{
		TxLatencies:       txLatencies,
		AverageLatency:    avgLatency,
		Throughput:        averageThroughput,
		ThroughputSeconds: calculatedThroughputSeconds,
		Success:           success,
		Fail:              fails,
	}
}

// throughputSeconds calculates the throughput over time, to show dynamic
func (f *DiemInterface) throughputSeconds() {
	f.ThroughputTicker = time.NewTicker(time.Duration(f.Window) * time.Second)
	seconds := float64(0)
	for {
		select {
		case <-f.ThroughputTicker.C:
			seconds += float64(f.Window)
			f.Throughputs = append(f.Throughputs, float64(f.NumTxDone-f.Fail))
		}
	}
}
// TODO
func (f *DiemInterface) getThroughput()  {
	// start the sequence counting
	go func() {
		conn, err := net.DialTCP("tcp", nil,f.throughputCommandSender)
		if err != nil{
			println("Failed to create connection listenForCommits")
			return
		}
		_,err = conn.Write([]byte("diablo connect "+ f.resultReceiver.Addr().String()))

		if err != nil {
			println("rust client unable to connect to diablo ResultReceiver")
			return
		}
		conn.Close()
		conn, err = net.DialTCP("tcp", nil,f.throughputCommandSender)
		if err != nil{
			println("Failed to create connection listenForCommits")
			return
		}

		defer conn.Close()
		_, err = conn.Write([]byte("d gsn "+ strconv.FormatUint(f.senderRefId, 10))) //TODO
		if err != nil {
			println("rust client unable to carry out command to get Sequence Number")
			return
		}
	}()

	c, err := f.resultReceiver.Accept()

	if err != nil {
		fmt.Println(err)
	}

	for{
		buffer := make([]byte, 1024)
		length, err := c.Read(buffer)
		if err != nil {
			return
		}
		result := string(buffer[:length])
		seqNum, _ := strconv.ParseUint(result, 10, 64)
		f.NumTxDone = seqNum
	}
}
func (f *DiemInterface) listenForCommits() {
	//go f.getThroughput()
	conn, err := net.DialTCP("tcp", nil,f.throughputCommandSender)
	if err != nil{
		println("Failed to create connection listenForCommits")
		return
	}
	_,err = conn.Write([]byte("diablo connect "+ f.resultReceiver.Addr().String()))

	if err != nil {
		println("rust client unable to connect to diablo ResultReceiver")
		return
	}
	conn.Close()


	c, err := f.resultReceiver.Accept()

	if err != nil {
		fmt.Println(err)
	}

	var counter = uint64(0)
	for {
		if int(counter) >= len(f.TransactionInfo) {break}

		conn, err = net.DialTCP("tcp", nil,f.throughputCommandSender)
		if err != nil{
			println("Failed to create connection listenForCommits")
			return
		}

		defer conn.Close()
		_, err = conn.Write([]byte("d gt "+ strconv.FormatUint(f.senderRefId, 10)+" "+strconv.FormatUint(counter, 10)+" "+"false"))
		if err != nil {
			println("rust client unable to carry out command to get DONE/NOT_DONE info")
			return
		}

		buffer := make([]byte, 1024)
		length, err := c.Read(buffer)
		if err != nil {
			return
		}
		result := string(buffer[:length])
		if result == "DONE"{
			f.TransactionInfo[counter] = append(f.TransactionInfo[counter], time.Now())
			counter++
			atomic.AddUint64(&f.Success, 1)
			atomic.AddUint64(&f.NumTxDone, 1)
		}else if result == "NOT_DONE"{
			continue
		}
		//select {
		//case commit := <-f.commitChannel:
		//
		//	ID := commit.ID
		//	zap.L().Debug("CommitChannel",
		//		zap.Uint64("ID", ID))
		//	// transaction failed, incrementing number of done and failed transactions
		//	if !commit.Valid {
		//		atomic.AddUint64(&f.Fail, 1)
		//	} else {
		//		//transaction validated, making the note of the time of return
		//		f.TransactionInfo[ID] = append(f.TransactionInfo[ID], commit.CommitTime)
		//		atomic.AddUint64(&f.Success, 1)
		//	}
		//	//atomic.AddUint64(&f.NumTxDone, 1)
		//}

	}
}

func (f *DiemInterface) Start() {
	f.StartTime = time.Now()
	go f.throughputSeconds()
	go f.listenForCommits()
}

func (f *DiemInterface) ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error) {
	// Thread workload = list of transactions in intervals
	// [interval][tx] = [][][]byte
	parsedWorkload := make([][]interface{}, 0)

	for _, v := range workload {
		intervalTxs := make([]interface{}, 0)
		for _, txBytes := range v {
			var t types.DiemTX
			err := json.Unmarshal(txBytes, &t)
			if err != nil {
				return nil, err
			}
			f.senderRefId = t.SenderRefId
			intervalTxs = append(intervalTxs, &t)
		}
		parsedWorkload = append(parsedWorkload, intervalTxs)
	}

	f.TotalTx = len(parsedWorkload)
	// the commitChannel buffer length should be the total number of transactions so that it's not a blocker
	f.commitChannel = make(chan *types.DiemCommitEvent, f.TotalTx)

	return parsedWorkload, nil
}

// TODO connect to first node in the list, currently connect to server in Init
func (f *DiemInterface) ConnectOne(id int) error {
	return nil
}
// TODO
func (f *DiemInterface) ConnectAll(primaryID int) error {
	return nil
}

func (f *DiemInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	return nil, nil
}
func getTimeFromString(str string) time.Time {
	timeMillis, _ := strconv.ParseUint(str,10, 64)
	return time.Unix(int64(timeMillis/1000), int64(timeMillis%1000*1_000_000))
}


func (f *DiemInterface) SendRawTransaction(tx interface{}) error {
	t := tx.(*types.DiemTX)
	zap.L().Debug("Submitting TX",
		zap.Uint64("ID", t.ID))

	// making note of the time we send the transaction
	//f.TransactionInfo[transaction.ID] = []time.Time{time.Now()}
	atomic.AddUint64(&f.NumTxSent, 1)
	go func() {
		//var reply int
		conn, err := net.DialTCP("tcp", nil,f.commandSender)
		if err != nil{
			println("Failed to create connection SendRawTransaction")
			return
		}
		defer conn.Close()
		command := "dev execute " + strconv.FormatUint(t.SenderRefId, 10)
		if t.FunctionType == "throughput"{
			command = "d men " + strconv.FormatUint(t.SenderRefId, 10) +" "+ strconv.FormatUint(t.SequenceNumber, 10)
		}
		command = command + " " + t.ScriptPath
		for _, arg := range t.Args{
			command = command + " " + arg
		}
		println(command)
		_, err = conn.Write([]byte(command))

		if err != nil {
			zap.L().Debug("TX got an error WRITE",
				zap.Error(err))
		}
		reply := make([]byte, 1024)
		replyLenth, err := conn.Read(reply)
		if err != nil {
			zap.L().Debug("TX got an error READ",
				zap.Error(err))
		}
		replyInfo := strings.Split(string(reply[:replyLenth]), "|")
		f.TransactionInfo[t.ID] = []time.Time{getTimeFromString(replyInfo[0])}
		//responseTime := getTimeFromString(replyInfo[1])
		//valid := err == nil
		//commit := types.DiemCommitEvent{
		//	Valid:      valid,
		//	ID:         t.ID,
		//	CommitTime: responseTime,
		//}
		//f.commitChannel <- &commit
	}()
	return nil
}

// SecureRead reads the value from the chain
// (NOT NEEDED IN FABRIC) SecureRead is useful in permissionless blockchains where transaction
// validation is not always clear but transactions are always clearly rejected or commited in Hyperledger Fabric
func (f *DiemInterface) SecureRead(callFunc string, callParams []byte) (interface{}, error) {
	return nil, nil
}

// GetBlockByNumber retrieves the block information at the given index
func (f *DiemInterface) GetBlockByNumber(index uint64) (GenericBlock, error) {
	return GenericBlock{
		Hash:              "",
		Index:             0,
		Timestamp:         0,
		TransactionNumber: 0,
		TransactionHashes: nil,
	}, nil
}

// GetBlockHeight returns the current height of the chain
func (f *DiemInterface) GetBlockHeight() (uint64, error) {
	return 0, nil
}

// ParseBlocksForTransactions retrieves block information from start to end index and
// is used as a post-benchmark check to learn about the block and transactions.
// (NOT NEEDED IN FABRIC) As transactions are confirmed to be validated whenever we submit a transaction
func (f *DiemInterface) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	return nil
}

// Close the connection to the blockchain node
func (f *DiemInterface) Close() {
	f.resultReceiver.Close()
	close(f.commitChannel)
}