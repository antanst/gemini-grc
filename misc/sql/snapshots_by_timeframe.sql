-- File: snapshots_by_timeframe.sql
-- Shows snapshot count by timeframe (day, week, month)
-- Usage: \i misc/sql/snapshots_by_timeframe.sql

WITH daily_snapshots AS (
    SELECT 
        date_trunc('day', timestamp) as day,
        COUNT(*) as snapshot_count,
        COUNT(DISTINCT url) as unique_urls
    FROM snapshots
    GROUP BY day
    ORDER BY day
),
weekly_snapshots AS (
    SELECT 
        date_trunc('week', timestamp) as week,
        COUNT(*) as snapshot_count,
        COUNT(DISTINCT url) as unique_urls
    FROM snapshots
    GROUP BY week
    ORDER BY week
),
monthly_snapshots AS (
    SELECT 
        date_trunc('month', timestamp) as month,
        COUNT(*) as snapshot_count,
        COUNT(DISTINCT url) as unique_urls
    FROM snapshots
    GROUP BY month
    ORDER BY month
)
SELECT 'Daily' as timeframe, * FROM daily_snapshots
UNION ALL
SELECT 'Weekly' as timeframe, * FROM weekly_snapshots
UNION ALL
SELECT 'Monthly' as timeframe, * FROM monthly_snapshots
ORDER BY timeframe, day;