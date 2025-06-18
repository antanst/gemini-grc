-- Cleanup script for snapshots table after adding last_crawled column
-- This script consolidates multiple snapshots per URL by:
-- 1. Keeping the latest snapshot with content (non-null gemtext OR data)
-- 2. Setting its last_crawled to the most recent timestamp from any snapshot for that URL
-- 3. Deleting all other snapshots for URLs with multiple snapshots
--
-- IMPORTANT: This script will permanently delete data. Make sure to backup your database first!

BEGIN;

-- Update last_crawled for URLs with multiple snapshots
-- Keep the latest snapshot with content and update its last_crawled to the most recent timestamp
WITH url_snapshots AS (
    -- Get all snapshots grouped by URL with row numbers
    SELECT 
        id,
        url,
        timestamp,
        last_crawled,
        gemtext,
        data,
        ROW_NUMBER() OVER (PARTITION BY url ORDER BY timestamp DESC) as rn_by_timestamp
    FROM snapshots
),
latest_content_snapshots AS (
    -- Find the latest snapshot with content for each URL
    SELECT 
        url,
        id as keep_id,
        timestamp as keep_timestamp
    FROM url_snapshots
    WHERE (gemtext IS NOT NULL OR data IS NOT NULL)
        AND rn_by_timestamp = (
            SELECT MIN(rn_by_timestamp) 
            FROM url_snapshots us2 
            WHERE us2.url = url_snapshots.url 
                AND (us2.gemtext IS NOT NULL OR us2.data IS NOT NULL)
        )
),
most_recent_timestamps AS (
    -- Get the most recent timestamp (last_crawled or timestamp) for each URL
    SELECT 
        url,
        GREATEST(
            MAX(timestamp),
            COALESCE(MAX(last_crawled), '1970-01-01'::timestamp)
        ) as most_recent_time
    FROM snapshots
    GROUP BY url
)
-- Update the last_crawled of snapshots we're keeping
UPDATE snapshots 
SET last_crawled = mrt.most_recent_time
FROM latest_content_snapshots lcs
JOIN most_recent_timestamps mrt ON lcs.url = mrt.url
WHERE snapshots.id = lcs.keep_id;

-- Delete all other snapshots for URLs that have multiple snapshots
WITH url_snapshots AS (
    SELECT 
        id,
        url,
        timestamp,
        gemtext,
        data,
        ROW_NUMBER() OVER (PARTITION BY url ORDER BY timestamp DESC) as rn_by_timestamp
    FROM snapshots
),
latest_content_snapshots AS (
    -- Find the latest snapshot with content for each URL
    SELECT 
        url,
        id as keep_id
    FROM url_snapshots
    WHERE (gemtext IS NOT NULL OR data IS NOT NULL)
        AND rn_by_timestamp = (
            SELECT MIN(rn_by_timestamp) 
            FROM url_snapshots us2 
            WHERE us2.url = url_snapshots.url 
                AND (us2.gemtext IS NOT NULL OR us2.data IS NOT NULL)
        )
),
snapshots_to_delete AS (
    -- Find snapshots to delete (all except the ones we're keeping)
    SELECT s.id
    FROM snapshots s
    LEFT JOIN latest_content_snapshots lcs ON s.id = lcs.keep_id
    WHERE lcs.keep_id IS NULL
        AND s.url IN (
            -- Only for URLs that have multiple snapshots
            SELECT url 
            FROM snapshots 
            GROUP BY url 
            HAVING COUNT(*) > 1
        )
)
DELETE FROM snapshots 
WHERE id IN (SELECT id FROM snapshots_to_delete);

-- Show summary of changes
SELECT 
    'Cleanup completed. Remaining snapshots: ' || COUNT(*) as summary
FROM snapshots;

-- Show URLs that still have multiple snapshots (should be 0 after cleanup)
SELECT 
    'URLs with multiple snapshots after cleanup: ' || COUNT(*) as validation
FROM (
    SELECT url 
    FROM snapshots 
    GROUP BY url 
    HAVING COUNT(*) > 1
) multi_snapshots;

COMMIT;