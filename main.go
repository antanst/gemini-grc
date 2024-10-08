package main

import (
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

var CONFIG Config

func main() {
	CONFIG = *getConfig()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(CONFIG.logLevel)
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "[2006-01-02 15:04:05]"})
	if err := runApp(); err != nil {
		LogError("Application error: %w", err)
		os.Exit(1)
	}
}

func runApp() error {
	// urls := []string{"gemini://smol.gr"}
	urls := []string{"gemini://gmi.noulin.net/", "gemini://warmedal.se/~antenna/"}

	queue := make(chan string, 1000)
	results := make(chan Snapshot, 100)
	done := make(chan struct{})

	go spawnStats(queue, results)
	go resultsHandler(queue, results)
	spawnWorkers(CONFIG.numOfWorkers, queue, results)

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
		LogInfo("Queue   length: %d", len(queue))
		LogInfo("Results length: %d", len(results))
	}
}

func spawnWorkers(numOfWorkers int, queue <-chan string, results chan Snapshot) {
	LogInfo("Spawning %d workers", numOfWorkers)
	// Start worker goroutines
	for i := 0; i < numOfWorkers; i++ {
		go func(i int) {
			worker(i, queue, results)
		}(i)
	}
}

func resultsHandler(queue chan string, results <-chan Snapshot) {
	for result := range results {
		if result.Error != nil {
			LogError("[%s] %w", result.URL, result.Error)
		} else {
			LogDebug("[%s] Done", result.URL)
			for _, link := range result.Links {
				if strings.HasPrefix(link.Full, "gemini://") {
					go func(link GeminiUrl) {
						queue <- link.Full
						// fmt.Printf("Sent %s to queue\n", link.Full)
					}(link)
				}
			}
			// if result.MimeType == "text/gemini" {
			// 	result.Data = ""
			// 	fmt.Printf(SnapshotToJSON(result))
			// }
		}
	}
}

func worker(id int, queue <-chan string, results chan Snapshot) {
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
		result = ProcessHeaders(result)
		if result.Error != nil {
			results <- *result
			continue
		}
		if result.MimeType == "text/gemini" {
			result = ProcessGemini(result)
		}
		if shouldPersist(result) {
			LogInfo("Worker %d saving %s", id, url)
			SaveResult(CONFIG.rootPath, result)
		}
		results <- *result
		time.Sleep(time.Duration(rand.IntN(5)) * time.Second)
	}
}

func shouldPersist(result *Snapshot) bool {
	if result.MimeType == "text/gemini" ||
		strings.HasPrefix(result.MimeType, "image/") ||
		strings.HasPrefix(result.MimeType, "text/") {
		return true
	}
	return false
}
