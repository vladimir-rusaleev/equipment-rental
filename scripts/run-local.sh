#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
storage_override="${STORAGE:-}"
transport_override="${NOTIFICATION_TRANSPORT:-}"
if [[ -f .env ]]; then set -a; source .env; set +a; fi
if [[ -n "$storage_override" ]]; then export STORAGE="$storage_override"; fi
if [[ -n "$transport_override" ]]; then export NOTIFICATION_TRANSPORT="$transport_override"; fi
: "${SERVICE_KEY:?copy .env.example to .env and configure SERVICE_KEY}"
: "${ADMIN_EMAIL:?configure ADMIN_EMAIL}"
: "${ADMIN_PASSWORD:?configure ADMIN_PASSWORD}"
export SERVICE_KEY ADMIN_EMAIL ADMIN_PASSWORD
export SEED_DEMO="${SEED_DEMO:-true}"
pids=()
cleanup() { trap - EXIT INT TERM; for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done; wait || true; }
trap cleanup EXIT INT TERM
for service in inventory notifier rental gateway; do
 case "$service" in
  rental) metrics_address="${RENTAL_METRICS_LISTEN:-127.0.0.1:9091}" ;;
  inventory) metrics_address="${INVENTORY_METRICS_LISTEN:-127.0.0.1:9092}" ;;
  notifier) metrics_address="${NOTIFIER_METRICS_LISTEN:-127.0.0.1:9093}" ;;
  gateway) metrics_address="" ;;
 esac
 METRICS_LISTEN="$metrics_address" ./bin/"$service" &
 pids+=("$!")
done
wait -n
