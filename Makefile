BIN_DIR ?= bin

.DEFAULT_TARGET := all
.PHONY: build lint test

all: build lint test

build:
	mkdir -p ${BIN_DIR}
	go build -o ${BIN_DIR} ./cmd/*

lint:
	golangci-lint run

test:
	go test ./internal/...

run: build
	${BIN_DIR}/archon
