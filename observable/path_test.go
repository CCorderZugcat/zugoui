package observable_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
)

func TestKeyPathSetter(t *testing.T) {
	type foo struct {
		Field1 map[string]int
	}
	var ar [5]*foo
	m := controllers.New(&ar)
	require.NotNil(t, m)

	s := observable.NewPathSetter(m)
	s.SetValue("1.Field1.Foo", 5)

	assert.Equal(t, 5, ar[1].Field1["Foo"])
}

func TestKeyPathObserver(t *testing.T) {
	type bar struct {
		Field1 string
	}
	type foo struct {
		Bar *bar
	}
	f := foo{}
	m := controllers.New(&f)
	require.NotNil(t, m)
	defer m.Release()

	var om map[string]map[string]string
	o := controllers.New(&om)
	require.NotNil(t, o)
	defer o.Release()

	p := observable.NewPathObserver("Bar.Field1", m)
	defer p.Release()
	p.AddObserver("Bar.Field1", observable.NewPathSetter(o))

	err := observable.SetKeyPath(m, "Bar.Field1", "abc")
	require.NoError(t, err)
	assert.Equal(t, "abc", om["Bar"]["Field1"])
}

func TestKeyPathWildcard(t *testing.T) {
	type bar struct {
		Field1 string
	}
	type foo struct {
		Bar     *bar
		Field   int
		Strings [2]string
	}
	f := &foo{}
	m := controllers.New(f)
	require.NotNil(t, m)
	defer m.Release()

	var om map[string]any
	o := controllers.New(&om)
	require.NotNil(t, o)
	defer o.Release()

	p := observable.NewPathObserver("*", m)
	p.AddObserver("", o)

	require.NoError(t, observable.SetKeyPath(m, "Strings.1", "one"))
	require.NoError(t, observable.SetKeyPath(m, "Bar.Field1", "two"))

	assert.Equal(t, "one", p.Value("Strings.1"))
	assert.Equal(t, "two", p.Value("Bar.Field1"))

	assert.Equal(t, "one", om["Strings.1"])
	assert.Equal(t, "two", om["Bar.Field1"])

	require.NoError(t, observable.SetKeyPath(m, "Bar", &bar{}))
	assert.Zero(t, om["Bar.Field1"])
}

func TestKeyPathSet(t *testing.T) {
	type bar struct {
		Field1 string
	}
	type foo struct {
		Bar     *bar
		Field   int
		Strings [2]string
	}
	f := &foo{}
	m := controllers.New(f)
	require.NotNil(t, m)
	defer m.Release()

	var om map[string]any
	o := controllers.New(&om)
	require.NotNil(t, o)
	defer o.Release()

	p := observable.NewPathObserver("*", m)
	p.AddObserver("", o)

	p.SetValue("Bar.Field1", "hello")
	assert.Equal(t, "hello", f.Bar.Field1)
	assert.Empty(t, om)

	require.NoError(t, observable.SetKeyPath(m, "Bar.Field1", "again"))
	assert.Equal(t, "again", f.Bar.Field1)
	assert.Equal(t, "again", om["Bar.Field1"])
}

func TestJoinKeyPath(t *testing.T) {
	assert.Equal(t, "a", observable.JoinKeyPath("", "a"))
	assert.Equal(t, "a", observable.JoinKeyPath("a", ""))
	assert.Equal(t, "a", observable.JoinKeyPath("a"))
	assert.Equal(t, "a.b", observable.JoinKeyPath("a", "b"))
	assert.Equal(t, "a.b.c", observable.JoinKeyPath("a", "b", "c"))
}

func TestPathObserverLeaf(t *testing.T) {
	type bar struct {
		Field1 string
	}
	type foo struct {
		Bars []*bar
	}
	f := &foo{}
	m := controllers.New(f)
	require.NotNil(t, m)
	defer m.Release()

	bars := m.Value("Bars").(observable.MutableSource)

	var om map[string]any
	o := controllers.New(&om)
	require.NotNil(t, o)
	defer o.Release()

	p := observable.NewPathObserver("Bars.1.Field1", m)
	p.AddObserver("", o)
	defer p.Release()

	bars.InsertValueAt(0, &bar{Field1: "first"})
	bars.InsertValueAt(0, &bar{Field1: "second"})

	assert.Equal(t, "first", om["Bars.1.Field1"])

	bars.RemoveValueAt(1)
	assert.Equal(t, nil, om["Bars.1.Field1"])
}
