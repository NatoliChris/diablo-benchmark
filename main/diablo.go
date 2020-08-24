package main

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core"
	"diablo-benchmark/core/configs/parsers"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func printWelcome(isPrimary bool) {
	fmt.Println("=====================")
	fmt.Println("  Welcome to Diablo  ")
	if isPrimary {
		fmt.Println("    PRIMARY SERVER")
	} else {
		fmt.Println("    SECONDARY CLIENT")
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

// Run the primary functions
func runPrimary(primaryArgs *core.PrimaryArgs) {
	// Check the arguments
	primaryArgs.CheckArgs()

	zap.L().Info("loading configs",
		zap.String("bench config", primaryArgs.BenchConfigPath),
		zap.String("chain config", primaryArgs.ChainConfigPath),
	)

	// Parse the configurations.
	bConfig, err := parsers.ParseBenchConfig(primaryArgs.BenchConfigPath)

	if err != nil {
		zap.L().Error(err.Error())
		os.Exit(1)
	}

	cConfig, err := parsers.ParseChainConfig(primaryArgs.ChainConfigPath)

	if err != nil {
		os.Exit(1)
	}

	generatorClass, err := workloadgenerators.GetWorkloadGenerator(cConfig)

	if err != nil {
		zap.L().Error("failed to get workload generators",
			zap.String("error", err.Error()))
		os.Exit(1)
	}

	wg := generatorClass.NewGenerator(cConfig, bConfig)

	// Initialise the TCP server
	m := core.InitPrimary(primaryArgs.ListenAddr, bConfig.Clients, wg, bConfig, cConfig)

	// Run the benchmark flow
	m.Run()
}

// Run the worker
func runWorker(workerArgs *core.WorkerArgs) {
	workerArgs.WorkerArgs()

	chainConfiguration, err := parsers.ParseChainConfig(workerArgs.ChainConfigPath)

	if err != nil {
		zap.L().Error("failed to parse config",
			zap.Error(err))
		os.Exit(1)
	}

	worker, err := core.NewWorker(chainConfiguration, workerArgs.PrimaryAddr)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	worker.Run()
}

// Main running function
func main() {
	prepareLogger()

	args := core.DefineArguments()

	if len(os.Args) < 2 {
		// This is going to be a primary
		zap.L().Warn("No subcommand given, running as primary!")
		args.PrimaryCommand.Parse(os.Args[1:])
		runPrimary(args.PrimaryArgs)
	} else {
		switch os.Args[1] {
		case "primary":
			// Print the welcome message
			printWelcome(true)

			// Parse the arguments
			args.PrimaryCommand.Parse(os.Args[2:])

			runPrimary(args.PrimaryArgs)

		case "worker":
			// Print the welcome message
			printWelcome(false)

			// Parse the arguments
			err := args.WorkerCommand.Parse(os.Args[2:])
			if err != nil {
				zap.L().Error("error parsing",
					zap.Error(err))
				os.Exit(1)
			}
			runWorker(args.WorkerArgs)
		}
	}
}
