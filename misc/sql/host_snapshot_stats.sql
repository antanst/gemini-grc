-- File: host_snapshot_stats.sql
-- Groups snapshots by hosts and shows URLs with multiple snapshots
-- Usage: \i misc/sql/host_snapshot_stats.sql

SELECT 
    host,
    COUNT(DISTINCT url) as unique_urls,
    SUM(CASE WHEN url_count > 1 THEN 1 ELSE 0 END) as urls_with_multiple_snapshots,
    SUM(snapshot_count) as total_snapshots
FROM (
    SELECT 
        host, 
        url, 
        COUNT(*) as snapshot_count,
        COUNT(*) OVER (PARTITION BY url) as url_count
    FROM snapshots
    GROUP BY host, url
) subquery
GROUP BY host
ORDER BY total_snapshots DESC;