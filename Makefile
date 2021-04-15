PACKAGE := miniflux-indexer
MODULE_NAME := github.com/QuantumGhost/$(PACKAGE)
COMMIT_ID := $(shell git rev-parse HEAD)$(shell ./scripts/is-dirty.sh)
VERSION :=
BUILD_ARGS := \
    -ldflags "-X '$(MODULE_NAME)/internal.version=$(VERSION)' -X '$(MODULE_NAME)/internal.commitID=$(COMMIT_ID)'" \
    -tags "osusergo netgo" \
    -trimpath
EXTRA_BUILD_ARGS =
OUTPUT_DIR := out
OUTPUT_FILE := $(OUTPUT_DIR)/$(PACKAGE)
MAIN_FILE := cmd/$(PACKAGE)/main.go


.PHONY: fotmat build check-style lint install-linter clean package migration install-deps dev release

format:
	@go fmt ./...

build:
	go build $(BUILD_ARGS) $(EXTRA_BUILD_ARGS) -o $(OUTPUT_FILE) $(MAIN_FILE)

lint:
	@golangci-lint run ./...

check-style:
	@golangci-lint run --disable-all -E gofmt ./...

clean:
	find . -name '*.fail' -delete
	rm -f $(OUTPUT_FILE)

dev:
	$(MAKE) build -e VERSION=dev -e EXTRA_BUILD_ARGS="-tags dev"
