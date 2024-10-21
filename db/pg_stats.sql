SELECT 
    query,
    total_exec_time AS total_time,  -- total time spent on the query execution
    calls,                          -- number of times the query has been called
    mean_exec_time AS mean_time     -- average time per execution
--    max_exec_time AS max_time        -- maximum time taken for any single execution
FROM 
    pg_stat_statements
ORDER BY 
    total_exec_time DESC             -- order by total execution time
LIMIT 5;

