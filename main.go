package main

import (
	"os"
	"sync"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if err := runApp(); err != nil {
		LogError("Application error: %v", err)
		os.Exit(1)
	}
}

func runApp() error {
	urls := []string{"gemini://smol.gr"} //, "gemini://gmi.noulin.neta/", "gemini://in.gr:443"}

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
	results := make(chan Result)
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
	// close it, and the ResultsProcessor can
	// finish
	close(results)
	<-resultsDone

	close(done)
}

func resultsHandler(results <-chan Result, done chan struct{}) {
	for result := range results {
		if result.error != nil {
			LogError("%w", result.error)
		} else {
			LogInfo("[%s] Done. Result: %#v", result.url, result)
		}
	}
	LogInfo("All results have been processed")
	close(done)
}

func worker(queue <-chan string, results chan Result) {
	for url := range queue {
		result := Visit(url)
		// If we encountered an error when
		// visiting, skip processing
		if result.error != nil {
			results <- *result
			continue
		}
		result = Process(result)
		results <- *result
	}
}
