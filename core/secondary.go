package core

import (
	"diablo-benchmark/blockchains"
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/communication"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/handlers"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
)

type Worker struct {
	ID              int                                  // This worker's unique ID
	ChainConfig     *configs.ChainConfig                 // Chain configuration
	Blockchain      clientinterfaces.BlockchainInterface // Blockchain Interface
	PrimaryComms    *communication.ConnClient            // Connection to the primary
	WorkloadHandler *handlers.WorkloadHandler            // Workload Handler
}

// Create a new worker, set up the things we need.
func NewWorker(config *configs.ChainConfig, primaryAddress string) (*Worker, error) {
	// Set up the communication
	c, err := communication.SetupClientTCP(primaryAddress)
	if err != nil {
		zap.L().Error("failed to connect to primary server")
		return nil, err
	}

	// Log and return, ready to go!
	zap.L().Info("Client init")
	return &Worker{
		ChainConfig:  config,
		PrimaryComms: c,
	}, nil
}

// Runs the main worker things, sets up the client and waits for the commands
func (w *Worker) Run() {
	// Main work loop that handles the commands from primary and dispatches
	// the workload from the benchmark.
	for {

		cmd, err := w.PrimaryComms.InitialRead()

		if err != nil {
			zap.L().Warn("failed to read",
				zap.String("err", err.Error()))

			w.PrimaryComms.CloseConn()
			return
		}

		switch cmd[0] {
		case communication.MsgPrepare[0]:
			// Prepare message, did we connect, and are we prepared for work?
			zap.L().Info("Got command from primary",
				zap.String("CMD", "PREPARE"))
			// It should also give us a client ID as aux value
			w.ID = int(cmd[1])
			w.ID = int(binary.BigEndian.Uint32(cmd[1:5]))
			numThreads := binary.BigEndian.Uint32(cmd[5:9])
			// Connect le blockchains
			var bcis []clientinterfaces.BlockchainInterface
			for i := uint32(0); i < numThreads; i++ {
				bc, err := clientinterfaces.GetBlockchainInterface(w.ChainConfig)
				if err != nil {
					w.PrimaryComms.ReplyERR(err.Error())
				}
				bcis = append(bcis, bc)
			}

			// Create the workload handler
			wHandler := handlers.NewWorkloadHandler(
				numThreads,
				bcis,
			)

			w.WorkloadHandler = wHandler

			err := w.WorkloadHandler.Connect(w.ChainConfig.Nodes, w.ID)
			if err != nil {
				w.PrimaryComms.ReplyERR(err.Error())
				continue
			}
		case communication.MsgBc[0]:
			// What type of blockchain are we running?
			// NOTE: see blockchains/bctypemessage.go for details about why feature
			// is not used (for now).
			zap.L().Info("Got command from primary",
				zap.String("CMD", "BLOCKCHAIN"))
			_, err = blockchains.MatchMessageToInterface(cmd[1])
			if err != nil {
				w.PrimaryComms.ReplyERR(err.Error())
				continue
			}
		case communication.MsgWorkload[0]:
			zap.L().Info("Got command from primary",
				zap.String("CMD", "WORKLOAD"))

			// Workload length = bytes 1-4
			workloadLen := binary.BigEndian.Uint64(cmd[1:])
			wl, err := w.PrimaryComms.ReadSize(workloadLen)

			if err != nil {
				zap.L().Warn("failed to read workload bytes",
					zap.String("err", err.Error()))
				w.PrimaryComms.ReplyERR(err.Error())
				continue
			}

			var unmarshaledWorkload workloadgenerators.ClientWorkload
			err = json.Unmarshal(wl, &unmarshaledWorkload)

			if err != nil {
				zap.L().Warn("failed to unmarshal workload",
					zap.String("err", err.Error()))
				w.PrimaryComms.ReplyERR(err.Error())
				continue
			}

			err = w.WorkloadHandler.ParseWorkloads(unmarshaledWorkload)
			if err != nil {
				zap.L().Warn("failed to parse workload",
					zap.String("err", err.Error()))
				w.PrimaryComms.ReplyERR(err.Error())
				continue
			}

		case communication.MsgRun[0]:
			zap.L().Info("Got command from primary",
				zap.String("CMD", "RUN"))
			errs := w.WorkloadHandler.RunBench()
			if errs != nil {
				zap.L().Warn("error during bench",
					zap.Error(err))
				w.PrimaryComms.ReplyERR(err.Error())
				continue
			}
		case communication.MsgResults[0]:
			zap.L().Info("Got command from primary",
				zap.String("CMD", "RESULTS"))
			res := w.WorkloadHandler.HandleCleanup()
			resBytes, err := json.Marshal(res)
			if err != nil {
				w.PrimaryComms.ReplyERR("failed to convert results to bytes")
			}
			fmt.Println(resBytes)
			w.PrimaryComms.SendDataOK(resBytes)
		case communication.MsgFin[0]:
			zap.L().Info("Got command from primary",
				zap.String("CMD", "FIN"))
			w.WorkloadHandler.CloseAll()
			return
		default:
			w.PrimaryComms.ReplyERR("no matching command")
			continue
		}

		w.PrimaryComms.ReplyOK()
	}

}
