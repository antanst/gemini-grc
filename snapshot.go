package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Snapshot struct {
	Url       GeminiUrl   `json:"url,omitempty"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
	Data      string      `json:"data,omitempty"`
	Links     []GeminiUrl `json:"links,omitempty"`
	Code      int         `json:"code,omitempty"`
	Error     error       `json:"error,omitempty"`
	UID       string      `json:"uid,omitempty"`
}

func (u Snapshot) String() string {
	return fmt.Sprintf(
		"[%s] %s %s %s %d %s",
		u.UID, u.Url, u.Timestamp, u.Links, u.Code, u.Error,
	)
}

func SnapshotToJSON(g Snapshot) string {
	// Serialize the Person struct to JSON
	jsonData, err := json.Marshal(g)
	if err != nil {
		LogError("Error serializing to JSON: %w", err)
	}
	return string(jsonData)
}

func SnapshotFromJSON(input string) Snapshot {
	var snapshot Snapshot
	err := json.Unmarshal([]byte(input), &snapshot)
	if err != nil {
		LogError("Error deserializing from JSON: %w", err)
	}
	return snapshot
}
