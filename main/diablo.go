package main

import (
	"diablo-benchmark/communication"
	"diablo-benchmark/core"
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
		fmt.Errorf("Failed to produce a logger: %s", err.Error())
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)
}

func main() {
	prepareLogger()

	args := core.DefineArguments()

	if len(os.Args) < 2 {
		// This is going to be a master
		zap.L().Info("No subcommand given, running as master!")
		args.MasterCommand.Parse(os.Args[1:])
	} else {
		switch os.Args[1] {
		case "master":
			// Print the welcome message
			printWelcome(true)

			// Parse the arguments
			args.MasterCommand.Parse(os.Args[2:])

			fmt.Println("HELLO")
			fmt.Println(args.MasterArgs.ConfigPath)

			args.MasterArgs.CheckArgs()

			m := core.InitMaster()
			m.Run()

		case "worker":
			// Print the welcome message
			printWelcome(false)

			// Parse the arguments
			args.WorkerCommand.Parse(os.Args[2:])
			s, err := communication.SetupClientTCP("localhost:8123")
			if err != nil {
				panic(err)
			}
			s.HandleCommands()
		}
	}
}
