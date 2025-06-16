-- File: snapshots_date_range.sql
-- Shows snapshot count with date range information for each URL
-- Usage: \i misc/sql/snapshots_date_range.sql

SELECT 
    url, 
    COUNT(*) as snapshot_count,
    MIN(timestamp) as first_snapshot,
    MAX(timestamp) as last_snapshot,
    MAX(timestamp) - MIN(timestamp) as time_span
FROM snapshots
GROUP BY url
HAVING COUNT(*) > 1
ORDER BY snapshot_count DESC;