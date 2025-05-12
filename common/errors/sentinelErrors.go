package commonErrors

import "fmt"

var (
	ErrBlacklistMatch = fmt.Errorf("black list match")
	ErrRobotsMatch    = fmt.Errorf("robots match")
)
