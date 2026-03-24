package observable_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/observabletest"
)

func TestMapObserver(t *testing.T) {
	m := make(map[string]int)
	o := observable.NewModel(&m)
	require.NotNil(t, o)

	testObserver(t, o)
	assert.Equal(t, map[string]int{"One": 1, "Two": 2, "Three": 3}, m)
}

func TestStructModel(t *testing.T) {
	type model struct {
		One   int
		Two   int `kabloouie:"too,two,to"`
		Three int
	}
	m := model{}
	o := observable.NewModel(&m)
	require.NotNil(t, o)

	testObserver(t, o)
	assert.Equal(t, model{One: 1, Two: 2, Three: 3}, m)

	assert.ElementsMatch(t, []string{"too", "two", "to"}, o.Tag("Two", "kabloouie"))
}

func testObserver(t testing.TB, o *observable.Model) {
	t.Helper()

	ob, ch := observabletest.New()
	defer close(ch)

	o.AddObserver("Two", ob)
	defer o.RemoveObserver("Two", ob)

	o.SetValue("One", 1)
	o.SetValue("Two", 2)
	o.SetValue("Three", 3)

	o1 := <-ch
	assert.Equal(t, 2, o1.Value)
	assert.Equal(t, 1, o.Value("One"))
	assert.ElementsMatch(t, []string{"One", "Two", "Three"}, o.Keys())
}

func TestValueObserver(t *testing.T) {
	m := "hello"
	o := observable.NewModelValue(reflect.ValueOf(&m).Elem())
	require.NotNil(t, o)

	assert.Equal(t, m, o.Interface())
	assert.Equal(t, reflect.TypeFor[string](), o.Type())

	ob, ch := observabletest.New()
	defer close(ch)

	o.AddObserver("value", ob)

	o.SetValue("value", "changed")

	o1 := <-ch
	assert.Equal(t, "changed", o1.Value)
	assert.Equal(t, "changed", o.Value("value"))
	assert.Equal(t, "changed", m)
}

func TestSliceObserver(t *testing.T) {
	m := []int{1, 2, 3}
	o := observable.NewModel(&m)
	require.NotNil(t, o)

	o.SetValue("1", 5)
	assert.Equal(t, []int{1, 5, 3}, m)

	o.InsertValueAt(3, 4)
	assert.Equal(t, []int{1, 5, 3, 4}, m)

	o.InsertValueAt(0, 9)
	assert.Equal(t, []int{9, 1, 5, 3, 4}, m)

	o.RemoveValueAt(2)
	assert.Equal(t, []int{9, 1, 3, 4}, m)

	o.SetValueAt(1, 8)
	assert.Equal(t, []int{9, 8, 3, 4}, m)

	assert.Equal(t, 4, o.ValueAt(3))

	assert.ElementsMatch(t, []string{"0", "1", "2", "3"}, o.Keys())
	assert.Equal(t, 4, o.Value("len"))
	assert.Equal(t, 8, o.Value("1"))
}

func TestMapObserverKeyType(t *testing.T) {
	type myString string
	var m map[myString]int
	o := observable.NewModel(&m)
	require.NotNil(t, o)

	o.SetValueFor("one", 1)
	o.SetValueFor("two", 2)

	assert.Equal(t, map[myString]int{"one": 1, "two": 2}, m)

	o.RemoveValueFor("one")
	assert.Equal(t, map[myString]int{"two": 2}, m)

	assert.Equal(t, 2, o.ValueFor("two"))

	o.SetValueFor("three", 3)
	assert.ElementsMatch(t, []string{"two", "three"}, o.Keys())
}

func TestSlice(t *testing.T) {
	dst, src := make([]int, 0), make([]int, 0)
	dm, sm := observable.NewModel(&dst), observable.NewModel(&src)

	sm.AddObserver("value", dm) // observing "value" keeps us to deltas
	sm.InsertValueAt(0, 3)
	sm.InsertValueAt(0, 2)
	sm.InsertValueAt(0, 1)

	assert.Equal(t, []int{1, 2, 3}, src)
	assert.Equal(t, []int{1, 2, 3}, dst)
}

func TestNilModel(t *testing.T) {
	type foo struct {
		Field1 string
	}
	var ar [5]*foo

	m := observable.NewModel(&ar)
	require.NotNil(t, m)
	defer m.Release()

	v := m.Value("0")
	assert.NotNil(t, v)
	s := v.(observable.Source)
	assert.Equal(t, []string{"Field1"}, s.Keys())
}

func TestObserveKeyPath(t *testing.T) {
	type bar struct {
		Field string
	}
	type foo struct {
		Bars [1]*bar
	}

	f := &foo{}
	m := observable.NewModel(f)
	require.NotNil(t, m)
	defer m.Release()

	om := make(map[string]map[string]map[string]string)
	o := observable.NewModel(om)
	require.NotNil(t, o)
	defer o.Release()

	m.AddObserver("Bars.0.Field", o)
	defer m.RemoveObserver("", o)

	m.SetValue("Bars.0", &bar{Field: "contents"})
	assert.Equal(t, "contents", om["Bars"]["0"]["Field"])
}
