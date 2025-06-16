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

## Error handling
- `xerrors` library is used for error creation/wrapping.
- The "Fatal" field is not used, we _always_ panic on fatal errors.
- _All_ internal functions _must_ return `xerror` errors.
- _All_ external errors are wrapped within `xerror` errors.


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

### Content Deduplication Strategy

The crawler implements a sophisticated content deduplication strategy that balances storage efficiency with comprehensive historical tracking:

#### `--skip-identical-content` Flag Behavior

**When `--skip-identical-content=true` (default)**:
- All content types are checked for duplicates before storing
- Identical content is skipped entirely to save storage space
- Only changed content results in new snapshots
- Applies to both Gemini and non-Gemini content uniformly

**When `--skip-identical-content=false`**:
- **Gemini content (`text/gemini` MIME type)**: Full historical tracking - every crawl creates a new snapshot regardless of content changes
- **Non-Gemini content**: Still deduplicated - identical content is skipped even when flag is false
- Enables comprehensive version history for Gemini capsules while avoiding unnecessary storage of duplicate static assets

#### Implementation Details

The deduplication logic is implemented in `shouldSkipIdenticalSnapshot()` function in `common/worker.go`:

1. **Primary Check**: When `--skip-identical-content=true`, all content is checked for duplicates
2. **MIME-Type Specific Check**: When the flag is false, only non-`text/gemini` content is checked for duplicates
3. **Content Comparison**: Uses `IsContentIdentical()` which compares either GemText fields or binary Data fields
4. **Dual Safety Checks**: Content is checked both in the worker layer and database layer for robustness

This approach ensures that Gemini capsules get complete version history when desired, while preventing storage bloat from duplicate images, binaries, and other static content.

### Time-based Crawl Frequency Control

The crawler can be configured to skip re-crawling URLs that have been recently updated, using the `--skip-if-updated-days=N` parameter:

* When set to a positive integer N, URLs that have a snapshot newer than N days ago will not be added to the crawl queue, even if they're found as links in other pages.
* This feature helps control crawl frequency, ensuring that resources aren't wasted on frequently checking content that rarely changes.
* Setting `--skip-if-updated-days=0` disables this feature, meaning all discovered URLs will be queued for crawling regardless of when they were last updated.
* Default value is 60 days.
* For example, `--skip-if-updated-days=7` will skip re-crawling any URL that has been crawled within the last week.

### Worker Pool Architecture

The crawler uses a sophisticated worker pool system with backpressure control:

* **Buffered Channel**: Job queue size equals the number of workers (`NumOfWorkers`)
* **Self-Regulating**: Channel backpressure naturally rate-limits the scheduler
* **Context-Aware**: Each URL gets its own context with timeout (default 120s)
* **Transaction Per Job**: Each worker operates within its own database transaction
* **SafeRollback**: Uses `gemdb.SafeRollback()` for graceful transaction cleanup on errors

### Database Transaction Patterns

* **Context Separation**: Scheduler uses long-lived context, while database operations use fresh contexts
* **Timeout Prevention**: Fresh `dbCtx := context.Background()` prevents scheduler timeouts from affecting DB operations
* **Error Handling**: Distinguishes between context cancellation, fatal errors, and recoverable errors

### Future Improvements

1. Add a web interface to browse snapshot history
2. Implement comparison features to highlight changes between snapshots
3. Add metadata to track crawl batches
4. Implement retention policies to manage storage