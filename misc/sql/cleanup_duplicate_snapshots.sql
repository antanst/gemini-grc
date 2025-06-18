WITH snapshot_rankings AS (
      SELECT
          id,
          url,
          ROW_NUMBER() OVER (
              PARTITION BY url
              ORDER BY
                  CASE WHEN (gemtext IS NOT NULL AND gemtext != '') OR data IS NOT NULL
                       THEN 0 ELSE 1 END,
                  timestamp DESC
          ) as rn
      FROM snapshots
  )
  DELETE FROM snapshots
  WHERE id IN (
      SELECT id
      FROM snapshot_rankings
      WHERE rn > 1
  );
