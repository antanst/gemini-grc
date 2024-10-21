#!/bin/sh
set -eu

MAX_RESPONSE_SIZE=10485760 \
LOG_LEVEL=info \
ROOT_PATH=./snaps \
RESPONSE_TIMEOUT=10 \
NUM_OF_WORKERS=5 \
PG_DATABASE=gemini \
PG_HOST=127.0.0.1 \
PG_PORT=5433 \
PG_USER=gemini \
PG_PASSWORD=gemini \
go run ./migrate1_host.go
