#!/bin/sh

GOOS=js GOARCH=wasm go build -o execme.wasm
