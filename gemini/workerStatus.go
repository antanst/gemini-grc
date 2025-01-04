package gemini

import (
	"fmt"
	"strings"
)

type WorkerStatus struct {
	id     int
	status string
}

var statusChan chan WorkerStatus

func PrintWorkerStatus(totalWorkers int, statusChan chan WorkerStatus) {
	// Create a slice to store current status of each worker
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

	// Continuously receive status updates
	for update := range statusChan {
		if update.id >= totalWorkers {
			continue
		}

		// Update the status
		statuses[update.id] = update.status

		// Build the complete output string
		output.Reset()
		output.WriteString("\033[H\033[J") // Clear screen and move cursor to top
		for i, status := range statuses {
			output.WriteString(fmt.Sprintf("[%2d] %.100s\n", i, status))
		}

		// Print the entire status
		fmt.Print(output.String())
	}
}
