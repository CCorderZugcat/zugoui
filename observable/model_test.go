package observable_test

import (
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
		Two   int
		Three int
	}
	m := model{}
	o := observable.NewModel(&m)
	require.NotNil(t, o)

	testObserver(t, o)
	assert.Equal(t, model{One: 1, Two: 2, Three: 3}, m)
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
	o := observable.NewModel(&m)
	require.NotNil(t, o)

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
}

func TestMapOvserver(t *testing.T) {
	var m map[string]int
	o := observable.NewModel(&m)
	require.NotNil(t, o)

	o.SetValueFor("one", 1)
	o.SetValueFor("two", 2)

	assert.Equal(t, map[string]int{"one": 1, "two": 2}, m)

	o.RemoveValueFor("one")
	assert.Equal(t, map[string]int{"two": 2}, m)
}
