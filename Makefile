DOCKER      ?= docker
IMAGE       ?= mcp-gedcom
VERSION     ?= $(shell v=$$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//'); echo "$${v:-dev}")
GEDCOM_FILE ?= $(CURDIR)/gedcom.ged
BUILD_ARGS  ?= --build-arg VERSION=$(VERSION)

# Test stages always re-run and stream their output. Override with e.g.
#   make test-integration INTEGRATION_NO_CACHE=
TEST_BUILD_FLAGS    ?= --progress=plain
UNIT_NO_CACHE       ?= --no-cache-filter unit
INTEGRATION_NO_CACHE ?= --no-cache-filter integration

.DEFAULT_GOAL := all

.PHONY: all doctor image build test test-unit test-integration ci run run-sample shell clean help

all: ci

doctor:
	@$(DOCKER) version >/dev/null 2>&1 || { echo "✗ Docker daemon unavailable"; exit 1; }
	@echo "✓ Docker $$($(DOCKER) version --format '{{.Server.Version}}')"
	@echo "✓ Version $(VERSION)"

image build:
	$(DOCKER) build $(BUILD_ARGS) --target runtime -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

test-unit:
	$(DOCKER) build $(BUILD_ARGS) $(TEST_BUILD_FLAGS) $(UNIT_NO_CACHE) --target unit .

test-integration:
	$(DOCKER) build $(BUILD_ARGS) $(TEST_BUILD_FLAGS) $(INTEGRATION_NO_CACHE) --target integration .

test: test-unit test-integration

ci: test image

run: image
	@test -f "$(abspath $(GEDCOM_FILE))" || { echo "✗ GEDCOM file not found: $(abspath $(GEDCOM_FILE))"; exit 1; }
	$(DOCKER) run --rm -i -v "$(abspath $(GEDCOM_FILE)):/app/gedcom.ged:ro" $(IMAGE):latest

run-sample: GEDCOM_FILE := $(CURDIR)/sample/simpsons.ged
run-sample: run

shell:
	$(DOCKER) build $(BUILD_ARGS) --target source -t $(IMAGE):dev .
	$(DOCKER) run --rm -it -v "$(CURDIR):/src" -w /src $(IMAGE):dev sh

clean:
	-$(DOCKER) image rm -f $(IMAGE):$(VERSION) $(IMAGE):latest $(IMAGE):dev

help:
	@echo "Usage:"
	@echo "  make               Default: ci (tests + runtime image)"
	@echo "  make doctor        Check Docker availability"
	@echo "  make image         Build the runtime Docker image"
	@echo "  make build         Alias for 'make image'"
	@echo "  make test-unit     Run unit tests in Docker"
	@echo "  make test-integration  Run integration tests in Docker (streams output, no cache)"
	@echo "  make test          Run unit + integration tests in Docker"
	@echo "  make ci            Run tests and build the runtime image"
	@echo "  make run           Run the image with GEDCOM_FILE mounted"
	@echo "  make run-sample    Run the image with sample/simpsons.ged"
	@echo "  make shell         Open a Go shell with the source mounted"
	@echo "  make clean         Remove tagged Docker images"
	@echo ""
	@echo "Test targets always re-run their stage and stream output."
	@echo "Re-enable caching with: make test-integration INTEGRATION_NO_CACHE="
