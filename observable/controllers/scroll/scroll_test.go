package scroll_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/controllers"
	"github.com/CCorderZugcat/zugoui/observable/controllers/scroll"
)

type item struct {
	Field1 string
	Field2 int
}

type foo struct {
	Items []*item `controller:"scroll,3"`
}

func TestScroll(t *testing.T) {
	f := &foo{
		Items: []*item{
			{
				Field1: "",
				Field2: 3,
			},
		},
	}
	m := controllers.New(f)
	require.NotNil(t, m)
	defer m.Release()

	observable.SetKeyPath(m, "Items.0.Field1", "item3")
	assert.Equal(t, "item3", f.Items[0].Field1)

	items := m.Value("Items").(*scroll.Scroll)

	items.Insert("insert")
	observable.SetKeyPath(m, "Items.0.Field1", "item2")
	observable.SetKeyPath(m, "Items.0.Field2", 2)

	items.Insert("insert")
	observable.SetKeyPath(m, "Items.0.Field1", "item1")
	observable.SetKeyPath(m, "Items.0.Field2", 1)

	items.Insert("insert")
	observable.SetKeyPath(m, "Items.0.Field1", "item0")
	observable.SetKeyPath(m, "Items.0.Field2", 0)

	assert.Equal(t, "item2", f.Items[2].Field1)
	assert.Equal(t, 2, f.Items[2].Field2)

	assert.False(t, items.Value("canUp").(bool))
	assert.True(t, items.Value("canDown").(bool))

	items.Down("down")

	assert.True(t, items.Value("canUp").(bool))
	assert.False(t, items.Value("canDown").(bool))

	assert.Equal(t, 1, items.Value("0").(observable.Source).Value("Field2"))
}
