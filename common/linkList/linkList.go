package linkList

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gemini-grc/common/url"
)

type LinkList []url.URL

func (l LinkList) Value() (driver.Value, error) {
	if len(l) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (l *LinkList) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan LinkList: expected []byte, got %T", value)
	}
	return json.Unmarshal(b, l)
}
