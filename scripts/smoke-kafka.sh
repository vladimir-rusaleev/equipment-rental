#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -f .env ]]; then set -a; source .env; set +a; fi
export KAFKA_TOPIC="rent-smoke-$$"
export KAFKA_GROUP="rent-smoke-$$"
export KAFKA_BROKERS="127.0.0.1:${KAFKA_PORT:-19092}"
cleanup() {
 docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 --delete --topic "$KAFKA_TOPIC" >/dev/null 2>&1 || true
}
trap cleanup EXIT
docker compose up -d --wait kafka
docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh \
 --bootstrap-server localhost:9092 --create --topic "$KAFKA_TOPIC" --partitions 1 --replication-factor 1
RENT_TEST_TRANSPORT=kafka RENT_TEST_STORAGE=memory python3 scripts/smoke.py
