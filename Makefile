.PHONY: setup lint test build

setup:
	git config core.hooksPath .githooks

lint:
	golangci-lint run ./...

test:
	go test ./...

build: setup
	go build -o vau .
