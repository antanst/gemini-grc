-- Explanation:

--     WITH DuplicateSnapshots AS:
--         This is a Common Table Expression (CTE) that selects all rows from the snapshots table.
--         ROW_NUMBER() OVER (PARTITION BY url ORDER BY id): This assigns a unique row number to each row with the same url. The PARTITION BY url groups the rows by url, and ORDER BY id ensures that the row with the smallest id is given row_num = 1.
--     DELETE FROM snapshots WHERE id IN:
--         The DELETE statement deletes rows from the snapshots table where the id is in the result of the subquery.
--     WHERE row_num > 1:
--         In the subquery, we select only rows where row_num > 1, which means only the duplicate rows (since row_num = 1 is the one row we want to keep).

-- Result:

--     This query will delete all duplicate rows from the snapshots table, keeping only the row with the smallest id for each url.
--     If multiple rows share the same url, only the first one (based on id) will be retained.

WITH DuplicateSnapshots AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY url ORDER BY id) AS row_num
    FROM snapshots
)
DELETE FROM snapshots
WHERE id IN (
    SELECT id
    FROM DuplicateSnapshots
    WHERE row_num > 1
);
