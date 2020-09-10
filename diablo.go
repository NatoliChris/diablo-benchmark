// DIABLO provides a unique blockchain benchmark framework focusing on analysis
// of real-world workloads with distributed clients from the core.
// The diablo framework provides modular blockchain implementation as well as
// simple workload definition and design, aiming to maximise the applicability
// and integration of as many systems as possible.
//
// About the architecture
//
// The main aspects of the Diablo benchmark is the "primary" and "secondary"
// benchmark nodes. "Primary" is the main node that orchestrates the benchmark
// and sends commands to the distributed clients to start the benchmark. It is
// the main generator of the workload and contains all the information required.
// The "Secondary" is used to accept commands from the master, connect to the
// blockchain node and then send the transactions executing the benchmark while
// measuring information. There should always be ONE primary and ONE OR MORE
// secondaries. The number of secondaries are defined in the benchmark
// configuration.
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

// Prints the welcome message that is seen at the start

func printWelcome(isPrimary bool) {
	fmt.Println("=====================")
	fmt.Println(
		"\n" +
			"████████▄   ▄█     ▄████████ ▀█████████▄   ▄█        ▄██████▄  \n" +
			"██   ▀███ ███    ███    ███   ███    ███ ███       ███    ███ \n" +
			"██    ███ ███▌   ███    ███   ███    ███ ███       ███    ███ \n" +
			"██    ███ ███▌   ███    ███  ▄███▄▄▄██▀  ███       ███    ███ \n" +
			"██    ███ ███▌ ▀███████████ ▀▀███▀▀▀██▄  ███       ███    ███ \n" +
			"██    ███ ███    ███    ███   ███    ██▄ ███       ███    ███ \n" +
			"██   ▄███ ███    ███    ███   ███    ███ ███▌    ▄ ███    ███ \n" +
			"███████▀  █▀     ███    █▀  ▄█████████▀  █████▄▄██  ▀██████▀  \n" +
			"                                          ▀",
	)
	if isPrimary {
		fmt.Println(
			"\n" +
				"______       _                                \n" +
				"| ___ \\     (_)                               \n" +
				"| |_/ /_ __  _  _ __ ___    __ _  _ __  _   _ \n" +
				"|  __/| '__|| || '_ ` _ \\  / _` || '__|| | | |\n" +
				"| |   | |   | || | | | | || (_| || |   | |_| |\n" +
				"\\_|   |_|   |_||_| |_| |_| \\__,_||_|    \\__, |\n" +
				"                                         __/ |\n" +
				"                                        |___/ ",
		)
	} else {
		fmt.Println("\n" +
			" _____                               _                     \n" +
			"/  ___|                             | |                    \n" +
			"\\ `--.   ___   ___  ___   _ __    __| |  __ _  _ __  _   _ \n" +
			" `--. \\ / _ \\ / __|/ _ \\ | '_ \\  / _` | / _` || '__|| | | |\n" +
			"/\\__/ /|  __/| (__| (_) || | | || (_| || (_| || |   | |_| |\n" +
			"\\____/  \\___| \\___|\\___/ |_| |_| \\__,_| \\__,_||_|    \\__, |\n" +
			"                                                      __/ |\n" +
			"                                                     |___/",
		)
	}
	fmt.Println("=====================")
}

// Prepares the logger
// TODO make a flag to change level of the logger (DEBUG, INFO, ..)
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
	m := core.InitPrimary(primaryArgs.ListenAddr, bConfig.Secondaries, wg, bConfig, cConfig)

	// Run the benchmark flow
	m.Run()
}

// Run the secondary
func runSecondary(secondaryArgs *core.SecondaryArgs) {
	secondaryArgs.SecondaryArgs()

	chainConfiguration, err := parsers.ParseChainConfig(secondaryArgs.ChainConfigPath)

	if err != nil {
		zap.L().Error("failed to parse config",
			zap.Error(err))
		os.Exit(1)
	}

	secondary, err := core.NewSecondary(chainConfiguration, secondaryArgs.PrimaryAddr)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	secondary.Run()
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

		case "secondary":
			// Print the welcome message
			printWelcome(false)

			// Parse the arguments
			err := args.SecondaryCommand.Parse(os.Args[2:])
			if err != nil {
				zap.L().Error("error parsing",
					zap.Error(err))
				os.Exit(1)
			}
			runSecondary(args.SecondaryArgs)
		}
	}
}
