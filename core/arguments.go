package core

import (
	"flag"
	"go.uber.org/zap"
	"os"
)

// All the available arguments
type Arguments struct {
	MasterCommand *flag.FlagSet // Commands related to the master
	WorkerCommand *flag.FlagSet // Commands related to the workers
	MasterArgs    *MasterArgs   // Master arguments
	WorkerArgs    *WorkerArgs   // Worker arguments
}

// Arguments for the paster
type MasterArgs struct {
	BenchConfigPath string // Path to the configurations
	ChainConfigPath string // Path to the chain configuration
	ListenAddr      string // host:port that it should run on
}

// Worker arguments
type WorkerArgs struct {
	ConfigPath      string // Path to the worker config
	ChainConfigPath string // Path to the blockchain configuration
	MasterAddr      string // Address of the master (can also be in worker config)
}

// Initialise the arguments
func DefineArguments() *Arguments {

	masterCommand := flag.NewFlagSet("master", flag.ExitOnError)
	workerCommand := flag.NewFlagSet("worker", flag.ExitOnError)

	masterArgs := MasterArgs{}
	workerArgs := WorkerArgs{}

	// General arguments
	// --config

	masterCommand.StringVar(&masterArgs.BenchConfigPath, "config", "", "--config=/path/to/config (required)")
	masterCommand.StringVar(&masterArgs.BenchConfigPath, "c", "", "-c /path/to/config")
	workerCommand.StringVar(&workerArgs.ConfigPath, "config", "", "--config=/path/to/config (required)")
	workerCommand.StringVar(&workerArgs.ConfigPath, "c", "", "-c /path/to/config")

	// Master Arguments
	masterCommand.StringVar(&masterArgs.ListenAddr, "addr", "", "--addr=addr (e.g. --addr=\"0.0.0.0:8323\")")
	masterCommand.StringVar(&masterArgs.ListenAddr, "a", "", "-a addr (e.g. -a \":8323\")")

	masterCommand.StringVar(&masterArgs.ChainConfigPath, "chain-config", "", "--chain-config=/path/to/chain/yml (required)")
	masterCommand.StringVar(&masterArgs.ChainConfigPath, "cc", "", "-cc /path/to/chain/yml")

	// Worker Arguments
	workerCommand.StringVar(&workerArgs.MasterAddr, "master", "", "--master=<ipaddr>:<port>")
	workerCommand.StringVar(&workerArgs.MasterAddr, "m", "", "-m <ipaddress>:<port>")

	workerCommand.StringVar(&workerArgs.ChainConfigPath, "chain-config", "", "--chain-config=/path/to/chain/yml (required)")
	workerCommand.StringVar(&workerArgs.ChainConfigPath, "cc", "", "-cc /path/to/chain/yml")

	// Return all the arguments
	return &Arguments{
		MasterCommand: masterCommand, // The master command FlagSet
		WorkerCommand: workerCommand, // The worker command FlagSet
		MasterArgs:    &masterArgs,   // The master argument list, contains config and other args
		WorkerArgs:    &workerArgs,   // The worker argument list, contains config and other args
	}
}

// Check the master arguments conform to specified requirements
func (ma *MasterArgs) CheckArgs() {
	if ma.BenchConfigPath == "" {
		zap.L().Error("benchmark config not provided")
		os.Exit(0)
	}

	if ma.ChainConfigPath == "" {
		zap.L().Error("chain configuration not provided")
		os.Exit(0)
	}
}

// Checks that the worker arguments are correct
func (wa *WorkerArgs) WorkerArgs() {
	// We must have at least one - either the master address or the config
	if wa.ConfigPath == "" && wa.MasterAddr == "" {
		zap.L().Error("master information not provided")
		os.Exit(0)
	}

	if wa.ChainConfigPath == "" {
		zap.L().Error("no chain config provided")
		os.Exit(1)
	}
}
