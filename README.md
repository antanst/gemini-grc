# gemini-grc

A crawler for the [Gemini](https://en.wikipedia.org/wiki/Gemini_(protocol)) network.
Easily extendable as a "wayback machine" of Gemini.

## Features
- [x] Concurrent downloading with configurable number of workers
- [x] Save image/* and text/* files
- [x] Connection limit per host
- [x] URL Blacklist
- [x] URL Whitelist (overrides blacklist and robots.txt)
- [x] Follow robots.txt, see gemini://geminiprotocol.net/docs/companion/robots.gmi
- [x] Configuration via command-line flags
- [x] Storing capsule snapshots in PostgreSQL
- [x] Proper response header & body UTF-8 and format validation
- [x] Proper URL normalization
- [x] Handle redirects (3X status codes)
- [x] Crawl Gopher holes

## Security Note
This crawler uses `InsecureSkipVerify: true` in TLS configuration to accept all certificates. This is a common approach for crawlers but makes the application vulnerable to MITM attacks. This trade-off is made to enable crawling self-signed certificates widely used in the Gemini ecosystem.

## How to run

```shell
make build
./dist/crawler --help
```

Check `misc/sql/initdb.sql` to create the PostgreSQL tables.

## Configuration

Available command-line flags:

```text
  -blacklist-path string
        File that has blacklist regexes
  -dry-run
        Dry run mode
  -gopher
        Enable crawling of Gopher holes
  -log-level string
        Logging level (debug, info, warn, error) (default "info")
  -max-db-connections int
        Maximum number of database connections (default 100)
  -max-response-size int
        Maximum size of response in bytes (default 1048576)
  -pgurl string
        Postgres URL
  -response-timeout int
        Timeout for network responses in seconds (default 10)
  -seed-url-path string
        File with seed URLs that should be added to the queue immediately
  -skip-if-updated-days int
        Skip re-crawling URLs updated within this many days (0 to disable) (default 60)
  -whitelist-path string
        File with URLs that should always be crawled regardless of blacklist
  -workers int
        Number of concurrent workers (default 1)
```

Example:

```shell
./dist/crawler \
  -pgurl="postgres://test:test@127.0.0.1:5434/test?sslmode=disable" \
  -log-level=info \
  -workers=10 \
  -blacklist-path="./blacklist.txt" \
  -whitelist-path="./whitelist.txt" \
  -max-response-size=10485760 \
  -response-timeout=10 \
  -max-db-connections=100 \
  -skip-if-updated-days=7 \
  -gopher \
  -seed-url-path="./seed_urls.txt"
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