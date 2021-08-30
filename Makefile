CONFIG_PATH=/usr/local/etc/archon
BIN_DIR=bin
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

protos:
	./gen_protos.sh

run: build
	${BIN_DIR}/server -config ${CONFIG_PATH}

analyzer: build
	${BIN_DIR}/analyzer -auto -folder ${ANALYZER_DST} capture && \
		${BIN_DIR}/analyzer compact ${ANALYZER_DST}/*.session && \
		${BIN_DIR}/analyzer -collapse aggregate ${ANALYZER_DST}/*.session