.PHONY: setup lint lint-fix test build

setup:
	git config core.hooksPath .githooks

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

test:
	go test ./...

build: setup
	go build -o vau .
