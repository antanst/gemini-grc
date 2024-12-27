package gemini

const (
	SQL_SELECT_RANDOM_UNVISITED_SNAPSHOTS = `
SELECT *
FROM snapshots
WHERE response_code IS NULL
  AND error IS NULL
ORDER BY RANDOM()
FOR UPDATE SKIP LOCKED
LIMIT $1
	`
	SQL_SELECT_RANDOM_UNVISITED_SNAPSHOTS_UNIQUE_HOSTS = `
SELECT *
FROM snapshots s
WHERE response_code IS NULL
  AND error IS NULL
  AND s.id IN (
      SELECT MIN(id)
      FROM snapshots
      WHERE response_code IS NULL 
        AND error IS NULL
      GROUP BY host
  )
ORDER BY RANDOM()
FOR UPDATE SKIP LOCKED
LIMIT $1
`
	SQL_SELECT_UNVISITED_SNAPSHOTS_UNIQUE_HOSTS = `
SELECT *
FROM snapshots s
WHERE response_code IS NULL
  AND error IS NULL
  AND s.id IN (
      SELECT MIN(id)
      FROM snapshots
      WHERE response_code IS NULL
        AND error IS NULL
      GROUP BY host
  )
FOR UPDATE SKIP LOCKED
LIMIT $1
`
	SQL_INSERT_SNAPSHOT_IF_NEW = `
        INSERT INTO snapshots (url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error)
        VALUES (:url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error)
        ON CONFLICT (url) DO NOTHING
    `
	SQL_UPSERT_SNAPSHOT = `INSERT INTO snapshots (url, host, timestamp, mimetype, data, gemtext, links, lang, response_code, error)
        VALUES (:url, :host, :timestamp, :mimetype, :data, :gemtext, :links, :lang, :response_code, :error)
        ON CONFLICT (url) DO UPDATE SET
            url = EXCLUDED.url,
            host = EXCLUDED.host,
            timestamp = EXCLUDED.timestamp,
            mimetype = EXCLUDED.mimetype,
            data = EXCLUDED.data,
            gemtext = EXCLUDED.gemtext,
            links = EXCLUDED.links,
            lang = EXCLUDED.lang,
            response_code = EXCLUDED.response_code,
            error = EXCLUDED.error
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
error = :error
WHERE id = :id
RETURNING id
`
)
