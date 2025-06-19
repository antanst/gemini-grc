package common

import (
	"os"
	"sync"
)

// FatalErrorsChan accepts errors from workers.
// In case of fatal error, gracefully
// exits the application.
var (
	FatalErrorsChan chan error
	SignalsChan     chan os.Signal
	WorkerWG        sync.WaitGroup
)

const VERSION string = "0.0.1"

const CtxKeyLogger string = "CtxKeyLogger"
