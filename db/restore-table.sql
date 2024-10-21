BEGIN;

SET statement_timeout = '10min';

TRUNCATE TABLE snapshots;

INSERT INTO snapshots SELECT * FROM backup;

COMMIT;
