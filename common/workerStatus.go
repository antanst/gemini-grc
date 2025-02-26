package common

import (
	"fmt"
	"strings"

	"gemini-grc/config"
)

type WorkerStatus struct {
	ID     int
	Status string
}

func UpdateWorkerStatus(workerID int, status string) {
	if !config.GetConfig().PrintWorkerStatus {
		return
	}
	if config.CONFIG.NumOfWorkers > 1 {
		StatusChan <- WorkerStatus{
			ID:     workerID,
			Status: status,
		}
	}
}

func PrintWorkerStatus(totalWorkers int, statusChan chan WorkerStatus) {
	if !config.GetConfig().PrintWorkerStatus {
		return
	}

	// Create a slice to store current Status of each worker
	statuses := make([]string, totalWorkers)

	// Initialize empty statuses
	for i := range statuses {
		statuses[i] = ""
	}

	// Initial print
	var output strings.Builder
	// \033[H moves the cursor to the top left corner of the screen
	// (ie, the first column of the first row in the screen).
	// \033[J clears the part of the screen from the cursor to the end of the screen.
	output.WriteString("\033[H\033[J") // Clear screen and move cursor to top
	for i := range statuses {
		output.WriteString(fmt.Sprintf("[%2d] \n", i))
	}
	fmt.Print(output.String())

	// Continuously receive Status updates
	for update := range statusChan {
		if update.ID >= totalWorkers {
			continue
		}

		// Update the Status
		statuses[update.ID] = update.Status

		// Build the complete output string
		output.Reset()
		output.WriteString("\033[H\033[J") // Clear screen and move cursor to top
		for i, status := range statuses {
			output.WriteString(fmt.Sprintf("[%2d] %.100s\n", i, status))
		}

		// Print the entire Status
		fmt.Print(output.String())
	}
}
