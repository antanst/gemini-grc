-- DB creation and users
CREATE USER gemini;
ALTER USER gemini WITH PASSWORD 'gemini';
CREATE DATABASE gemini;
GRANT ALL PRIVILEGES ON DATABASE gemini TO gemini;
ALTER DATABASE gemini OWNER TO gemini;
GRANT ALL PRIVILEGES ON SCHEMA public TO gemini;
GRANT ALL PRIVILEGES ON gemini TO gemini;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO gemini;

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


\c gemini

-- Tables
DROP TABLE IF EXISTS snapshots;

CREATE TABLE snapshots (
    id SERIAL PRIMARY KEY,
    uid TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
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

ALTER TABLE snapshots OWNER TO gemini;

CREATE INDEX idx_uid ON snapshots (uid);
CREATE INDEX idx_url ON snapshots (url);
CREATE INDEX idx_timestamp ON snapshots (timestamp);
CREATE INDEX idx_mimetype ON snapshots (mimetype);
CREATE INDEX idx_lang ON snapshots (lang);
CREATE INDEX idx_response_code ON snapshots (response_code);
CREATE INDEX idx_error ON snapshots (error);
CREATE INDEX idx_host ON snapshots (host);
CREATE INDEX idx_response_code_error_nulls ON snapshots (response_code, error) WHERE response_code IS NULL AND error IS NULL;
