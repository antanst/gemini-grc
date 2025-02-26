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
ORDER BY RANDOM()
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
	SQL_INSERT_URL = `
        INSERT INTO urls (url, host, timestamp)
        VALUES (:url, :host, :timestamp)
        ON CONFLICT (url) DO NOTHING
    `
	SQL_DELETE_URL = `
        DELETE FROM urls WHERE url=$1
    `
)
