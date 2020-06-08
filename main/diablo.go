package main

import (
	"diablo-benchmark/communication"
	"diablo-benchmark/core"
	"diablo-benchmark/core/configs/parsers"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func printWelcome(isMaster bool) {
	fmt.Println("=====================")
	fmt.Println("  Welcome to Diablo  ")
	if isMaster {
		fmt.Println("    MASTER SERVER")
	} else {
		fmt.Println("    CLIENT WORKER")
	}
	fmt.Println("=====================")
}

func prepareLogger() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()

	if err != nil {
		_ = fmt.Errorf("failed to produce a logger: %s", err.Error())
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)
}

// Run the master functions
func runMaster(masterArgs *core.MasterArgs) {
	// Check the arguments
	masterArgs.CheckArgs()

	zap.L().Info("loading configs",
		zap.String("bench config", masterArgs.BenchConfigPath),
		zap.String("chain config", masterArgs.ChainConfigPath),
	)

	// Parse the configurations.
	bConfig, err := parsers.ParseBenchConfig(masterArgs.BenchConfigPath)

	if err != nil {
		zap.L().Error(err.Error())
		os.Exit(1)
	}

	// Initialise the TPC server
	m := core.InitMaster(masterArgs.ListenAddr, bConfig.Workers)

	// Run the benchmark flow
	m.Run()
}

// Run the worker
func runWorker(workerArgs *core.WorkerArgs) {
	s, err := communication.SetupClientTCP(workerArgs.MasterAddr)
	if err != nil {
		panic(err)
	}
	s.HandleCommands()
}

// Main running function
func main() {
	prepareLogger()

	args := core.DefineArguments()

	if len(os.Args) < 2 {
		// This is going to be a master
		zap.L().Warn("No subcommand given, running as master!")
		args.MasterCommand.Parse(os.Args[1:])
		runMaster(args.MasterArgs)
	} else {
		switch os.Args[1] {
		case "master":
			// Print the welcome message
			printWelcome(true)

			// Parse the arguments
			args.MasterCommand.Parse(os.Args[2:])

			runMaster(args.MasterArgs)

		case "worker":
			// Print the welcome message
			printWelcome(false)

			// Parse the arguments
			args.WorkerCommand.Parse(os.Args[2:])

			runWorker(args.WorkerArgs)
		}
	}
}
