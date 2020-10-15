package core

import (
	"flag"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Arguments provides all the argument sets
type Arguments struct {
	PrimaryCommand   *flag.FlagSet  // Commands related to the primary
	SecondaryCommand *flag.FlagSet  // Commands related to the secondarys
	PrimaryArgs      *PrimaryArgs   // Primary arguments
	SecondaryArgs    *SecondaryArgs // Secondary arguments
}

// PrimaryArgs contains the command-line arguments for the primary
type PrimaryArgs struct {
	BenchConfigPath string        // Path to the configurations
	ChainConfigPath string        // Path to the chain configuration
	ListenAddr      string        // host:port that it should run on
	LogLevel        zapcore.Level // log level
}

// SecondaryArgs provides command-line arguments for secondary
type SecondaryArgs struct {
	ConfigPath      string        // Path to the secondary config
	ChainConfigPath string        // Path to the blockchain configuration
	PrimaryAddr     string        // Address of the primary (can also be in secondary config)
	LogLevel        zapcore.Level // log level
}

// DefineArguments sets the arguments that will be used for the subcommands
func DefineArguments() *Arguments {

	primaryCommand := flag.NewFlagSet("primary", flag.ExitOnError)
	secondaryCommand := flag.NewFlagSet("secondary", flag.ExitOnError)

	primaryArgs := PrimaryArgs{}
	secondaryArgs := SecondaryArgs{}

	// General arguments
	// --config

	primaryCommand.StringVar(&primaryArgs.BenchConfigPath, "config", "", "--config=/path/to/config (required)")
	primaryCommand.StringVar(&primaryArgs.BenchConfigPath, "c", "", "-c /path/to/config")
	secondaryCommand.StringVar(&secondaryArgs.ConfigPath, "config", "", "--config=/path/to/config (required)")
	secondaryCommand.StringVar(&secondaryArgs.ConfigPath, "c", "", "-c /path/to/config")

	// --level
	primaryArgs.LogLevel = zapcore.InfoLevel
	primaryCommand.Var(&primaryArgs.LogLevel, "level", "--level INFO|WARN|DEBUG|ERROR")
	secondaryArgs.LogLevel = zapcore.InfoLevel
	secondaryCommand.Var(&secondaryArgs.LogLevel, "level", "--level INFO|WARN|DEBUG|ERROR")

	// Primary Arguments
	primaryCommand.StringVar(&primaryArgs.ListenAddr, "addr", "", "--addr=addr (e.g. --addr=\"0.0.0.0:8323\")")
	primaryCommand.StringVar(&primaryArgs.ListenAddr, "a", "", "-a addr (e.g. -a \":8323\")")

	primaryCommand.StringVar(&primaryArgs.ChainConfigPath, "chain-config", "", "--chain-config=/path/to/chain/yml (required)")
	primaryCommand.StringVar(&primaryArgs.ChainConfigPath, "cc", "", "-cc /path/to/chain/yml")

	// Secondary Arguments
	secondaryCommand.StringVar(&secondaryArgs.PrimaryAddr, "primary", "", "--primary=<ipaddr>:<port>")
	secondaryCommand.StringVar(&secondaryArgs.PrimaryAddr, "m", "", "-m <ipaddress>:<port>")

	secondaryCommand.StringVar(&secondaryArgs.ChainConfigPath, "chain-config", "", "--chain-config=/path/to/chain/yml (required)")
	secondaryCommand.StringVar(&secondaryArgs.ChainConfigPath, "cc", "", "-cc /path/to/chain/yml")

	// Return all the arguments
	return &Arguments{
		PrimaryCommand:   primaryCommand,   // The primary command FlagSet
		SecondaryCommand: secondaryCommand, // The secondary command FlagSet
		PrimaryArgs:      &primaryArgs,     // The primary argument list, contains config and other args
		SecondaryArgs:    &secondaryArgs,   // The secondary argument list, contains config and other args
	}
}

// CheckArgs checks that the primary arguments conform to specified requirements
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

// SecondaryArgs validates that the secondary arguments are correct
func (sa *SecondaryArgs) SecondaryArgs() {
	// We must have at least one - either the primary address or the config
	if sa.ConfigPath == "" && sa.PrimaryAddr == "" {
		zap.L().Error("primary information not provided")
		os.Exit(0)
	}

	if sa.ChainConfigPath == "" {
		zap.L().Error("no chain config provided")
		os.Exit(1)
	}
}
