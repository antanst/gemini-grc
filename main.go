package main

import (
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	config := *getConfig()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(config.logLevel)
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "[2006-01-02 15:04:05]"})
	if err := runApp(&config); err != nil {
		LogError("Application error: %w", err)
		os.Exit(1)
	}
}

func runApp(config *Config) error {
	// urls := []string{"gemini://smol.gr"}
	urls := []string{"gemini://gmi.noulin.net/", "gemini://warmedal.se/~antenna/"}

	queue := make(chan string, 10000)
	results := make(chan Snapshot, 100)
	done := make(chan struct{})

	go spawnStats(queue, results)
	go resultsHandler(queue, results)
	spawnWorkers(config, queue, results)

	for _, url := range urls {
		queue <- url
	}
	<-done
	return nil
}

func spawnStats(queue chan string, results chan Snapshot) {
	ticker := time.NewTicker(time.Duration(time.Second * 10))
	defer ticker.Stop()
	for range ticker.C {
		LogInfo("Queue   length: %d\n", len(queue))
		LogInfo("Results length: %d\n", len(results))
	}
}

func spawnWorkers(config *Config, queue <-chan string, results chan Snapshot) {
	workers := config.numOfWorkers
	LogInfo("Spawning %d workers", workers)
	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go func(i int) {
			worker(i, config.rootPath, queue, results)
		}(i)
	}
}

func resultsHandler(queue chan string, results <-chan Snapshot) {
	for result := range results {
		if result.Error != nil {
			LogError("[%s] %w", result.Url, result.Error)
		} else {
			LogDebug("[%s] Done", result.Url)
			for _, link := range result.Links {
				if strings.HasPrefix(link.Full, "gemini://") {
					go func(link GeminiUrl) {
						queue <- link.Full
						// fmt.Printf("Sent %s to queue\n", link.Full)
					}(link)
				}
			}
			// fmt.Printf(SnapshotToJSON(result))
		}
	}
}

func worker(id int, rootPath string, queue <-chan string, results chan Snapshot) {
	for url := range queue {
		LogDebug("Worker %d visiting %s", id, url)
		result := Visit(url)
		// If we encountered an error when
		// visiting, skip processing
		if result.Error != nil {
			results <- *result
			continue
		}
		LogDebug("Worker %d processing %s", id, url)
		result = Process(result)
		if result.Error != nil {
			results <- *result
			continue
		}
		LogDebug("Worker %d saving %s", id, url)
		SaveResult(rootPath, result)
		results <- *result
		time.Sleep(time.Duration(rand.IntN(5)) * time.Second)
	}
}
