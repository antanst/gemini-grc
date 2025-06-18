# TODO

## Outstanding Issues

### 1. Ctrl+C Signal Handling Issue

**Problem**: The crawler sometimes doesn't exit properly when Ctrl+C is pressed.

**Root Cause**: The main thread gets stuck in blocking operations before it can check for signals:
- Database operations in the polling loop (`cmd/crawler/crawler.go:239-250`)
- Job queueing when channel is full (`jobs <- url` can block if workers are slow)
- Long-running database transactions

**Location**: `cmd/crawler/crawler.go` - main polling loop starting at line 233

**Solution**: Add signal/context checking to blocking operations:
- Use cancellable context instead of `context.Background()` for database operations
- Make job queueing non-blocking or context-aware
- Add timeout/cancellation to database operations

### 2. fetchSnapshotsFromHistory() Doesn't Work with --skip-identical-content=true

**Problem**: When `--skip-identical-content=true` (default), URLs with unchanged content get continuously re-queued.

**Root Cause**: The function tracks when content last changed, not when URLs were last crawled:
- Identical content → no new snapshot created
- Query finds old snapshot timestamp → re-queues URL
- Creates infinite loop of re-crawling unchanged content

**Location**: `cmd/crawler/crawler.go:388-470` - `fetchSnapshotsFromHistory()` function

**Solution Options**:
1. Add `last_crawled` timestamp to URLs table
2. Create separate `crawl_attempts` table  
3. Always create snapshot entries (even for duplicates) but mark them as such
4. Modify logic to work with existing schema constraints

**Current Status**: Function assumes `SkipIdenticalContent=false` per original comment at line 391.