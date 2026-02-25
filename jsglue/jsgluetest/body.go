//go:build js

package jsgluetest

import (
	"io"
	"io/fs"
	"path"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/require"
)

func readFile(t testing.TB, fsys fs.FS, filename string) []byte {
	t.Helper()

	fd, err := fsys.Open(path.Join("testdata", filename))
	require.NoError(t, err)
	defer fd.Close()

	out, err := io.ReadAll(fd)
	require.NoError(t, err)

	return out
}

// SetBody reads from your test's testdata directory for a body.
// Alos inserts the beforeend button needed by wasmbrowsertest
func SetBody(t testing.TB, fsys fs.FS, filename string) {
	t.Helper()

	pageData := readFile(t, fsys, filename)

	body := js.Global().Get("document").Get("body")
	body.Set("innerHTML", string(pageData))

	// unit test framework needs this button
	body.Call(
		"insertAdjacentHTML",
		"beforeend", `<button id="doneButton" style="display: none;" disabled>Done</button>`,
	)
}
