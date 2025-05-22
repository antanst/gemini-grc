package db

const (
	SQL_SELECT_RANDOM_URLS_UNIQUE_HOSTS = `
SELECT url
FROM urls u
WHERE u.id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY host ORDER BY id) as rn
        FROM urls
    ) t
    WHERE rn <= 3
)
ORDER BY RANDOM()
FOR UPDATE SKIP LOCKED
LIMIT $1
	`
	SQL_SELECT_RANDOM_URLS = `
SELECT url
FROM urls u
WHERE u.being_processed IS NOT TRUE
ORDER BY RANDOM()
FOR UPDATE SKIP LOCKED
LIMIT $1
`
	SQL_MARK_URLS_BEING_PROCESSED      = `UPDATE urls SET being_processed = true WHERE url IN (%s)`
	SQL_SELECT_RANDOM_URLS_GEMINI_ONLY = `
SELECT url
FROM urls u
WHERE u.url like 'gemini://%'
  AND u.being_processed IS NOT TRUE
ORDER BY RANDOM()
FOR UPDATE SKIP LOCKED
LIMIT $1
`
	SQL_SELECT_RANDOM_URLS_GEMINI_ONLY_2 = `
WITH RankedUrls AS (
    -- Step 1: Assign a random rank to each URL within its host group
    SELECT
        url,
        host,
        ROW_NUMBER() OVER (PARTITION BY host ORDER BY RANDOM()) as rn
    FROM
        urls
    WHERE url like 'gemini://%'
      AND being_processed IS NOT TRUE
),
OneUrlPerHost AS (
    -- Step 2: Filter to keep only the first-ranked (random) URL per host
    SELECT
        url,
        host
    FROM
        RankedUrls
    WHERE
        rn = 1
)
-- Step 3: From the set of one URL per host, randomly select X
SELECT
    url
FROM
    OneUrlPerHost
ORDER BY
    RANDOM()
FOR UPDATE SKIP LOCKED
LIMIT $1
`
	// New query - always insert a new snapshot without conflict handling
	SQL_INSERT_SNAPSHOT = `
        INSERT INTO snapshots (url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error, header)
        VALUES (:url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error, :header)
        RETURNING id
    `
	// Keep for backward compatibility, but should be phased out
	SQL_INSERT_SNAPSHOT_IF_NEW = `
        INSERT INTO snapshots (url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error, header)
        VALUES (:url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error, :header)
        ON CONFLICT DO NOTHING
    `
	// Update to match the SQL_INSERT_SNAPSHOT - we no longer want to upsert, just insert new versions
	SQL_UPSERT_SNAPSHOT = `
        INSERT INTO snapshots (url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error, header)
        VALUES (:url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error, :header)
        RETURNING id
    `
	SQL_UPDATE_SNAPSHOT = `UPDATE snapshots
SET url = :url,
host = :host,
timestamp = :timestamp,
mimetype = :mimetype,
data = :data,
gemtext = :gemtext,
links = :links,
lang = :lang,
response_code = :response_code,
error = :error,
header = :header
WHERE id = :id
RETURNING id
`
	SQL_INSERT_URL = `
        INSERT INTO urls (url, host, timestamp)
        VALUES (:url, :host, :timestamp)
        ON CONFLICT (url) DO NOTHING
    `
	SQL_UPDATE_URL = `
        UPDATE urls
        SET url = :NormalizedURL
        WHERE url = :Url
        AND NOT EXISTS (
            SELECT 1 FROM urls WHERE url = :NormalizedURL
        )
    `
	SQL_DELETE_URL = `
        DELETE FROM urls WHERE url=$1
    `
	// New queries for retrieving snapshots
	SQL_GET_LATEST_SNAPSHOT = `
        SELECT * FROM snapshots
        WHERE url = $1
        ORDER BY timestamp DESC
        LIMIT 1
    `
	SQL_GET_SNAPSHOT_AT_TIMESTAMP = `
        SELECT * FROM snapshots
        WHERE url = $1
        AND timestamp <= $2
        ORDER BY timestamp DESC
        LIMIT 1
    `
	SQL_GET_ALL_SNAPSHOTS_FOR_URL = `
        SELECT * FROM snapshots
        WHERE url = $1
        ORDER BY timestamp DESC
    `
	SQL_GET_SNAPSHOTS_BY_DATE_RANGE = `
        SELECT * FROM snapshots
        WHERE url = $1
        AND timestamp BETWEEN $2 AND $3
        ORDER BY timestamp DESC
    `
)
