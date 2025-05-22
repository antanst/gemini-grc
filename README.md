# gemini-grc

A crawler for the [Gemini](https://en.wikipedia.org/wiki/Gemini_(protocol)) network.
Easily extendable as a "wayback machine" of Gemini.

## Features
- [x] Save image/* and text/* files
- [x] Concurrent downloading with configurable number of workers
- [x] Connection limit per host
- [x] URL Blacklist
- [x] URL Whitelist (overrides blacklist and robots.txt)
- [x] Follow robots.txt, see gemini://geminiprotocol.net/docs/companion/robots.gmi
- [x] Configuration via environment variables
- [x] Storing capsule snapshots in PostgreSQL
- [x] Proper response header & body UTF-8 and format validation
- [x] Proper URL normalization
- [x] Handle redirects (3X status codes)
- [x] Crawl Gopher holes

## Security Note
This crawler uses `InsecureSkipVerify: true` in TLS configuration to accept all certificates. This is a common approach for crawlers but makes the application vulnerable to MITM attacks. This trade-off is made to enable crawling self-signed certificates widely used in the Gemini ecosystem.

## How to run

Spin up a PostgreSQL, check `db/sql/initdb.sql` to create the tables and start the crawler.
All configuration is done via environment variables.

## Configuration

Bool can be `true`,`false` or `0`,`1`.

```text
	LogLevel               string // Logging level (debug, info, warn, error)
	MaxResponseSize        int // Maximum size of response in bytes
	NumOfWorkers           int // Number of concurrent workers
	ResponseTimeout        int // Timeout for responses in seconds
	PanicOnUnexpectedError bool // Panic on unexpected errors when visiting a URL
	BlacklistPath          string // File that has blacklisted strings of "host:port"
	WhitelistPath          string // File with URLs that should always be crawled regardless of blacklist or robots.txt
	DryRun                 bool // If false, don't write to disk
	SkipIdenticalContent   bool // When true, skip storing snapshots with identical content
	SkipIfUpdatedDays      int  // Skip re-crawling URLs updated within this many days (0 to disable)
```

Example:

```shell
LOG_LEVEL=info \
NUM_OF_WORKERS=10 \
BLACKLIST_PATH="./blacklist.txt" \ # one url per line, can be empty
WHITELIST_PATH="./whitelist.txt" \ # URLs that override blacklist and robots.txt
MAX_RESPONSE_SIZE=10485760 \
RESPONSE_TIMEOUT=10 \
PANIC_ON_UNEXPECTED_ERROR=true \
PG_DATABASE=test \
PG_HOST=127.0.0.1 \
PG_MAX_OPEN_CONNECTIONS=100 \
PG_PORT=5434 \
PG_USER=test \
PG_PASSWORD=test \
DRY_RUN=false \
SKIP_IDENTICAL_CONTENT=false \
SKIP_IF_UPDATED_DAYS=7 \
./gemini-grc
```

## Development

Install linters. Check the versions first.
```shell
go install mvdan.cc/gofumpt@v0.7.0
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
```

## Snapshot History

The crawler now supports versioned snapshots, storing multiple snapshots of the same URL over time. This allows you to view how content changes over time, similar to the Internet Archive's Wayback Machine.

### Accessing Snapshot History

You can access the snapshot history using the included `snapshot_history.sh` script:

```bash
# Get the latest snapshot
./snapshot_history.sh -u gemini://example.com/

# Get a snapshot from a specific point in time
./snapshot_history.sh -u gemini://example.com/ -t 2023-05-01T12:00:00Z

# Get all snapshots for a URL
./snapshot_history.sh -u gemini://example.com/ -a

# Get snapshots in a date range
./snapshot_history.sh -u gemini://example.com/ -r 2023-01-01T00:00:00Z 2023-12-31T23:59:59Z
```

## TODO
- [x] Add snapshot history
- [ ] Add a web interface
- [ ] Provide to servers a TLS cert for sites that require it, like Astrobotany
- [ ] Use pledge/unveil in OpenBSD hosts

## TODO (lower priority)
- [ ] More protocols? http://dbohdan.sdf.org/smolnet/

## Notes
Good starting points:

gemini://warmedal.se/~antenna/
gemini://tlgs.one/
gopher://i-logout.cz:70/1/bongusta/
gopher://gopher.quux.org:70/