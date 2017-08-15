package ga

import (
	"bytes"
	"testing"
	"time"
)

func Test_Event_WriteTo(t *testing.T) {
	e := &Event{}
	e.Set("alpha", "one")
	e.Set("beta", "two")

	buf := bytes.NewBuffer(nil)
	_, err := e.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != "alpha=one&beta=two" {
		t.Fatal(buf.String())
	}
}

func Test_Event_WriteTo_WithEscaping(t *testing.T) {
	e := &Event{}
	e.Set("alpha", "@lpha")
	e.Set("beta", "&eta")

	buf := bytes.NewBuffer(nil)
	_, err := e.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != "alpha=%40lpha&beta=%26eta" {
		t.Fatal(buf.String())
	}
}

func Test_Events_WriteTo_WithEscaping(t *testing.T) {

	empty := Event{}

	a := Event{}
	a.Set("alpha", "@lpha")
	a.Set("beta", "&eta")
	a.Set("delta", "")

	b := Event{}
	b.Set("one", "*ne")
	b.Set("two", "!wo")

	l := events{
		event{reportedAt: time.Now(), e: empty},
		event{reportedAt: time.Now(), e: a},
		event{reportedAt: time.Now(), e: empty},
		event{reportedAt: time.Now(), e: b},
	}

	buf := bytes.NewBuffer(nil)
	_, err := l.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != "alpha=%40lpha&beta=%26eta\none=%2Ane&two=%21wo" {
		t.Fatal(buf.String())
	}
}
