//go:build js

package input_test

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/formtest"
	"github.com/CCorderZugcat/zugoui/input"
)

//go:embed testdata/*
var fsys embed.FS

func TestClickBinding(t *testing.T) {
	formtest.SetBody(t, fsys, "click.html")

	elem, err := input.Element("button1")
	require.NoError(t, err)

	action := make(chan string, 1)

	b := input.NewClickBinding("button1", "button1", func(name string) {
		action <- name
	})
	defer b.Destroy()

	elem.Call("click")
	assert.Equal(t, "button1", <-action)
	t.Log("got click")
}
