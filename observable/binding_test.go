package observable_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
)

func TestFullDupluxBinding(t *testing.T) {
	var source, dest int
	sm := observable.NewModel(&source)
	dm := observable.NewModel(&dest)

	b, err := observable.NewBinding("value", dm, "value", sm, "")
	require.NoError(t, err)
	defer b.Release()

	sm.SetValue("value", 5)
	assert.Equal(t, 5, dest)

	dm.SetValue("value", 6)
	assert.Equal(t, 6, source)
}

func TestBindingXform(t *testing.T) {
	var source []int
	sm := observable.NewModel(&source)
	require.NotNil(t, sm)

	var dest [5]bool
	dm := observable.NewModel(&dest)
	require.NotNil(t, dm)

	for n := range 5 {
		key := strconv.Itoa(n)
		b, err := observable.NewBinding(key, dm, key, sm, "isNil")
		require.NoError(t, err)
		defer b.Release()

		assert.True(t, dest[n])
	}

	sm.InsertValueAt(0, 0)
	assert.False(t, dest[0])
	assert.True(t, dest[1])
}

func TestFullDuplexBindingXForm(t *testing.T) {
	celcius := 5.0
	fahrenheit := 0.0 // test the initial binding TO here doesn't backwash this zero value to celcius

	c, f := observable.NewModel(&celcius), observable.NewModel(&fahrenheit)

	b, err := observable.NewBinding("value", f, "value", c, "cToF")
	require.NoError(t, err)
	defer b.Release()

	assert.Equal(t, 41.0, fahrenheit)
	f.SetValue("value", 50.0)
	assert.Equal(t, 10.0, celcius)
}
