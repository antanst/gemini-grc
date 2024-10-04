package main

import (
	"fmt"
)

type GeminiUrl struct {
	protocol string
	hostname string
	port     int
	path     string
	descr    string
}

func (self GeminiUrl) String() string {
	return fmt.Sprintf("%s://%s:%d%s", self.protocol, self.hostname, self.port, self.path)
}

type Result struct {
	url   GeminiUrl
	data  string
	links []GeminiUrl
	code  int
	error error
}
