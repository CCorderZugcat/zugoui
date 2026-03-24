#!/bin/sh
set -e

cd "$(dirname "$0")"
GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o client.wasm .

