package controllers_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
	"github.com/CCorderZugcat/zugoui/observable/observabletest"
)

func TestMapObserver(t *testing.T) {
	m := make(map[string]int)
	o := controllers.New(&m)
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
	o := controllers.New(&m)
	require.NotNil(t, o)

	testObserver(t, o)
	assert.Equal(t, model{One: 1, Two: 2, Three: 3}, m)

	assert.ElementsMatch(t, []string{"too", "two", "to"}, o.Tag("Two", "kabloouie"))
}

func testObserver(t testing.TB, o *controllers.Model) {
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
	o := controllers.NewValue(reflect.ValueOf(&m).Elem())
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
	o := controllers.New(&m)
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
	o := controllers.New(&m)
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
	dm, sm := controllers.New(&dst), controllers.New(&src)

	sm.AddObserver("value", dm) // observing "value" keeps us to deltas
	sm.InsertValueAt(0, 3)
	sm.InsertValueAt(0, 2)
	sm.InsertValueAt(0, 1)

	assert.Equal(t, []int{1, 2, 3}, src)
	assert.Equal(t, []int{1, 2, 3}, dst)
}

func TestSliceGrowth(t *testing.T) {
	s := make([]int, 0)
	m := controllers.New(&s)

	m.InsertValueAt(5, 1)
	assert.Equal(t, []int{0, 0, 0, 0, 0, 1}, s)

	s = s[:0]
	assert.Equal(t, 0, m.Value("len").(int))

	m.SetValueAt(3, 2)
	assert.Equal(t, []int{0, 0, 0, 2}, s)
}

func TestNilModel(t *testing.T) {
	type foo struct {
		Field1 string
	}
	var ar [5]*foo

	m := controllers.New(&ar)
	require.NotNil(t, m)
	defer m.Release()

	v := m.Value("0")
	require.NotNil(t, v)
	s := v.(observable.Source)
	assert.Equal(t, []string{"Field1"}, s.Keys())
}

func TestNilMap(t *testing.T) {
	var p map[string]map[string]int
	m := controllers.New(&p)
	require.NotNil(t, m)
	defer m.Release()

	v := m.Value("foo").(observable.MutableSource)
	require.NotNil(t, v)
	v.SetValue("bar", 5)

	assert.Equal(t, 5, p["foo"]["bar"])
}

func TestSetWithController(t *testing.T) {
	type bar struct {
		Field1 string
	}
	type foo struct {
		Bars []*bar
	}
	f1 := &foo{
		Bars: []*bar{
			{
				Field1: "value",
			},
		},
	}
	f2 := &foo{}

	m1 := controllers.New(f1)
	m2 := controllers.New(f2)

	m2.SetValue("Bars", m1.Value("Bars"))

	assert.Equal(t, f1.Bars[0].Field1, f2.Bars[0].Field1)
}
