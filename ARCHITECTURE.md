# gemini-grc Architectural Notes

## 20250513 - Versioned Snapshots

The crawler now supports saving multiple versions of the same URL over time, similar to the Internet Archive's Wayback Machine. This document outlines the architecture and changes made to support this feature.

### Database Schema Changes

The following changes to the database schema are required:

```sql
-- Remove UNIQUE constraint from url in snapshots table
ALTER TABLE snapshots DROP CONSTRAINT unique_url;

-- Create a composite primary key using url and timestamp
CREATE UNIQUE INDEX idx_url_timestamp ON snapshots (url, timestamp);

-- Add a new index to efficiently find the latest snapshot
CREATE INDEX idx_url_latest ON snapshots (url, timestamp DESC);
```

### Code Changes

1. **Updated SQL Queries**:
   - Changed queries to insert new snapshots without conflict handling
   - Added queries to retrieve snapshots by timestamp
   - Added queries to retrieve all snapshots for a URL
   - Added queries to retrieve snapshots in a date range

2. **Context-Aware Database Methods**:
   - `SaveSnapshot`: Saves a new snapshot with the current timestamp using a context
   - `GetLatestSnapshot`: Retrieves the most recent snapshot for a URL using a context
   - `GetSnapshotAtTimestamp`: Retrieves the nearest snapshot at or before a given timestamp using a context
   - `GetAllSnapshotsForURL`: Retrieves all snapshots for a URL using a context
   - `GetSnapshotsByDateRange`: Retrieves snapshots within a date range using a context

3. **Backward Compatibility**:
   - The `OverwriteSnapshot` method has been maintained for backward compatibility
   - It now delegates to `SaveSnapshot`, effectively creating a new version instead of overwriting

### Utility Scripts

A new utility script `snapshot_history.sh` has been created to demonstrate the versioned snapshot functionality:

- Retrieve the latest snapshot for a URL
- Retrieve a snapshot at a specific point in time
- Retrieve all snapshots for a URL
- Retrieve snapshots within a date range

### Usage Examples

```bash
# Get the latest snapshot
./snapshot_history.sh -u gemini://example.com/

# Get a snapshot from a specific point in time
./snapshot_history.sh -u gemini://example.com/ -t 2023-05-01T12:00:00Z

# Get all snapshots for a URL
./snapshot_history.sh -u gemini://example.com/ -a

# Get snapshots in a date range
./snapshot_history.sh -u gemini://example.com/ -r 2023-01-01T00:00:00Z 2023-12-31T23:59:59Z
```

### API Usage Examples

```go
// Save a new snapshot
ctx := context.Background()
snapshot, _ := snapshot.SnapshotFromURL("gemini://example.com", true)
tx, _ := Database.NewTx(ctx)
err := Database.SaveSnapshot(ctx, tx, snapshot)
tx.Commit()

// Get the latest snapshot
ctx := context.Background()
tx, _ := Database.NewTx(ctx)
latestSnapshot, err := Database.GetLatestSnapshot(ctx, tx, "gemini://example.com")
tx.Commit()

// Get a snapshot at a specific time
ctx := context.Background()
timestamp := time.Date(2023, 5, 1, 12, 0, 0, 0, time.UTC)
tx, _ := Database.NewTx(ctx)
historicalSnapshot, err := Database.GetSnapshotAtTimestamp(ctx, tx, "gemini://example.com", timestamp)
tx.Commit()

// Get all snapshots for a URL
ctx := context.Background()
tx, _ := Database.NewTx(ctx)
allSnapshots, err := Database.GetAllSnapshotsForURL(ctx, tx, "gemini://example.com")
tx.Commit()

// Using a timeout context to limit database operations
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
tx, _ := Database.NewTx(ctx)
latestSnapshot, err := Database.GetLatestSnapshot(ctx, tx, "gemini://example.com")
tx.Commit()
```

### Optimizations

1. **Content Deduplication**: The crawler can avoid storing duplicate content in the database. When content deduplication is enabled (by setting `--skip-identical-content=true`), the system implements two key behaviors:

   * **Content-based deduplication**: It compares the content with the latest existing snapshot for the same URL before saving. If the content is identical, the new snapshot is skipped, saving storage space. However, if the content has changed, a new snapshot is still created, preserving the version history.

   When deduplication is disabled (default, `--skip-identical-content=false`), the system stores every snapshot regardless of content similarity and may re-queue URLs that already have snapshots, leading to more frequent re-crawling of all content.

   This approach ensures that the version history for URLs with changing content is always preserved, regardless of the flag setting. The flag only controls whether to store snapshots when content hasn't changed.

2. **Time-based Crawl Frequency Control**: The crawler can be configured to skip re-crawling URLs that have been recently updated, using the `--skip-if-updated-days=N` parameter:

   * When set to a positive integer N, URLs that have a snapshot newer than N days ago will not be added to the crawl queue, even if they're found as links in other pages.

   * This feature helps control crawl frequency, ensuring that resources aren't wasted on frequently checking content that rarely changes.

   * Setting `--skip-if-updated-days=0` (the default) disables this feature, meaning all discovered URLs will be queued for crawling regardless of when they were last updated.

   * For example, `--skip-if-updated-days=7` will skip re-crawling any URL that has been crawled within the last week.

### Future Improvements

1. Add a web interface to browse snapshot history
2. Implement comparison features to highlight changes between snapshots
3. Add metadata to track crawl batches
4. Implement retention policies to manage storage