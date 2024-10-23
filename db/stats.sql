SELECT
    COUNT(CASE WHEN response_code IS NOT NULL AND error IS NULL THEN 1 END) AS "Visited",
    COUNT(CASE WHEN response_code IS NULL     AND error IS NULL THEN 1 END) AS "Pending",
    COUNT(CASE WHEN error IS NOT NULL THEN 1 END) AS "Errors"
FROM snapshots;
