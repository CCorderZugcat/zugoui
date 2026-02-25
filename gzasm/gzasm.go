package gzasm

import (
	"io"
	"io/fs"
	"maps"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// GZasm rewrites before and after file server handler.
// If there is a gzip'd wasm version available,
// return it instead if we are able to.
type GZasm struct {
	next http.Handler
	fsys fs.FS
}

// New creates a new GZasm handler
func New(next http.Handler, fsys fs.FS) *GZasm {
	return &GZasm{next: next, fsys: fsys}
}

func (z GZasm) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if st := z.canServeGz(r); st != nil {
		r := r.Clone(r.Context())
		z.serveGz(st, w, r)
		return
	}
	z.next.ServeHTTP(w, r)
}

func (z GZasm) canServeGz(r *http.Request) fs.FileInfo {
	if r.Method != http.MethodGet {
		return nil // not GETing the resource
	}
	if path.Ext(r.URL.Path) != ".wasm" {
		return nil // not a wasm file
	}
	if r.Header.Get("Range") != "" {
		return nil // no ranged requests
	}
	encodings := strings.Split(r.Header.Get("Accept-Encoding"), ",")
	if slices.IndexFunc(encodings, func(e string) bool {
		return strings.TrimSpace(e) == "gzip"
	}) < 0 {
		return nil // no Accept-Encoding including gzip
	}
	st, err := fs.Stat(z.fsys, filepath.FromSlash(path.Join(".", r.URL.Path+".gz")))
	if err != nil {
		return nil // we don't have a file.gz to give
	}
	return st
}

type writer struct {
	http.ResponseWriter
	statusCode int
	h          http.Header
}

func (wr *writer) assertHeaders() {
	if wr.statusCode == 0 {
		wr.statusCode = http.StatusOK
	}
	if wr.statusCode == http.StatusOK {
		maps.Copy(wr.Header(), wr.h)
	}
}

func (wr *writer) WriteHeader(statusCode int) {
	wr.statusCode = statusCode
	wr.assertHeaders()
	wr.ResponseWriter.WriteHeader(statusCode)

}

func (wr *writer) Write(buf []byte) (int, error) {
	wr.assertHeaders()
	return wr.Write(buf)
}

type readerFrom struct {
	*writer
	io.ReaderFrom
}

func (rf *readerFrom) ReadFrom(r io.Reader) (int64, error) {
	rf.writer.assertHeaders()
	return rf.ReaderFrom.ReadFrom(r)
}

func (z GZasm) serveGz(st fs.FileInfo, w http.ResponseWriter, r *http.Request) {
	r.URL.Path = r.URL.Path + ".gz"

	wr := &writer{ResponseWriter: w}

	// post handler header assertions
	wr.h = make(http.Header)
	wr.h.Set("Content-Type", "application/wasm")
	wr.h.Set("Content-Length", strconv.Itoa(int(st.Size())))
	wr.h.Set("Content-Encoding", "gzip")

	if rf, ok := w.(io.ReaderFrom); ok {
		w = &readerFrom{writer: wr, ReaderFrom: rf}
	} else {
		w = wr
	}

	z.next.ServeHTTP(w, r)
}
