# ==================================================================================== #
# HELPERS
# ==================================================================================== #

.PHONY: prepare
prepare:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.1.6
	go install github.com/vektra/mockery/v3@v3.4.0

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: no-dirty
no-dirty:
	git diff --exit-code


.PHONY: build
build:
	go build -o sync ./cmd/ba-sync/main.go

.PHONY: prepare
prepare:
	go env -w GOPRIVATE=github.com/arisu-archive/*

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## lint: run quality control checks
.PHONY: lint
lint:
	golangci-lint run ./...

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit: lint
	go mod verify
	go vet ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
