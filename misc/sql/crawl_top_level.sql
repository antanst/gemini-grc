BEGIN;

WITH matching_urls AS (
    SELECT url, host
    FROM snapshots
    WHERE url ~ '^gemini://[^/]+/$'
      AND timestamp < (NOW() - INTERVAL '1 week')
    ORDER BY random()
    LIMIT 500
)
INSERT INTO urls (url, host)
SELECT url, host
FROM matching_urls
ON CONFLICT DO NOTHING;

-- WITH matching_urls AS (
--     SELECT url, host
--     FROM snapshots
--     WHERE url ~ '^gemini://[^/]+/$'
--       AND timestamp < (NOW() - INTERVAL '1 week')
--     ORDER BY random()
--     LIMIT 500
-- )
-- DELETE FROM snapshots
-- WHERE url IN (
--     SELECT url
--     FROM matching_urls
-- );

COMMIT;
