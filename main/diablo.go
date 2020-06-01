package main

import (
	"diablo-benchmark/communication"
	"diablo-benchmark/core"
	"flag"
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

func main() {

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()

	if err != nil {
		fmt.Errorf("Failed to produce a logger: %s", err.Error())
		os.Exit(1)
	}

	masterCommand := flag.NewFlagSet("master", flag.ExitOnError)
	workerCommand := flag.NewFlagSet("worker", flag.ExitOnError)

	logger.Info("Welcome to Diablo")

	if len(os.Args) < 2 {
		// This is going to be a master
		fmt.Println("Hello, I am the master.")
		masterCommand.Parse(os.Args[1:])
	} else {
		switch os.Args[1] {
		case "master":
			// Print the welcome message
			printWelcome(true)

			// Parse the arguments
			masterCommand.Parse(os.Args[2:])
			m := core.InitMaster()
			m.Run()

		case "worker":
			// Print the welcome message
			printWelcome(false)

			// Parse the arguments
			workerCommand.Parse(os.Args[2:])
			s, err := communication.SetupClientTCP("localhost:8123")
			if err != nil {
				panic(err)
			}
			s.HandleCommands()
		}
	}
}
