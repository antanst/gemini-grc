package linkList

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gemini-grc/common/url"
)

type LinkList []url.URL

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
