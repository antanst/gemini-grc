BEGIN;

-- Increase statement timeout
SET statement_timeout = '10min';

-- Step 1: Create a new table with the same schema
CREATE TABLE backup (LIKE snapshots INCLUDING ALL);

-- Step 2: Copy data from the old table to the new one
INSERT INTO backup SELECT * FROM snapshots;

-- (Optional) Step 3: Truncate the original table if you are moving the data
-- TRUNCATE TABLE snapshots;

-- Commit the transaction if everything went well
COMMIT;
