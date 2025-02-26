package common

var (
	StatusChan chan WorkerStatus
	// ErrorsChan accepts errors from workers.
	// In case of fatal error, gracefully
	// exits the application.
	ErrorsChan chan error
)

const VERSION string = "0.0.1"

const CtxKeyLogger string = "CtxKeyLogger"
