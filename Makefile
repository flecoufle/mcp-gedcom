APP_NAME    ?= mcp-gedcom
BINARY      ?= $(APP_NAME)
GO          ?= go
CMD_PATH    ?= ./cmd/mcp-gedcom/server
GEDCOM_FILE ?= gedcom.ged
-include .env

VPC_HOST    ?=
USER_DIR    ?=

.PHONY: all build clean test test-unit test-integration vet run run-sample docker deps help publish

all: vet build test-unit

build:
	$(GO) build -o $(BINARY) $(CMD_PATH)

test-unit:
	$(GO) test ./...

test-integration: build
	./test.sh

test: test-unit test-integration

vet:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY) -gedcom-file sample/$(GEDCOM_FILE)

run-sample: build
	./$(BINARY) -gedcom-file sample/simpsons.ged

docker:
	docker build -t $(APP_NAME) .

deps:
	$(GO) mod download

publish:
	scp ./dist/$(APP_NAME) $(VPC_HOST):/home/$(USER_DIR)/services/pico/
	ssh $(VPC_HOST) "sudo chown root:root /home/$(USER_DIR)/services/pico/$(APP_NAME)"

help:
	@echo "Usage:"
	@echo "  make               Default: vet + build + test-unit"
	@echo "  make build         Build the server binary"
	@echo "  make test-unit     Run unit tests (go test ./...)"
	@echo "  make test          Run all tests (unit + integration)"
	@echo "  make vet           Run go vet"
	@echo "  make clean         Remove built artifacts"
	@echo "  make run           Run server with default GEDCOM"
	@echo "  make run-sample    Run server with simpsons.ged"
	@echo "  make docker        Build Docker image"
	@echo "  make deps          Download Go dependencies"
