-- File: storage_efficiency.sql
-- Shows potential storage savings from deduplication
-- Usage: \i misc/sql/storage_efficiency.sql

WITH duplicate_stats AS (
    SELECT 
        url,
        COUNT(*) as snapshot_count,
        COUNT(DISTINCT gemtext) as unique_gemtexts,
        COUNT(DISTINCT data) as unique_datas
    FROM snapshots
    GROUP BY url
    HAVING COUNT(*) > 1
)
SELECT 
    SUM(snapshot_count) as total_snapshots,
    SUM(unique_gemtexts + unique_datas) as unique_contents,
    SUM(snapshot_count) - SUM(unique_gemtexts + unique_datas) as duplicate_content_count,
    ROUND((SUM(snapshot_count) - SUM(unique_gemtexts + unique_datas)) * 100.0 / SUM(snapshot_count), 2) as duplicate_percentage
FROM duplicate_stats;