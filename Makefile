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
	go test ./...

run: build
	${BIN_DIR}/server

# Requires that protobuf be installed: https://twitchtv.github.io/twirp/docs/install.html
protos:
	protoc --proto_path=. --go_out=paths=source_relative:. internal/core/proto/archon.proto
	protoc --proto_path=. --go_out=paths=source_relative:. --twirp_out=paths=source_relative:. internal/shipgate/shipgate.proto
