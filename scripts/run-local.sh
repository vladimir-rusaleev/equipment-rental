#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
storage_override="${STORAGE:-}"
if [[ -f .env ]]; then set -a; source .env; set +a; fi
if [[ -n "$storage_override" ]]; then export STORAGE="$storage_override"; fi
: "${SERVICE_KEY:?copy .env.example to .env and configure SERVICE_KEY}"
: "${ADMIN_EMAIL:?configure ADMIN_EMAIL}"
: "${ADMIN_PASSWORD:?configure ADMIN_PASSWORD}"
export SERVICE_KEY ADMIN_EMAIL ADMIN_PASSWORD
export SEED_DEMO="${SEED_DEMO:-true}"
pids=()
cleanup() { trap - EXIT INT TERM; for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done; wait || true; }
trap cleanup EXIT INT TERM
for service in inventory notifier rental gateway; do
 ./bin/"$service" &
 pids+=("$!")
done
wait -n
