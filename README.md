[![Go Report Card](https://goreportcard.com/badge/github.com/romainmenke/ga)](https://goreportcard.com/report/github.com/romainmenke/ga)
[![GoDoc](https://godoc.org/github.com/romainmenke/ga?status.svg)](https://godoc.org/github.com/romainmenke/ga)

# GA

Report events to google analytics with the [Measurement Protocol](https://developers.google.com/analytics/devguides/collection/protocol/v1/).

GA doesn't contain helper types for GA parameters. Their API might change/grow and maintaining API parity would cost me too much time.

It does expose a simple and effective means to batch report. `Report` doesn't block which makes it great for server-side reporting where you do not want to spawn a network call for each incoming request.

`Report` will add events to a slice. After X time (you can adjust this) the events will be submitted to GA. If 20 or more events are reported before X time has passed it will immediately submit the events, as 20 is the max batch size.

---

### Simple Usage

```go
func main() {
	// create a Client
	c := &ga.Client{}

	// start a go routine to handle Events
	go func() {

		// Start the Client
		err := c.Start()
		if err != nil && err != ga.ErrClientClosed {
			fmt.Println(err)
			return
		}
	}()

	// report an Event
	c.Report(&ga.Event{
		"t": "pageview",
		// ...
	})

	// or report like so
	e := &ga.Event{}
	e.Set("t", "pageview")
	c.Report(e)

	// Shutdown the Client
	err := c.Shutdown(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}
```

---

### Measurement Protocol Reference

[reference](https://developers.google.com/analytics/devguides/collection/protocol/v1/parameters)

---

**see tests and examples for more details**
