package core

import (
	"diablo-benchmark/blockchains"
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/communication"
	"diablo-benchmark/core/configs"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
)

type Worker struct {
	ID          int                                  // This worker's unique ID
	ChainConfig *configs.ChainConfig                 // Chain configuration
	Blockchain  clientinterfaces.BlockchainInterface // Blockchain Interface
	MasterComms *communication.ConnClient            // Connection to the master
	Workload    []interface{}                        // The workload after it has been parsed
}

// Create a new worker, set up the things we need.
func NewWorker(config *configs.ChainConfig, masterAddress string) (*Worker, error) {
	// First make a new worker
	w := Worker{ChainConfig: config}

	// Set up the communication
	c, err := communication.SetupClientTCP(masterAddress)
	if err != nil {
		zap.L().Error("failed to connect to master server")
		return nil, err
	}
	w.MasterComms = c
	bc, err := clientinterfaces.GetBlockchainInterface(config)
	if err != nil {
		return nil, err
	}
	w.Blockchain = bc

	zap.L().Info("Client init")

	return &w, nil
}

// Runs the main worker things, sets up the client and waits for the commands
func (w *Worker) Run() {
	// Main work loop that handles the commands from master and dispatches
	// the workload from the benchmark.
	for {

		cmd, err := w.MasterComms.InitialRead()

		if err != nil {
			zap.L().Warn("failed to read",
				zap.String("err", err.Error()))

			w.MasterComms.CloseConn()
			return
		}

		switch cmd[0] {
		case communication.MsgPrepare[0]:
			// Prepare message, did we connect, and are we prepared for work?
			zap.L().Info("Got command from master",
				zap.String("CMD", "PREPARE"))
			// It should also give us a client ID as aux value
			w.ID = int(cmd[1])
			w.Blockchain.Init(w.ChainConfig.Nodes)
			err := w.Blockchain.ConnectAll(w.ID)
			if err != nil {
				w.MasterComms.ReplyERR(err.Error())
				continue
			}
		case communication.MsgBc[0]:
			// What type of blockchain are we running?
			// NOTE: see blockchains/bctypemessage.go for details about why feature
			// is not used (for now).
			zap.L().Info("Got command from master",
				zap.String("CMD", "BLOCKCHAIN"))
			_, err = blockchains.MatchMessageToInterface(cmd[1])
			if err != nil {
				w.MasterComms.ReplyERR(err.Error())
				continue
			}
		case communication.MsgWorkload[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "WORKLOAD"))

			// Workload length = bytes 1-4
			workloadLen := binary.BigEndian.Uint32(cmd[1:])
			fmt.Println("Got cmd")
			fmt.Println(cmd)
			fmt.Println("length: ", workloadLen)

			wl, err := w.MasterComms.ReadSize(workloadLen)

			if err != nil {
				zap.L().Warn("failed to read workload bytes",
					zap.String("err", err.Error()))
				w.MasterComms.ReplyERR(err.Error())
				continue
			}

			var unmarshaledWorkload [][]byte
			err = json.Unmarshal(wl, &unmarshaledWorkload)

			if err != nil {
				zap.L().Warn("failed to unmarshal workload",
					zap.String("err", err.Error()))
				w.MasterComms.ReplyERR(err.Error())
				continue
			}

			parsedWL, err := w.Blockchain.ParseWorkload(unmarshaledWorkload)

			if err != nil {
				zap.L().Warn("failed to parse workload",
					zap.String("err", err.Error()))
				w.MasterComms.ReplyERR(err.Error())
				continue
			}

			w.Workload = parsedWL
		case communication.MsgRun[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "RUN"))
			errs := w.runBench()
			if errs != nil {
				zap.L().Warn("error during bench",
					zap.Errors("err", errs))
				w.MasterComms.ReplyERR(err.Error())
				continue
			}
		case communication.MsgResults[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "RESULTS"))
		case communication.MsgFin[0]:
			zap.L().Info("Got command from master",
				zap.String("CMD", "FIN"))
			return
		default:
			w.MasterComms.ReplyERR("no matching command")
			continue
		}

		w.MasterComms.ReplyOK()
	}

}

func (w *Worker) runBench() []error {
	var errs []error
	for i := 0; i < len(w.Workload); i++ {
		err := w.Blockchain.SendRawTransaction(w.Workload[i])
		errs = append(errs, err)
	}

	//tNow := time.Now()
	//for {
	//	if w.Blockchain.NumTxDone == uint64(len(w.Workload)) {
	//		break
	//	}
	//	if time.Now().Sub(tNow) > 10*time.Second {
	//		break
	//	}
	//
	//	fmt.Printf("Sent: %d, Complete: %d\n", w.Blockchain.NumTxSent, w.Blockchain.NumTxDone)
	//	time.Sleep(1000 * time.Millisecond)
	//}
	//
	//res := w.Blockchain.Cleanup()

	return nil
}
