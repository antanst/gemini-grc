package gemini

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gemini-grc/logging"
	"strings"

	"github.com/guregu/null/v5"
)

type LinkList []GeminiUrl

func (l LinkList) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *LinkList) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	b, ok := value.([]byte) // Type assertion! Converts to []byte
	if !ok {
		return fmt.Errorf("failed to scan LinkList: expected []byte, got %T", value)
	}
	return json.Unmarshal(b, l)
}

type Snapshot struct {
	ID           int                `db:"id" json:"id,omitempty"`
	UID          string             `db:"uid" json:"uid,omitempty"`
	URL          GeminiUrl          `db:"url" json:"url,omitempty"`
	Host         string             `db:"host" json:"host,omitempty"`
	Timestamp    null.Time          `db:"timestamp" json:"timestamp,omitempty"`
	MimeType     null.String        `db:"mimetype" json:"mimetype,omitempty"`
	Data         null.Value[[]byte] `db:"data" json:"data,omitempty"`       // For non text/gemini files.
	GemText      null.String        `db:"gemtext" json:"gemtext,omitempty"` // For text/gemini files.
	Links        *LinkList          `db:"links" json:"links,omitempty"`
	Lang         null.String        `db:"lang" json:"lang,omitempty"`
	ResponseCode null.Int           `db:"response_code" json:"code,omitempty"` // Gemini response status code.
	Error        null.String        `db:"error" json:"error,omitempty"`        // On network errors only
}

func SnapshotToJSON(g Snapshot) string {
	// Serialize the Person struct to JSON
	jsonData, err := json.MarshalIndent(g, "", "\t")
	if err != nil {
		logging.LogError("Error serializing to JSON: %w", err)
	}
	return string(jsonData)
}

func SnapshotFromJSON(input string) Snapshot {
	var snapshot Snapshot
	err := json.Unmarshal([]byte(input), &snapshot)
	if err != nil {
		logging.LogError("Error deserializing from JSON: %w", err)
	}
	return snapshot
}

func ShouldPersistSnapshot(result *Snapshot) bool {
	if !result.MimeType.Valid {
		return false
	}
	if result.MimeType.String == "text/gemini" ||
		strings.HasPrefix(result.MimeType.String, "image/") ||
		strings.HasPrefix(result.MimeType.String, "text/") {
		return true
	}
	return false
}
