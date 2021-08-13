#!/bin/bash
set -e

if ! command -v protoc >/dev/null 2>&1 ; then
    echo "Please install protoc."
    echo "brew install protoc"
    exit 1
fi

if ! command -v protoc-gen-go >/dev/null 2>&1 ; then
    echo "Please install protoc-gen-go."
    echo "brew install protoc-gen-go"
    exit 1
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1 ; then
    echo "Please install protoc-gen-go-grpc."
    echo "brew install protoc-gen-go-grpc"
    exit 1
fi

protoc \
  --proto_path=internal/shipgate/api \
  --go_out=internal/shipgate/api \
  --go-grpc_out=internal/shipgate/api \
  internal/shipgate/api/api.proto