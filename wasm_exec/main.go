package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), chromedp.Headless, chromedp.NoSandbox)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	a := NewAuthorizer(sha256.New)

	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s program.wasm ( args... )\n", os.Args[0])
		os.Exit(2)
	}

	wasmBinary := flag.Arg(0)

	gopath, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cmd := exec.Command(gopath, "env", "GOROOT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n%s\n", err, string(out))
		os.Exit(1)
	}

	wasmExec := filepath.Join(strings.TrimSpace(string(out)), "lib", "wasm", "wasm_exec.js")

	s := httptest.NewServer(HTTPHandler(a, wasmBinary, wasmExec))

	jsonArgs, err := json.Marshal(args)
	if err != nil {
		panic(err)

	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	wd = path.Join(".", filepath.ToSlash(wd))
	jsonWd, err := json.Marshal(wd)
	if err != nil {
		panic(err)
	}

	chromedp.ListenTarget(ctx, func(ev any) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			for _, arg := range ev.Args {
				var out any
				err := json.Unmarshal(arg.Value, &out)
				if err != nil {
					fmt.Println(arg.Value)
				} else {
					fmt.Println(out)
				}
			}
		case *runtime.EventExceptionThrown:
			fmt.Fprintf(os.Stderr, "exception: %s\n", ev.ExceptionDetails.Error())

		case *fetch.EventRequestPaused:
			token, err := a.Issue()
			if err != nil {
				panic(err)
			}

			req := fetch.ContinueRequest(ev.RequestID)

			if strings.HasPrefix(ev.Request.URL, s.URL) {
				req.Headers = append(req.Headers, &fetch.HeaderEntry{
					Name:  TokenHeader,
					Value: token.String(),
				})
			}
			go func() {
				chromedp.Run(ctx, req)
			}()
		}
	})

	var result int

	if err := chromedp.Run(
		ctx,
		fetch.Enable().WithHandleAuthRequests(false),
		chromedp.Navigate(s.URL+"/index.html"),
		chromedp.Evaluate(fmt.Sprintf("const goargs=%s;", string(jsonArgs)), nil),
		chromedp.Evaluate(fmt.Sprintf("const hostwd=%s;", string(jsonWd)), nil),
		chromedp.Evaluate(
			"run();", &result,
			func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
				return p.WithAwaitPromise(true)
			}),
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	chromedp.Cancel(ctx)
	os.Exit(result)
}
