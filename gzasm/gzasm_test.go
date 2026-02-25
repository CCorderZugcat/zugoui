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

	fsys["test.wasm.gz"] = &fstest.MapFile{
		Data:    gzdata.Bytes(),
		Mode:    0555,
		ModTime: time.Now(),
	}

	s := httptest.NewServer(gzasm.New(http.FileServerFS(fsys), fsys))
	defer s.Close()

	r, err := http.NewRequest(http.MethodGet, s.URL+"/test.wasm", nil)
	require.NoError(t, err)

	r.Header.Add("Accept-Encoding", "gzip")

	res, err := http.DefaultClient.Do(r)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	assert.Equal(t, "application/wasm", res.Header.Get("Content-Type"))
	assert.Equal(t, "gzip", res.Header.Get("Content-Encoding"))

	buf := &bytes.Buffer{}
	io.Copy(buf, res.Body)
	res.Body.Close()

	assert.Equal(t, gzdata.Bytes(), buf.Bytes())

	r, err = http.NewRequest(http.MethodGet, s.URL+"/test.wasm", nil)
	require.NoError(t, err)

	res, err = http.DefaultClient.Do(r)
	require.NoError(t, err)

	assert.Equal(t, "application/wasm", res.Header.Get("Content-Type"))

	buf.Reset()
	io.Copy(buf, res.Body)
	res.Body.Close()

	assert.Equal(t, data, buf.Bytes())
}
