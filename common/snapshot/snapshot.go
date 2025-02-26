package snapshot

import (
	"time"

	"gemini-grc/common/linkList"
	commonUrl "gemini-grc/common/url"
	"github.com/antanst/go_errors"
	"github.com/guregu/null/v5"
)

type Snapshot struct {
	ID           int                           `db:"ID" json:"ID,omitempty"`
	URL          commonUrl.URL                 `db:"url" json:"url,omitempty"`
	Host         string                        `db:"host" json:"host,omitempty"`
	Timestamp    null.Time                     `db:"timestamp" json:"timestamp,omitempty"`
	MimeType     null.String                   `db:"mimetype" json:"mimetype,omitempty"`
	Data         null.Value[[]byte]            `db:"data" json:"data,omitempty"`       // For non text/gemini files.
	GemText      null.String                   `db:"gemtext" json:"gemtext,omitempty"` // For text/gemini files.
	Header       null.String                   `db:"header" json:"header,omitempty"`   // Response header.
	Links        null.Value[linkList.LinkList] `db:"links" json:"links,omitempty"`
	Lang         null.String                   `db:"lang" json:"lang,omitempty"`
	ResponseCode null.Int                      `db:"response_code" json:"code,omitempty"` // Gemini response Status code.
	Error        null.String                   `db:"error" json:"error,omitempty"`        // On network errors only
}

func SnapshotFromURL(u string, normalize bool) (*Snapshot, error) {
	url, err := commonUrl.ParseURL(u, "", normalize)
	if err != nil {
		return nil, go_errors.NewError(err)
	}
	newSnapshot := Snapshot{
		URL:       *url,
		Host:      url.Hostname,
		Timestamp: null.TimeFrom(time.Now()),
	}
	return &newSnapshot, nil
}
