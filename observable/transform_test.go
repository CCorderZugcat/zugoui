package observable_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
)

func TestXforms(t *testing.T) {
	foo := &struct {
		Field1 string
		Field2 *int
	}{}

	m := controllers.New(foo)
	require.NotNil(t, m)
	defer m.Release()

	x := observable.NewTransformer("isNil")
	require.NotNil(t, x)
	assert.True(t, x.Get(m.Value("Field2")).(bool))
	assert.False(t, x.Mutable())

	x = observable.NewTransformer("isZero")
	require.NotNil(t, x)
	assert.True(t, x.Get(m.Value("Field1")).(bool))

	foo.Field1 = "abc"
	x = observable.NewTransformer("len")
	require.NotNil(t, x)
	assert.Equal(t, 3, x.Get(m.Value("Field1")))
}

type cToF struct {
	*observable.BaseTransform
}

func (_ cToF) NewTransformer() observable.Transformer {
	return cToF{
		BaseTransform: observable.NewBaseTransform(
			func(value any) any {
				return (value.(float64) * (9.0 / 5.0)) + 32.0
			},
			func(value any) any {
				return (value.(float64) - 32.0) * (5.0 / 9.0)
			},
		),
	}
}

func init() {
	observable.RegisterTransformer("cToF", cToF{})
}

func TestFullDuplexTransform(t *testing.T) {
	x := observable.NewTransformer("cToF")
	assert.True(t, x.Mutable())

	assert.Equal(t, 41.0, x.Get(5.0))
	assert.Equal(t, 10.0, x.Set(50.0))
}
