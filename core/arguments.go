package core

import (
	"flag"
	"go.uber.org/zap"
	"os"
)

// All the available arguments
type Arguments struct {
	PrimaryCommand *flag.FlagSet // Commands related to the primary
	WorkerCommand  *flag.FlagSet // Commands related to the workers
	PrimaryArgs    *PrimaryArgs  // Primary arguments
	WorkerArgs     *WorkerArgs   // Worker arguments
}

// Arguments for the paster
type PrimaryArgs struct {
	BenchConfigPath string // Path to the configurations
	ChainConfigPath string // Path to the chain configuration
	ListenAddr      string // host:port that it should run on
}

// Worker arguments
type WorkerArgs struct {
	ConfigPath      string // Path to the worker config
	ChainConfigPath string // Path to the blockchain configuration
	PrimaryAddr     string // Address of the primary (can also be in worker config)
}

// Initialise the arguments
func DefineArguments() *Arguments {

	primaryCommand := flag.NewFlagSet("primary", flag.ExitOnError)
	workerCommand := flag.NewFlagSet("worker", flag.ExitOnError)

	primaryArgs := PrimaryArgs{}
	workerArgs := WorkerArgs{}

	// General arguments
	// --config

	primaryCommand.StringVar(&primaryArgs.BenchConfigPath, "config", "", "--config=/path/to/config (required)")
	primaryCommand.StringVar(&primaryArgs.BenchConfigPath, "c", "", "-c /path/to/config")
	workerCommand.StringVar(&workerArgs.ConfigPath, "config", "", "--config=/path/to/config (required)")
	workerCommand.StringVar(&workerArgs.ConfigPath, "c", "", "-c /path/to/config")

	// Primary Arguments
	primaryCommand.StringVar(&primaryArgs.ListenAddr, "addr", "", "--addr=addr (e.g. --addr=\"0.0.0.0:8323\")")
	primaryCommand.StringVar(&primaryArgs.ListenAddr, "a", "", "-a addr (e.g. -a \":8323\")")

	primaryCommand.StringVar(&primaryArgs.ChainConfigPath, "chain-config", "", "--chain-config=/path/to/chain/yml (required)")
	primaryCommand.StringVar(&primaryArgs.ChainConfigPath, "cc", "", "-cc /path/to/chain/yml")

	// Worker Arguments
	workerCommand.StringVar(&workerArgs.PrimaryAddr, "primary", "", "--primary=<ipaddr>:<port>")
	workerCommand.StringVar(&workerArgs.PrimaryAddr, "m", "", "-m <ipaddress>:<port>")

	workerCommand.StringVar(&workerArgs.ChainConfigPath, "chain-config", "", "--chain-config=/path/to/chain/yml (required)")
	workerCommand.StringVar(&workerArgs.ChainConfigPath, "cc", "", "-cc /path/to/chain/yml")

	// Return all the arguments
	return &Arguments{
		PrimaryCommand: primaryCommand, // The primary command FlagSet
		WorkerCommand:  workerCommand,  // The worker command FlagSet
		PrimaryArgs:    &primaryArgs,   // The primary argument list, contains config and other args
		WorkerArgs:     &workerArgs,    // The worker argument list, contains config and other args
	}
}

// Check the primary arguments conform to specified requirements
func (pa *PrimaryArgs) CheckArgs() {
	if pa.BenchConfigPath == "" {
		zap.L().Error("benchmark config not provided")
		os.Exit(0)
	}

	if pa.ChainConfigPath == "" {
		zap.L().Error("chain configuration not provided")
		os.Exit(0)
	}
}

// Checks that the worker arguments are correct
func (wa *WorkerArgs) WorkerArgs() {
	// We must have at least one - either the primary address or the config
	if wa.ConfigPath == "" && wa.PrimaryAddr == "" {
		zap.L().Error("primary information not provided")
		os.Exit(0)
	}

	if wa.ChainConfigPath == "" {
		zap.L().Error("no chain config provided")
		os.Exit(1)
	}
}
