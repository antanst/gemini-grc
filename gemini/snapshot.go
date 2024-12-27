package gemini

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/guregu/null/v5"
)

type LinkList []URL

func (l *LinkList) Value() (driver.Value, error) {
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
	ID           int                  `db:"id" json:"id,omitempty"`
	URL          URL                  `db:"url" json:"url,omitempty"`
	Host         string               `db:"host" json:"host,omitempty"`
	Timestamp    null.Time            `db:"timestamp" json:"timestamp,omitempty"`
	MimeType     null.String          `db:"mimetype" json:"mimetype,omitempty"`
	Data         null.Value[[]byte]   `db:"data" json:"data,omitempty"`       // For non text/gemini files.
	GemText      null.String          `db:"gemtext" json:"gemtext,omitempty"` // For text/gemini files.
	Header       null.String          `db:"header" json:"header,omitempty"`   // Response header.
	Links        null.Value[LinkList] `db:"links" json:"links,omitempty"`
	Lang         null.String          `db:"lang" json:"lang,omitempty"`
	ResponseCode null.Int             `db:"response_code" json:"code,omitempty"` // Gemini response status code.
	Error        null.String          `db:"error" json:"error,omitempty"`        // On network errors only
}
