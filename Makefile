CONFIG_PATH=/usr/local/etc/archon
BIN_DIR ?= bin
ANALYZER_DST ?= analyzer

.DEFAULT_TARGET := all
.PHONY: build lint test

all: build test

build:
	mkdir -p ${BIN_DIR}
	go build -o ${BIN_DIR}/archon ./cmd
	go build -o ${BIN_DIR} ./cmd/account
	go build -o ${BIN_DIR} ./cmd/analyzer
	go build -o ${BIN_DIR} ./cmd/certgen
	go build -o ${BIN_DIR} ./cmd/patcher

lint:
	golangci-lint run

test:
	go test ./...

protos:
	./gen_protos.sh

setup:
	./setup/setup.sh

# cd first so we're in the same dir as the config file
run: setup
	cd archon_server && ./${BIN_DIR}/archon

analyzer: build
	${BIN_DIR}/analyzer -auto -folder ${ANALYZER_DST} capture && \
		${BIN_DIR}/analyzer compact ${ANALYZER_DST}/*.session && \
		${BIN_DIR}/analyzer -collapse aggregate ${ANALYZER_DST}/*.session