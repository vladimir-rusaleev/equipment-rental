GO ?= go
SERVICES = gateway rental inventory notifier
.PHONY: build test run generate
build:
	@mkdir -p bin
	@for service in $(SERVICES); do $(GO) build -buildvcs=false -o bin/$$service ./$$service/cmd/app || exit 1; done
test:
	$(GO) test -buildvcs=false -race ./...
generate:
	protoc -I api --go_out=. --go_opt=module=rent --go-grpc_out=. --go-grpc_opt=module=rent api/rent/v1/rent.proto
run: build
	./scripts/run-local.sh

smoke: build
	python3 scripts/smoke.py

.PHONY: db-up up down run-memory test-postgres

db-up:
	docker compose up -d --wait postgres

up:
	docker compose up --build

down:
	docker compose down

run-memory: build
	STORAGE=memory ./scripts/run-local.sh

test-postgres: build
	GO="$(GO)" ./scripts/test-postgres.sh
