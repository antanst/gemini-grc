# SQL Queries for Snapshot Analysis

This directory contains SQL queries to analyze snapshot data in the gemini-grc database.

## Usage

You can run these queries directly from psql using the `\i` directive:

```
\i misc/sql/snapshots_per_url.sql
```

## Available Queries

- **snapshots_per_url.sql** - Basic count of snapshots per URL
- **snapshots_date_range.sql** - Shows snapshot count with date range information for each URL
- **host_snapshot_stats.sql** - Groups snapshots by hosts and shows URLs with multiple snapshots
- **content_changes.sql** - Finds URLs with the most content changes between consecutive snapshots
- **snapshot_distribution.sql** - Shows the distribution of snapshots per URL (how many URLs have 1, 2, 3, etc. snapshots)
- **recent_snapshot_activity.sql** - Shows URLs with most snapshots in the last 7 days
- **storage_efficiency.sql** - Shows potential storage savings from deduplication
- **snapshots_by_timeframe.sql** - Shows snapshot count by timeframe (day, week, month)

## Notes

- These queries are designed to work with PostgreSQL and the gemini-grc database schema
- Some queries may be resource-intensive on large databases
- The results can help optimize storage and understand the effectiveness of the versioned snapshot feature