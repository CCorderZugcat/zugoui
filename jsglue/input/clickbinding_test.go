//go:build js

package input_test

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/jsglue/input"
	"github.com/CCorderZugcat/zugoui/jsglue/jsgluetest"
)

//go:embed testdata/*
var fsys embed.FS

func TestClickBinding(t *testing.T) {
	jsgluetest.SetBody(t, fsys, "click.html")

	elem, err := input.Element("button1")
	require.NoError(t, err)

	action := make(chan string, 1)

	b := input.NewClickBinding(elem, "button1", func(name string) {
		action <- name
	})
	defer b.Destroy()

	elem.Call("click")
	assert.Equal(t, "button1", <-action)
	t.Log("got click")
}
