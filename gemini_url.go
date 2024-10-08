package main

import (
	"encoding/json"
)

type GeminiUrl struct {
	Protocol string `json:"protocol,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     int    `json:"port,omitempty"`
	Path     string `json:"path,omitempty"`
	Descr    string `json:"descr,omitempty"`
	Full     string `json:"full,omitempty"`
}

func (u GeminiUrl) String() string {
	return u.Full
	//	return fmt.Sprintf("%s://%s:%d%s", u.Protocol, u.Hostname, u.Port, u.Path)
}

func GeminiUrltoJSON(g GeminiUrl) string {
	// Serialize the Person struct to JSON
	jsonData, err := json.Marshal(g)
	if err != nil {
		LogError("Error serializing to JSON: %w", err)
	}
	return string(jsonData)
}

func GeminiUrlFromJSON(input string) GeminiUrl {
	var geminiUrl GeminiUrl
	err := json.Unmarshal([]byte(input), &geminiUrl)
	if err != nil {
		LogError("Error deserializing from JSON: %w", err)
	}
	return geminiUrl
}
