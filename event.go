package ga

import (
	"io"
	"net/url"
	"sort"
)

// Event represents a single Google Analytics event.
// It is unaware of the exact GA api but knows how to format data for GA.
type Event map[string]string

// Set inserts a key value pair into Event.
func (e Event) Set(key, value string) {
	e[key] = value
}

// Get retrieves the value for key or an empty string.
func (e Event) Get(key string) string {
	if v, ok := e[key]; ok {
		return v
	}
	return ""
}

// Del removes a key value pair from Event.
func (e Event) Del(key string) {
	delete(e, key)
}

func (e Event) sortedKeyValues() pairs {
	l := make([]pair, 0, len(e))

	for k, v := range e {
		if k != "" && v != "" {
			l = append(l, pair{key: k, value: v})
		}
	}

	sort.Sort(pairs(l))

	return l
}

// WriteTo formats the key value pairs in Event and writes them to w.
func (e Event) WriteTo(w io.Writer) (int64, error) {
	ws, ok := w.(writeStringer)
	if !ok {
		ws = stringWriter{w}
	}

	var (
		err  error
		n    int
		sumN int64
	)

	s := e.sortedKeyValues()

	for _, p := range s {
		var ps string
		if sumN == 0 {
			ps = p.key + "=" + url.QueryEscape(p.value)
		} else {
			ps = "&" + p.key + "=" + url.QueryEscape(p.value)
		}

		n, err = ws.WriteString(ps)
		sumN += int64(n)
		if err != nil {
			return sumN, err
		}
	}

	return sumN, err
}

// Events is a collection of Event instances.
// It is used to batch report Events.
type Events []*Event

// WriteTo formats a batch of Events and writes them to w.
func (l Events) WriteTo(w io.Writer) (int64, error) {
	ws, ok := w.(writeStringer)
	if !ok {
		ws = stringWriter{w}
	}

	var (
		err          error
		sumN         int64
		firstWritten bool
	)

	for _, e := range l {
		if len(*e) == 0 {
			continue
		}
		if firstWritten {
			var n int
			n, err = ws.WriteString("\n")
			sumN += int64(n)
			if err != nil {
				return sumN, err
			}
		}

		var n int64
		n, err = e.WriteTo(ws)
		sumN += n
		if err != nil {
			return sumN, err
		}

		firstWritten = true
	}

	return sumN, err
}

type writeStringer interface {
	io.Writer
	WriteString(string) (int, error)
}

type stringWriter struct {
	io.Writer
}

func (w stringWriter) WriteString(s string) (n int, err error) {
	return w.Writer.Write([]byte(s))
}

type pair struct {
	key   string
	value string
}

type pairs []pair

func (p pairs) Len() int {
	return len(p)
}
func (p pairs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p pairs) Less(i, j int) bool {
	return p[i].key < p[j].key
}
