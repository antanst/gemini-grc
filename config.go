package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type Config struct {
	logLevel        zerolog.Level
	rootPath        string
	numOfWorkers    int
	maxResponseSize int
}

func getConfig() *Config {
	var config Config
	for _, envVar := range []string{
		"LOG_LEVEL",
		"ROOT_PATH",
		"NUM_OF_WORKERS",
		"MAX_RESPONSE_SIZE",
	} {
		if env, ok := os.LookupEnv(envVar); !ok {
			fmt.Fprintf(os.Stderr, "Missing env var %s\n", envVar)
			os.Exit(1)
		} else {
			switch envVar {
			case "LOG_LEVEL":
				{
					logLevel, err := zerolog.ParseLevel(env)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Invalid LOG_LEVEL value\n")
						os.Exit(1)
					}
					config.logLevel = logLevel
				}
			case "ROOT_PATH":
				{
					config.rootPath = env
				}
			case "NUM_OF_WORKERS":
				{
					if numOfWorkers, err := strconv.Atoi(env); err != nil {
						fmt.Fprintf(os.Stderr, "Invalid NUM_OF_WORKERS value\n")
						os.Exit(1)
					} else {
						config.numOfWorkers = numOfWorkers
					}
				}
			case "MAX_RESPONSE_SIZE":
				{
					if maxResponseSize, err := strconv.Atoi(env); err != nil {
						fmt.Fprintf(os.Stderr, "Invalid MAX_RESPONSE_SIZE value\n")
						os.Exit(1)
					} else {
						config.maxResponseSize = maxResponseSize
					}
				}
			}
		}
	}
	return &config
}
