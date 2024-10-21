package gemini

import (
	"encoding/json"
	"fmt"
	"gemini-grc/logging"
)

type GeminiUrl struct {
	Protocol string `json:"protocol,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     int    `json:"port,omitempty"`
	Path     string `json:"path,omitempty"`
	Descr    string `json:"descr,omitempty"`
	Full     string `json:"full,omitempty"`
}

func (g *GeminiUrl) Scan(value interface{}) error {
	if value == nil {
		// Clear the fields in the current GeminiUrl object (not the pointer itself)
		*g = GeminiUrl{}
		return nil
	}
	b, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan GeminiUrl: expected string, got %T", value)
	}
	parsedUrl, err := ParseUrl(b, "")
	if err != nil {
		return err
	}
	*g = *parsedUrl
	return nil
}

func (u GeminiUrl) String() string {
	return u.Full
	//	return fmt.Sprintf("%s://%s:%d%s", u.Protocol, u.Hostname, u.Port, u.Path)
}

func GeminiUrltoJSON(g GeminiUrl) string {
	// Serialize the Person struct to JSON
	jsonData, err := json.Marshal(g)
	if err != nil {
		logging.LogError("Error serializing to JSON: %w", err)
	}
	return string(jsonData)
}

func GeminiUrlFromJSON(input string) GeminiUrl {
	var geminiUrl GeminiUrl
	err := json.Unmarshal([]byte(input), &geminiUrl)
	if err != nil {
		logging.LogError("Error deserializing from JSON: %w", err)
	}
	return geminiUrl
}
