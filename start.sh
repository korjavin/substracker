#!/bin/bash
set -e

if [ ! -f .env ]; then
  echo "No .env file found. Copy .env.example to .env and fill in values."
  echo "  cp .env.example .env"
  exit 1
fi

set -a
source .env
set +a

export DB_PATH="${DB_PATH:-data.db}"
export PORT="${PORT:-8080}"

echo "Starting SubsTracker on :$PORT ..."
go run ./cmd/server
