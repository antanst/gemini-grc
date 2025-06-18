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
        INSERT INTO snapshots (url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error, header, last_crawled)
        VALUES (:url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error, :header, :last_crawled)
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
	// Update last_crawled timestamp for the most recent snapshot of a URL
	SQL_UPDATE_LAST_CRAWLED = `
        UPDATE snapshots 
        SET last_crawled = CURRENT_TIMESTAMP 
        WHERE id = (
            SELECT id FROM snapshots 
            WHERE url = $1 
            ORDER BY timestamp DESC 
            LIMIT 1
        )
    `
	// SQL_FETCH_SNAPSHOTS_FROM_HISTORY Fetches URLs from snapshots for re-crawling based on last_crawled timestamp
	// This query finds root domain URLs that haven't been crawled recently and selects
	// one URL per host for diversity. Uses CTEs to:
	// 1. Find latest crawl attempt per URL (via MAX(last_crawled))
	// 2. Filter to URLs with actual content and successful responses (20-29)
	// 3. Select URLs where latest crawl is older than cutoff date
	// 4. Rank randomly within each host and pick one URL per host
	// Parameters: $1 = cutoff_date, $2 = limit
	SQL_FETCH_SNAPSHOTS_FROM_HISTORY = `
		WITH latest_attempts AS (
			SELECT 
				url,
				host,
				COALESCE(MAX(last_crawled), '1970-01-01'::timestamp) as latest_attempt
			FROM snapshots
			WHERE url ~ '^gemini://[^/]+/?$' AND mimetype = 'text/gemini'
			GROUP BY url, host
		),
		root_urls_with_content AS (
			SELECT DISTINCT
				la.url,
				la.host,
				la.latest_attempt
			FROM latest_attempts la
			JOIN snapshots s ON s.url = la.url 
			WHERE (s.gemtext IS NOT NULL OR s.data IS NOT NULL)
				AND s.response_code BETWEEN 20 AND 29
		),
		eligible_urls AS (
			SELECT 
				url,
				host,
				latest_attempt
			FROM root_urls_with_content
			WHERE latest_attempt < $1
		),
		ranked_urls AS (
			SELECT
				url,
				host,
				latest_attempt,
				ROW_NUMBER() OVER (PARTITION BY host ORDER BY RANDOM()) as rank
			FROM eligible_urls
		)
		SELECT url, host
		FROM ranked_urls
		WHERE rank = 1
		ORDER BY RANDOM()
		LIMIT $2
    `
)
