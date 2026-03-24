//go:build js

package main

import (
	"context"
	"flag"
	"os"

	"github.com/CCorderZugcat/zugoui/browser"
	_ "github.com/CCorderZugcat/zugoui/samples/pizza/model"
)

func main() {
	ctx := context.Background()

	flag.Parse()

	err := browser.Main(ctx, flag.Arg(0)) // if empty, defaults to "rpc"
	os.Stdout.Write([]byte("debug: exiting\n"))

	if err != nil {
		os.Exit(1)
	}
}
