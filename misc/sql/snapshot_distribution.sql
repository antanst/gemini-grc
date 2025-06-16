-- File: snapshot_distribution.sql
-- Shows the distribution of snapshots per URL (how many URLs have 1, 2, 3, etc. snapshots)
-- Usage: \i misc/sql/snapshot_distribution.sql

WITH counts AS (
    SELECT url, COUNT(*) as snapshot_count
    FROM snapshots
    GROUP BY url
)
SELECT 
    snapshot_count,
    COUNT(*) as url_count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM counts
GROUP BY snapshot_count
ORDER BY snapshot_count;