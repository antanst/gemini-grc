package snapshot

import (
	"time"

	"gemini-grc/common/linkList"
	commonUrl "gemini-grc/common/url"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
)

type Snapshot struct {
	ID           int                           `db:"id" json:"ID,omitempty"`
	URL          commonUrl.URL                 `db:"url" json:"url,omitempty"`
	Host         string                        `db:"host" json:"host,omitempty"`
	Timestamp    null.Time                     `db:"timestamp" json:"timestamp,omitempty"`
	MimeType     null.String                   `db:"mimetype" json:"mimetype,omitempty"`
	Data         null.Value[[]byte]            `db:"data" json:"data,omitempty"`       // For non text/gemini files.
	GemText      null.String                   `db:"gemtext" json:"gemtext,omitempty"` // For text/gemini files.
	Header       null.String                   `db:"header" json:"header,omitempty"`   // Response header.
	Links        null.Value[linkList.LinkList] `db:"links" json:"links,omitempty"`
	Lang         null.String                   `db:"lang" json:"lang,omitempty"`
	ResponseCode null.Int                      `db:"response_code" json:"code,omitempty"`        // Gemini response Status code.
	Error        null.String                   `db:"error" json:"error,omitempty"`               // On network errors only
	LastCrawled  null.Time                     `db:"last_crawled" json:"last_crawled,omitempty"` // When URL was last processed (regardless of content changes)
}

func SnapshotFromURL(u string, normalize bool) (*Snapshot, error) {
	url, err := commonUrl.ParseURL(u, "", normalize)
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}
	newSnapshot := Snapshot{
		URL:       *url,
		Host:      url.Hostname,
		Timestamp: null.TimeFrom(time.Now()),
	}
	return &newSnapshot, nil
}
