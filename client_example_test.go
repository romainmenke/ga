package ga_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/romainmenke/ga"
)

func ExampleClient() {

	// zero Client is good as is.
	c := &ga.Client{}

	go func() {
		err := c.Start()
		if err != nil && err != ga.ErrClientClosed {
			fmt.Println(err)
			return
		}
	}()

	time.Sleep(time.Millisecond * 5)

	c.Report(ga.Event{
		"foo": "baz",
	})

	e := ga.Event{}
	e.Set("t", "pageview")
	c.Report(e)

	err := c.Shutdown(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func ExampleClient_HTTP() {

	c := &ga.Client{
		HTTP: &http.Client{
			Timeout:   time.Second * 10,
			Transport: &http.Transport{
			// more setup
			},
		},
	}

	go func() {
		err := c.Start()
		if err != nil && err != ga.ErrClientClosed {
			fmt.Println(err)
			return
		}
	}()

	time.Sleep(time.Millisecond * 5)

	c.Report(ga.Event{
		"foo": "baz",
	})

	err := c.Shutdown(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func ExampleClient_SendTimeout() {

	// configure pauzes and timeouts to fit your needs.
	c := &ga.Client{
		BatchWait:   time.Second * 30,
		SendTimeout: time.Second * 10,
		HTTP: &http.Client{
			Timeout: time.Second * 15,
		},
	}

	go func() {
		err := c.Start()
		if err != nil && err != ga.ErrClientClosed {
			fmt.Println(err)
			return
		}
	}()

	time.Sleep(time.Millisecond * 5)

	c.Report(ga.Event{
		"foo": "baz",
	})

	err := c.Shutdown(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func ExampleClient_HandleErr() {

	c := &ga.Client{}

	// determine what happens with errors.
	c.HandleErr(ga.ErrHandlerFunc(func(events []ga.Event, err error) {
		// you can log the error
		fmt.Println(err)

		// some errs might be Wrapped
		fmt.Println(errors.Cause(err))

		// you could re-report the events here if you consider the error a temporary glitch.
		if err.Error() == "just a flesh wound" {
			for _, e := range events {
				c.Report(e)
			}
		}
	}))

	go func() {
		err := c.Start()
		if err != nil && err != ga.ErrClientClosed {
			fmt.Println(err)
			return
		}
	}()

	time.Sleep(time.Millisecond * 5)

	c.Report(ga.Event{
		"foo": "baz",
	})

	err := c.Shutdown(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func ExampleClient_DefaultHTTPHandler() {

	c := &ga.Client{
		TID: "some tid",
	}

	myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do server stuff
	})

	wrappedHandler := c.DefaultHTTPHandler(myHandler)

	http.Handle("/", wrappedHandler)

	go func() {
		err := c.Start()
		if err != nil && err != ga.ErrClientClosed {
			fmt.Println(err)
			return
		}
	}()

	time.Sleep(time.Millisecond * 5)

	c.Report(ga.Event{
		"foo": "baz",
	})

	err := c.Shutdown(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}
