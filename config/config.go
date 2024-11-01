package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type Config struct {
	LogLevel        zerolog.Level
	rootPath        string
	MaxResponseSize int
	NumOfWorkers    int
	ResponseTimeout int
	WorkerBatchSize int
}

var CONFIG Config

func GetConfig() *Config {
	var config Config
	for _, envVar := range []string{
		"LOG_LEVEL",
		"ROOT_PATH",
		"NUM_OF_WORKERS",
		"WORKER_BATCH_SIZE",
		"MAX_RESPONSE_SIZE",
		"RESPONSE_TIMEOUT",
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
					config.LogLevel = logLevel
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
						config.NumOfWorkers = numOfWorkers
					}
				}
			case "WORKER_BATCH_SIZE":
				{
					if workerBatchSize, err := strconv.Atoi(env); err != nil {
						fmt.Fprintf(os.Stderr, "Invalid WORKER_BATCH_SIZE value\n")
						os.Exit(1)
					} else {
						config.WorkerBatchSize = workerBatchSize
					}
				}
			case "MAX_RESPONSE_SIZE":
				{
					if maxResponseSize, err := strconv.Atoi(env); err != nil {
						fmt.Fprintf(os.Stderr, "Invalid MAX_RESPONSE_SIZE value\n")
						os.Exit(1)
					} else {
						config.MaxResponseSize = maxResponseSize
					}
				}
			case "RESPONSE_TIMEOUT":
				{
					if val, err := strconv.Atoi(env); err != nil {
						fmt.Fprintf(os.Stderr, "Invalid RESPONSE_TIMEOUT value\n")
						os.Exit(1)
					} else {
						config.ResponseTimeout = val
					}
				}
			}
		}
	}
	return &config
}
