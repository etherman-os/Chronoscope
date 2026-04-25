#!/usr/bin/env bash
set -euo pipefail

MIGRATIONS_DIR="$(cd "$(dirname "$0")/.." && pwd)/migrations"

if [ -z "${DATABASE_URL:-}" ]; then
    echo "ERROR: DATABASE_URL environment variable is required"
    exit 1
fi

case "${1:-}" in
    up)
        migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" up
        ;;
    down)
        migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" down 1
        ;;
    force)
        migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" force "$2"
        ;;
    version)
        migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" version
        ;;
    *)
        echo "Usage: $0 {up|down|force <version>|version}"
        exit 1
        ;;
esac
