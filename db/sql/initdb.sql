DROP TABLE IF EXISTS snapshots;

CREATE TABLE snapshots (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    host TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    mimetype TEXT,
    data BYTEA,
    gemtext TEXT,
    links JSONB,
    lang TEXT,
    response_code INTEGER,
    error TEXT
);

CREATE INDEX idx_url ON snapshots (url);
CREATE INDEX idx_timestamp ON snapshots (timestamp);
CREATE INDEX idx_mimetype ON snapshots (mimetype);
CREATE INDEX idx_lang ON snapshots (lang);
CREATE INDEX idx_response_code ON snapshots (response_code);
CREATE INDEX idx_error ON snapshots (error);
CREATE INDEX idx_host ON snapshots (host);
CREATE INDEX unique_uid_url ON snapshots (uid, url);
CREATE INDEX idx_response_code_error_nulls ON snapshots (response_code, error) WHERE response_code IS NULL AND error IS NULL;

CREATE TABLE urls (
                      id SERIAL PRIMARY KEY,
                      url TEXT NOT NULL UNIQUE,
                      host TEXT NOT NULL,
                      timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_urls_url ON urls (url);
CREATE INDEX idx_urls_timestamp ON urls (timestamp);

