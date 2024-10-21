SELECT host, COUNT(*) AS row_count
FROM snapshots
GROUP BY host
ORDER BY row_count DESC
LIMIT 10;
