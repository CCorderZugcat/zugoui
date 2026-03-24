package observable_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CCorderZugcat/zugoui/observable"
	"github.com/CCorderZugcat/zugoui/observable/observabletest"
)

func TestWriter(t *testing.T) {
	m := struct {
		Field string
	}{}

	source := observable.NewModel(&m)

	os, cs := observabletest.New()
	defer close(cs)
	source.AddObserver("Field", os)

	w := observable.NewWriter(source)
	defer w.Release()

	ow, cw := observabletest.New()
	defer close(cw)
	w.AddObserver("Field", ow)

	source.SetValue("Field", "initial")

	// everybody sees updates at the source
	u := <-cw
	assert.Equal(t, u.Value, "initial")
	u = <-cs
	assert.Equal(t, u.Value, "initial")

	// only the source should see the writer update
	w.SetValue("Field", "update")

	select {
	case <-cw:
		t.Fatalf("got self update")
	default:
	}

	u = <-cs
	assert.Equal(t, "update", u.Value)
	assert.Equal(t, "update", source.Value("Field"))

	w2 := observable.NewWriter(w)
	defer w2.Release()

	o2, c2 := observabletest.New()
	defer close(c2)
	w2.AddObserver("Field", o2)

	w.SetValue("Field", "update again")
	<-cs
	w2.SetValue("Field", "update again")

	select {
	case u = <-cw:
		t.Fatalf("got self update on w")
	case u = <-c2:
		t.Fatalf("got self update on w2")
	default:
	}

	u = <-cs
	assert.Equal(t, u.Value, "update again")
	assert.Equal(t, "update again", source.Value("Field"))
}
