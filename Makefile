CONFIG_PATH=/usr/local/etc/archon
BIN_DIR ?= bin
ANALYZER_DST ?= analyzer

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
	${BIN_DIR}/server -config ${CONFIG_PATH}

# Requires that protobuf be installed: https://twitchtv.github.io/twirp/docs/install.html
protos:
	protoc \
		--go_out=internal/shipgate \
		--twirp_out=internal/shipgate \
		internal/shipgate/api.proto

analyzer: build
	${BIN_DIR}/analyzer -auto -folder ${ANALYZER_DST} capture && \
		${BIN_DIR}/analyzer compact ${ANALYZER_DST}/*.session && \
		${BIN_DIR}/analyzer -collapse aggregate ${ANALYZER_DST}/*.session