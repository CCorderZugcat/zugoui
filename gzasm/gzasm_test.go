package gzasm_test

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/gzasm"
	"github.com/andybalholm/brotli"
)

func TestGZasm(t *testing.T) {
	data := make([]byte, 4096)
	rand.Read(data)

	fsys := make(fstest.MapFS)
	fsys["test.wasm"] = &fstest.MapFile{
		Data:    data,
		Mode:    0555,
		ModTime: time.Now(),
	}

	gzdata := &bytes.Buffer{}
	gzw := gzip.NewWriter(gzdata)
	gzw.Write(data)
	gzw.Close()

	brdata := &bytes.Buffer{}
	brw := brotli.NewWriter(brdata)
	brw.Write(data)
	brw.Close()

	fsys["test.wasm.gz"] = &fstest.MapFile{
		Data:    gzdata.Bytes(),
		Mode:    0555,
		ModTime: time.Now(),
	}

	fsys["test.wasm.br"] = &fstest.MapFile{
		Data:    brdata.Bytes(),
		Mode:    0555,
		ModTime: time.Now(),
	}

	s := httptest.NewServer(gzasm.New(http.FileServerFS(fsys), fsys))
	defer s.Close()

	roundTrip := func(encoding string, expect []byte) {
		r, err := http.NewRequest(http.MethodGet, s.URL+"/test.wasm", nil)
		require.NoError(t, err)

		if encoding != "" {
			r.Header.Add("Accept-Encoding", "gzip, br")
		}

		res, err := http.DefaultClient.Do(r)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		if encoding != "" {
			assert.Equal(t, encoding, res.Header.Get("Content-Encoding"))
		}

		body := &bytes.Buffer{}
		io.Copy(body, res.Body)
		assert.Truef(t, bytes.Equal(expect, body.Bytes()), "contents did not match")
	}

	t.Run("default", func(t *testing.T) {
		roundTrip("", data)
	})

	t.Run("br", func(t *testing.T) {
		roundTrip("br", brdata.Bytes())
	})

	t.Run("gz", func(t *testing.T) {
		delete(fsys, "test.wasm.br")
		roundTrip("gzip", gzdata.Bytes())
	})
}
