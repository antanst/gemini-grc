#!/bin/env bash
set -eu
set -o pipefail

# Max response size 10MiB
LOG_LEVEL=debug \
PRINT_WORKER_STATUS=false \
DRY_RUN=false \
NUM_OF_WORKERS=1 \
WORKER_BATCH_SIZE=1 \
BLACKLIST_PATH="$(pwd)/blacklist.txt" \
MAX_RESPONSE_SIZE=10485760 \
RESPONSE_TIMEOUT=10 \
PANIC_ON_UNEXPECTED_ERROR=true \
go run ./bin/gemget/main.go "$@"

