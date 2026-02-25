package main

import (
	"embed"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
)

//go:embed site/*
var sitefs embed.FS

func getFile(filename string, w http.ResponseWriter, r *http.Request) {
	fd, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v: %v\n", r.URL, err)

		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "%s %v\r\n", http.StatusText(http.StatusNotFound), err)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v\r\n", err)
		return
	}
	defer fd.Close()

	if mimeType := mime.TypeByExtension(path.Ext(r.URL.Path)); mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}

	io.Copy(w, fd)
}

func HTTPHandler[H hash.Hash](a *Authorizer[H], wasmBinary, wasmExec string) http.Handler {
	siteDir, err := fs.Sub(sitefs, "site")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(siteDir))
	mux.HandleFunc("GET /wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		getFile(wasmExec, w, r)
	})
	mux.HandleFunc("GET /binary.wasm", func(w http.ResponseWriter, r *http.Request) {
		getFile(wasmBinary, w, r)
	})

	return a.Handler(mux)

}
