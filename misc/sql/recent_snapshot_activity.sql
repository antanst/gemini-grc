-- File: recent_snapshot_activity.sql
-- Shows URLs with most snapshots in the last 7 days
-- Usage: \i misc/sql/recent_snapshot_activity.sql

SELECT 
    url, 
    COUNT(*) as snapshot_count
FROM snapshots
WHERE timestamp > NOW() - INTERVAL '7 days'
GROUP BY url
HAVING COUNT(*) > 1
ORDER BY snapshot_count DESC
LIMIT 20;