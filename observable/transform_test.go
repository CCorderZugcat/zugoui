package observable_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
)

func TestXforms(t *testing.T) {
	foo := &struct {
		Field1 string
		Field2 *int
	}{}

	m := observable.NewModel(foo)
	require.NotNil(t, m)
	defer m.Release()

	x := observable.NewTransformer("isNil", "Field2", m)
	require.NotNil(t, x)
	assert.Equal(t, true, x.Value("value"))
	x.Release()

	x = observable.NewTransformer("isZero", "Field1", m)
	require.NotNil(t, x)
	assert.Equal(t, true, x.Value("value"))
	x.Release()

	foo.Field1 = "abc"
	x = observable.NewTransformer("len", "Field1", m)
	require.NotNil(t, x)
	assert.Equal(t, 3, x.Value("value"))
	x.Release()
}

type cToF struct {
	*observable.BaseTransform
}

func (_ cToF) NewTransformer(key string, source observable.Source) observable.Transformer {
	return cToF{
		BaseTransform: observable.NewBaseTransform(
			key, source,
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
	var celcius float64
	s := observable.NewModel(&celcius)
	defer s.Release()

	x := observable.NewTransformer("cToF", "value", s)
	defer x.Release()

	var o float64
	om := observable.NewModel(&o)
	defer om.Release()

	x.AddObserver("value", om)
	defer x.RemoveObserver("value", om)

	s.SetValue("value", 5.0)
	assert.Equal(t, 41.0, x.Value("value"))
	assert.Equal(t, 41.0, o)

	x.SetValue("value", 50.0)
	assert.Equal(t, 10.0, celcius)
	assert.Equal(t, 50.0, o)
}

func TestObserveTransform(t *testing.T) {
	s := make(map[string]int)
	sm := observable.NewModel(s)
	defer sm.Release()

	x := observable.NewTransformer("isNil", "one", sm)
	defer x.Release()

	b := true
	bm := observable.NewModel(&b)
	defer bm.Release()

	x.AddObserver("value", bm)
	defer x.RemoveObserver("value", bm)

	sm.SetValue("one", 1)
	assert.False(t, b)
}
