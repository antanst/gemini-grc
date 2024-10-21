WITH DuplicateSnapshots AS (
    SELECT id,
           url,
           ROW_NUMBER() OVER (PARTITION BY url ORDER BY id) AS row_num
    FROM snapshots
)
SELECT *
FROM DuplicateSnapshots
WHERE row_num > 1;
