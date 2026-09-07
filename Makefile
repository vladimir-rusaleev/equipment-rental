GO ?= go
SERVICES = inventory
.PHONY: build test run run-memory generate
build:
	@mkdir -p bin
	@for service in $(SERVICES); do $(GO) build -buildvcs=false -o bin/$$service ./$$service/cmd/app || exit 1; done
test:
	$(GO) test -buildvcs=false -race ./...
generate:
	protoc -I api --go_out=. --go_opt=module=rent --go-grpc_out=. --go-grpc_opt=module=rent api/rent/v1/rent.proto
run: build
	./scripts/run-local.sh

run-memory: run
