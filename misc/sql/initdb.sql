DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS urls;

CREATE TABLE urls (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    host TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    being_processed BOOLEAN
);

CREATE UNIQUE INDEX urls_url_key ON urls (url);
CREATE INDEX idx_urls_url ON urls (url);
CREATE INDEX idx_urls_timestamp ON urls (timestamp);
CREATE INDEX idx_being_processed ON urls (being_processed);

CREATE TABLE snapshots (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    host TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    mimetype TEXT,
    data BYTEA,
    gemtext TEXT,
    links JSONB,
    lang TEXT,
    response_code INTEGER,
    error TEXT,
    header TEXT,
    last_crawled TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_url_timestamp ON snapshots (url, timestamp);
CREATE INDEX idx_url ON snapshots (url);
CREATE INDEX idx_timestamp ON snapshots (timestamp);
CREATE INDEX idx_mimetype ON snapshots (mimetype);
CREATE INDEX idx_lang ON snapshots (lang);
CREATE INDEX idx_response_code ON snapshots (response_code);
CREATE INDEX idx_error ON snapshots (error);
CREATE INDEX idx_host ON snapshots (host);
CREATE INDEX idx_response_code_error ON snapshots (response_code, error);
CREATE INDEX idx_response_code_error_nulls ON snapshots (response_code, error) WHERE response_code IS NULL AND error IS NULL;
CREATE INDEX idx_snapshots_unprocessed ON snapshots (host) WHERE response_code IS NULL AND error IS NULL;
CREATE INDEX idx_url_latest ON snapshots (url, timestamp DESC);
CREATE INDEX idx_last_crawled ON snapshots (last_crawled);
CREATE INDEX idx_url_last_crawled ON snapshots (url, last_crawled DESC);