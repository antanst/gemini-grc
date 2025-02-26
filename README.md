# gemini-grc

A crawler for the [Gemini](https://en.wikipedia.org/wiki/Gemini_(protocol)) network.
Easily extendable as a "wayback machine" of Gemini.

## Features
- [x] Save image/* and text/* files
- [x] Concurrent downloading with configurable number of workers
- [x] Connection limit per host
- [x] URL Blacklist
- [x] Follow robots.txt, see gemini://geminiprotocol.net/docs/companion/robots.gmi
- [x] Configuration via environment variables
- [x] Storing capsule snapshots in PostgreSQL
- [x] Proper response header & body UTF-8 and format validation
- [x] Proper URL normalization
- [x] Handle redirects (3X status codes)
- [x] Crawl Gopher holes

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
	WorkerBatchSize        int // Batch size for worker processing
	PanicOnUnexpectedError bool // Panic on unexpected errors when visiting a URL
	BlacklistPath          string // File that has blacklisted strings of "host:port"
	DryRun                 bool // If false, don't write to disk
	PrintWorkerStatus      bool // If false, print logs and not worker status table
```

Example:

```shell
LOG_LEVEL=info \
NUM_OF_WORKERS=10 \
WORKER_BATCH_SIZE=10 \
BLACKLIST_PATH="./blacklist.txt" \ # one url per line, can be empty
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
./gemini-grc
```

## Development

Install linters. Check the versions first.
```shell
go install mvdan.cc/gofumpt@v0.7.0
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4
```

## TODO
- [ ] Add snapshot history
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