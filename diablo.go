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
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/parsers"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
func prepareLogger(logType string, level zapcore.Level) {
	// config := zap.NewDevelopmentConfig()
	// config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// logger, err := config.Build()

	// Set up the console log - so we can see the colours
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)
	atomicLevel := zap.NewAtomicLevel()

	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), atomicLevel)
	atomicLevel.SetLevel(level)

	// Set up the file-logger
	fileConfig := zap.NewProductionEncoderConfig()
	fileConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create the file, add the sync
	file, err := os.Create(fmt.Sprintf("%s_diablo.log", logType))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file")
		os.Exit(1)
	}
	fileSync := zapcore.AddSync(file)
	fileEncoder := zapcore.NewJSONEncoder(fileConfig)
	fileLevel := zap.NewAtomicLevel()
	fileCore := zapcore.NewCore(fileEncoder, fileSync, fileLevel)

	// Set up both the loggers
	cores := zapcore.NewTee(
		consoleCore,
		fileCore,
	)

	logger := zap.New(cores)
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
		zap.L().Error(err.Error())
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
	zap.L().Info("Primary ready, running benchmark flow")
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

	benchConfiguration, err := parsers.ParseBenchConfig(secondaryArgs.BenchConfigPath)

	// Check the timeout with args
	if secondaryArgs.Timeout == 0 && benchConfiguration.Timeout <= 0 {
		zap.L().Warn(fmt.Sprintf("Invalid or no timeout provided, defaulting to %d", configs.DefaultTimeout))
		benchConfiguration.Timeout = configs.DefaultTimeout
	} else if secondaryArgs.Timeout > 0 {
		zap.L().Warn(fmt.Sprintf("Overwriting config timeout (%d) with flag %d", benchConfiguration.Timeout, secondaryArgs.Timeout))
		benchConfiguration.Timeout = secondaryArgs.Timeout
	}

	secondary, err := core.NewSecondary(chainConfiguration, benchConfiguration, secondaryArgs.PrimaryAddr)

	if err != nil {
		zap.L().Error("Failed to start new secondary",
			zap.Error(err))
		os.Exit(1)
	}

	secondary.Run()
}

// Main running function
func main() {
	args := core.DefineArguments()

	if len(os.Args) < 2 {
		// This is going to be a primary
		fmt.Fprintf(os.Stderr, "No subcommand given (primary/secondary), exiting!")
		os.Exit(1)
	} else {
		switch os.Args[1] {
		case "primary":
			// Print the welcome message
			printWelcome(true)

			// Parse the arguments
			args.PrimaryCommand.Parse(os.Args[2:])

			prepareLogger("primary", args.PrimaryArgs.LogLevel)
			runPrimary(args.PrimaryArgs)

		case "secondary":
			// Print the welcome message
			printWelcome(false)

			// Parse the arguments
			err := args.SecondaryCommand.Parse(os.Args[2:])

			prepareLogger("secondary", args.SecondaryArgs.LogLevel)
			if err != nil {
				zap.L().Error("error parsing",
					zap.Error(err))
				os.Exit(1)
			}
			runSecondary(args.SecondaryArgs)
		}
	}
}
