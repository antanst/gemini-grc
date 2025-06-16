-- File: content_changes.sql
-- Finds URLs with the most content changes between consecutive snapshots
-- Usage: \i misc/sql/content_changes.sql

WITH snapshot_changes AS (
    SELECT 
        s1.url,
        s1.timestamp as prev_timestamp,
        s2.timestamp as next_timestamp,
        s1.gemtext IS DISTINCT FROM s2.gemtext as gemtext_changed,
        s1.data IS DISTINCT FROM s2.data as data_changed
    FROM snapshots s1
    JOIN snapshots s2 ON s1.url = s2.url AND s1.timestamp < s2.timestamp
    WHERE NOT EXISTS (
        SELECT 1 FROM snapshots s3
        WHERE s3.url = s1.url AND s1.timestamp < s3.timestamp AND s3.timestamp < s2.timestamp
    )
)
SELECT 
    url,
    COUNT(*) + 1 as snapshot_count,
    SUM(CASE WHEN gemtext_changed OR data_changed THEN 1 ELSE 0 END) as content_changes
FROM snapshot_changes
GROUP BY url
HAVING COUNT(*) + 1 > 1
ORDER BY content_changes DESC, snapshot_count DESC;