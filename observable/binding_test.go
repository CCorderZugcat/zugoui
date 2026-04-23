package observable_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
)

func TestBinding(t *testing.T) {
	type bar struct {
		Field1 float64
	}
	type foo struct {
		Bar *bar
	}
	f := &foo{}
	fm := controllers.New(f)
	require.NotNil(t, fm)
	defer fm.Release()

	var m map[string]map[string]float64
	mm := controllers.New(&m)
	require.NotNil(t, mm)
	defer mm.Release()

	b, err := observable.NewBinding(
		"Bar.Field1", fm,
		"X.Y", mm,
		"cToF",
	)
	require.NoError(t, err)
	defer b.Release()

	require.NoError(t, observable.SetKeyPath(fm, "Bar.Field1", 5.0))
	assert.Equal(t, 41.0, m["X"]["Y"])

	require.NoError(t, observable.SetKeyPath(mm, "X.Y", 50.0))
	assert.Equal(t, 10.0, f.Bar.Field1)
}
