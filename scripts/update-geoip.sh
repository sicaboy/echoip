#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p data
# Serialize downloads and restart across manual and scheduled runs.
exec 9>data/.update.lock
flock -n 9 || exit 0
docker compose run --rm --build updater
docker compose up -d --no-build --no-deps --force-recreate echoip
