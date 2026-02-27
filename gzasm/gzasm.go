package gzasm

import (
	"io"
	"io/fs"
	"iter"
	"maps"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// GZasm rewrites before and after file server handler.
// If there is a gzip'd or br'd wasm version available,
// return it instead if we are able to.
type GZasm struct {
	next http.Handler
	fsys fs.FS
}

var extensions = map[string]string{
	"gzip": ".gz",
	"br":   ".br",
}

var preferred = []string{"br", "gzip"}

// New creates a new GZasm handler. next is the handler we are wrapping,
// fsys is the filesystem from which we are serving.
func New(next http.Handler, fsys fs.FS) *GZasm {
	return &GZasm{next: next, fsys: fsys}
}

// ServeHTTP is the [http.Handler]
func (z GZasm) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if encoding, st := z.canServe(r); st != nil {
		r := r.Clone(r.Context())
		z.serveGz(st, encoding, w, r)
		return
	}
	z.next.ServeHTTP(w, r)
}

// for each element e, return a modified version ee, ok = true to be included in the output.
// if f returns ok = false, the element is not included.
func filter[E any](in iter.Seq[E], f func(e E) (ee E, ok bool)) iter.Seq[E] {
	return func(yield func(e E) bool) {
		for e := range in {
			if e, ok := f(e); ok {
				if !yield(e) {
					return
				}
			}
		}
	}
}

// convert a sequence of E into a sequence of E,struct{}{}, useful for building
// maps used as sets.
func setOf[E any](in iter.Seq[E]) iter.Seq2[E, struct{}] {
	return func(yield func(e E, ok struct{}) bool) {
		for i := range in {
			if !yield(i, struct{}{}) {
				return
			}
		}
	}
}

func (z GZasm) canServe(r *http.Request) (encoding string, st fs.FileInfo) {
	if r.Method != http.MethodGet {
		return "", nil // not GETing the resource
	}
	if path.Ext(r.URL.Path) != ".wasm" {
		return "", nil // not a wasm file
	}
	if r.Header.Get("Range") != "" {
		return "", nil // no ranged requests
	}

	acceptEncoding := slices.Values(strings.Split(r.Header.Get("Accept-Encoding"), ","))
	accepts := maps.Collect(
		setOf(
			filter(acceptEncoding, func(e string) (string, bool) {
				e = strings.TrimSpace(e)
				return e, e != ""
			}),
		),
	)

	for _, encoding = range preferred {
		if _, ok := accepts[encoding]; !ok {
			// client does not expect this one
			continue
		}

		st, err := fs.Stat(z.fsys, filepath.FromSlash(path.Join(".", r.URL.Path+extensions[encoding])))
		if err != nil {
			// we don't have this one
			continue
		}

		return encoding, st
	}

	// no overlap between what client expects and what we have
	return "", nil
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

// WriteHeader asserts our headers are included in a successful response
func (wr *writer) WriteHeader(statusCode int) {
	wr.statusCode = statusCode
	wr.assertHeaders()
	wr.ResponseWriter.WriteHeader(statusCode)

}

// Write asserts our headers are included in a successful response,
// if WriteHeaders was not called before Write.
func (wr *writer) Write(buf []byte) (int, error) {
	wr.assertHeaders()
	return wr.ResponseWriter.Write(buf)
}

type readerFrom struct {
	*writer
	io.ReaderFrom
}

// ReadFrom implements [io.ReaderFrom] if the original writer does.
// Also asserts our headers are included in a successful response.
func (rf *readerFrom) ReadFrom(r io.Reader) (int64, error) {
	rf.writer.assertHeaders()
	return rf.ReaderFrom.ReadFrom(r)
}

func (z GZasm) serveGz(st fs.FileInfo, encoding string, w http.ResponseWriter, r *http.Request) {
	r.URL.Path = r.URL.Path + extensions[encoding]

	wr := &writer{ResponseWriter: w}

	// post handler header assertions, if the response is http.StatusOK
	wr.h = make(http.Header)
	wr.h.Set("Content-Type", "application/wasm")
	wr.h.Set("Content-Length", strconv.Itoa(int(st.Size())))
	wr.h.Set("Content-Encoding", encoding)

	if rf, ok := w.(io.ReaderFrom); ok {
		// do not defeat the ReaderFrom if the writer happens to be one
		w = &readerFrom{writer: wr, ReaderFrom: rf}
	} else {
		w = wr
	}

	z.next.ServeHTTP(w, r)
}
