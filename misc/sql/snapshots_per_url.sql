-- File: snapshots_per_url.sql
-- Basic count of snapshots per URL
-- Usage: \i misc/sql/snapshots_per_url.sql

SELECT url, COUNT(*) as snapshot_count
FROM snapshots
GROUP BY url
ORDER BY snapshot_count DESC;