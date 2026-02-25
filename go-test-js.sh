#!/bin/sh
here="$(dirname "$0")"
testExec="$here/wasm_exec/wasm_exec"

if [ ! -f "$testExec" ]; then
    go build -o "$testExec" "$here/wasm_exec"
fi

GOOS=js GOARCH=wasm go test -exec "$(pwd)/$testExec" "$@"
exit $?
