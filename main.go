package main

import (
	"os"
	"sync"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

const ROOTPATH string = "./a"

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	//zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "[2006-01-02 15:04:05]"})
	if err := runApp(); err != nil {
		LogError("Application error: %w", err)
		os.Exit(1)
	}
}

func runApp() error {
	//urls := []string{"gemini://smol.gr"}
	urls := []string{"gemini://smol.gr", "gemini://gmi.noulin.net/"}

	queue := make(chan string)
	done := make(chan struct{})

	// Start the crawler.
	go crawler(queue, done)

	// Send URLs to the queue
	for _, url := range urls {
		// Send URL to queue; blocks until crawler receives it
		queue <- url
	}

	// All URLs have been sent and received
	// because queue is unbuffered; safe to close the queue
	close(queue)

	// Wait until crawler signals finish
	<-done
	return nil
}

func crawler(queue <-chan string, done chan struct{}) {
	// Start processing results.
	results := make(chan Snapshot)
	resultsDone := make(chan struct{})
	go resultsHandler(results, resultsDone)

	// Create workers that consume the queue channel,
	// and send their result to results channel.
	workers := 3
	LogInfo("Spawning %d workers", workers)
	var wg sync.WaitGroup
	// Start worker goroutines
	for range workers {
		wg.Add(1)
		go func() {
			worker(queue, results)
			wg.Done()
		}()
	}

	// Wait until all workers have finished.
	wg.Wait()
	LogInfo("All workers have finished")

	// Nobody left to send to results, so we
	// close it, and the SnapshotsProcessor can
	// finish
	close(results)
	<-resultsDone

	close(done)
}

func resultsHandler(results <-chan Snapshot, done chan struct{}) {
	for result := range results {
		if result.Error != nil {
			LogError("[%s] %w", result.Url, result.Error)
		} else {
			LogInfo("[%s] Done", result.Url)
			// fmt.Printf(SnapshotToJSON(result))
		}
	}
	LogInfo("All results have been processed")
	close(done)
}

func worker(queue <-chan string, results chan Snapshot) {
	for url := range queue {
		result := Visit(url)
		// If we encountered an error when
		// visiting, skip processing
		if result.Error != nil {
			results <- *result
			continue
		}
		result = Process(result)
		if result.Error != nil {
			results <- *result
			continue
		}
		SaveResult(ROOTPATH, result)
		results <- *result
	}
}
