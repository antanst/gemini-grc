select count(*) from snapshots
  where last_crawled < now() - interval '30 days'
    and error IS NULL
    and gemtext IS NOT NULL
    and mimetype='text/gemini'
    and url ~ '^gemini://[^/]+/?$';
