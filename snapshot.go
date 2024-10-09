package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Snapshot struct {
	UID       string      `json:"uid,omitempty"`
	URL       GeminiUrl   `json:"url,omitempty"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
	MimeType  string      `json:"mimetype,omitempty"`
	Data      []byte      `json:"data,omitempty"`
	GemText   string      `json:"gemtext,omitempty"`
	Links     []GeminiUrl `json:"links,omitempty"`
	Lang      string      `json:"lang,omitempty"`
	// Gemini status code
	ResponseCode int `json:"code,omitempty"`
	// On network errors, for Gemini server errors
	// we have ResponseCode above.
	Error error `json:"error,omitempty"`
}

func (u Snapshot) String() string {
	return fmt.Sprintf(
		"[%s] %s %s %s %d %s %s %s",
		u.UID, u.URL, u.Timestamp, u.Links, u.ResponseCode, u.MimeType, u.Lang, u.Error,
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
