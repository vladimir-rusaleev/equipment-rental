#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -f .env ]]; then set -a; source .env; set +a; fi
: "${SERVICE_KEY:?copy .env.example to .env and configure SERVICE_KEY}"
export SERVICE_KEY
export SEED_DEMO="${SEED_DEMO:-true}"
pids=()
cleanup() { trap - EXIT INT TERM; for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done; wait || true; }
trap cleanup EXIT INT TERM
for service in inventory; do
 ./bin/"$service" &
 pids+=("$!")
done
wait -n
