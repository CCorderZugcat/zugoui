package main_test

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	main "github.com/CCorderZugcat/zugoui/wasm_exec"
)

func TestAuth(t *testing.T) {
	a := main.NewAuthorizer(sha256.New)

	t1, err := a.Issue()
	require.NoError(t, err)

	t2, err := a.Issue()
	require.NoError(t, err)

	err = a.Verify(t1)
	assert.NoError(t, err)
	err = a.Verify(t1)
	t.Logf("%v (error expected)", err)
	assert.Error(t, err)
	err = a.Verify(t2)
	assert.NoError(t, err)
}

func TestAuthHandler(t *testing.T) {
	a := main.NewAuthorizer(sha256.New)

	token, err := a.Issue()
	require.NoError(t, err)

	w := &httptest.ResponseRecorder{}
	r := httptest.NewRequest(http.MethodGet, "http://www.example.com", nil)
	r.Header.Set(main.TokenHeader, token.String())

	ok := false
	a.Handler(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { ok = true })).ServeHTTP(w, r)
	assert.True(t, ok)

	ok = false
	a.Handler(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { ok = true })).ServeHTTP(w, r)
	assert.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
