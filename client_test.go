package ga

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_Zero_Client_Start_Stop(t *testing.T) {

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
	)
	defer ts.Close()

	c := &Client{}

	c.urlStr = ts.URL

	go func() {
		err := c.Start()
		if err != nil && err != ErrClientClosed {
			t.Fatal(err)
		}
	}()

	for i := 0; i < 30; i++ {
		time.Sleep(time.Millisecond * 5)
		err := c.Report(&Event{
			"foo": fmt.Sprintf("%d", i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	err := c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	err = c.Report(&Event{
		"foo": fmt.Sprintf("%d", 999),
	})
	if err != ErrClientClosed {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

}

func Test_Zero_Client_Bad_API(t *testing.T) {

	errChan := make(chan error)

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad", 500)
		}),
	)
	defer ts.Close()

	c := &Client{
		BatchWait: time.Millisecond * 100,
	}

	c.urlStr = ts.URL

	c.HandleErr(ErrHandlerFunc(func(e Events, err error) {
		if err != nil {
			errChan <- err
		}
	}))

	go func() {
		err := c.Start()
		if err != nil && err != ErrClientClosed {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Millisecond * 10)

	err := c.Report(&Event{
		"foo": "baz",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	select {
	case err = <-errChan:
		if err == nil {
			t.Fatal("expected err")
		}
	case <-ctx.Done():
		t.Fatal("expected err")
	}

	err = c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

}

func Test_Zero_Client_Report(t *testing.T) {

	reqChan := make(chan string)
	errChan := make(chan error)

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				errChan <- err
			}
			reqChan <- string(b)
		}),
	)
	defer ts.Close()

	c := &Client{
		BatchWait: time.Millisecond * 100,
	}

	c.urlStr = ts.URL

	go func() {
		err := c.Start()
		if err != nil && err != ErrClientClosed {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Millisecond * 10)

	err := c.Report(&Event{
		"foo":   "baz",
		"alpha": "beta",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Report(&Event{
		"fooz":  "&azz",
		"delta": "$amma",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	select {
	case req := <-reqChan:
		if req != "alpha=beta&foo=baz\ndelta=%24amma&fooz=%26azz" {
			t.Fatal(req)
		}
	case err = <-errChan:
		if err == nil {
			t.Fatal("unexpected err", err)
		}
	case <-ctx.Done():
		t.Fatal("expected request")
	}

	err = c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

}

func Test_Zero_Client_Done_Send(t *testing.T) {

	reqChan := make(chan string)
	errChan := make(chan error)

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				errChan <- err
			}
			reqChan <- string(b)
		}),
	)
	defer ts.Close()

	c := &Client{
		BatchWait: time.Hour * 100,
	}

	c.urlStr = ts.URL

	go func() {
		err := c.Start()
		if err != nil && err != ErrClientClosed {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Millisecond * 10)

	err := c.Report(&Event{
		"foo":   "baz",
		"alpha": "beta",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	err = c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	select {
	case req := <-reqChan:
		if req != "alpha=beta&foo=baz" {
			t.Fatal(req)
		}
	case err := <-errChan:
		if err == nil {
			t.Fatal("unexpected err", err)
		}
	case <-ctx.Done():
		t.Fatal("expected request")
	}

	time.Sleep(time.Millisecond * 100)

}
