//go:build js

package main

import (
	"fmt"
	"os"
)

//go:generate ./build.sh

func main() {
	fmt.Println("hello, wasm:", os.Args)
}
