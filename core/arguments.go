package core

import (
	"flag"
	"go.uber.org/zap"
	"os"
)

type Arguments struct {
	MasterCommand *flag.FlagSet
	WorkerCommand *flag.FlagSet
	MasterArgs    *MasterArgs
	WorkerArgs    *WorkerArgs
}

type MasterArgs struct {
	ConfigPath string
	Port       int
}

type WorkerArgs struct {
	ConfigPath string
}

// Initialise the arguments
func DefineArguments() *Arguments {

	masterCommand := flag.NewFlagSet("master", flag.ExitOnError)
	workerCommand := flag.NewFlagSet("worker", flag.ExitOnError)

	masterArgs := MasterArgs{}
	workerArgs := WorkerArgs{}

	// General arguments
	// --config

	masterCommand.StringVar(&masterArgs.ConfigPath, "config", "", "--config=/path/to/config (required)")
	masterCommand.StringVar(&masterArgs.ConfigPath, "c", "", "-c /path/to/config")
	workerCommand.StringVar(&workerArgs.ConfigPath, "config", "", "--config=/path/to/config (required)")
	workerCommand.StringVar(&workerArgs.ConfigPath, "c", "", "-c /path/to/config")

	// Master Arguments
	masterCommand.IntVar(&masterArgs.Port, "port", 0, "--port=portnumber (e.g. --port=34226)")
	masterCommand.IntVar(&masterArgs.Port, "p", 0, "-p portnumber (e.g. --p 34226)")

	// Worker Arguments

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
	if ma.ConfigPath == "" {
		zap.L().Error("config not provided")
		os.Exit(0)
	}
}

func (wa *WorkerArgs) WorkerArgs() {

}
