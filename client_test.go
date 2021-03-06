package ga

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
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
		err := c.Report(Event{
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

	err = c.Report(Event{
		"foo": fmt.Sprintf("%d", 999),
	})
	if err != ErrClientClosed {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

}

func Test_Zero_Client_Bad_API(t *testing.T) {

	errChan := make(chan error, 1)
	var receivedErr bool

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

	c.HandleErr(ErrHandlerFunc(func(e []Event, err error) {
		if err != nil {
			errChan <- err
		}
		if err == nil {
			errChan <- errors.New("no error")
		}
	}))

	go func() {
		err := c.Start()
		if err != nil && err != ErrClientClosed {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Millisecond * 10)

	err := c.Report(Event{
		"foo": "baz",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

RESULT_LOOP:
	for {
		select {
		case err = <-errChan:
			if err != nil && err.Error() != "no error" {
				t.Log(err)
				receivedErr = true
			}
		case <-ctx.Done():
			if !receivedErr {
				t.Fatal("expected err")
			}
			break RESULT_LOOP
		}
	}

	err = c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

}

// One report, gives timeout on first try
// Succeeds on second try
func Test_Zero_Client_Sometimes_Slow_API(t *testing.T) {

	errChan := make(chan error, 1)
	var receivedErr bool
	reqChan := make(chan string, 1)
	var receivedReq bool

	var fast bool
	var fastMu sync.Mutex

	c := &Client{
		BatchWait:   time.Millisecond * 500,
		SendTimeout: time.Second * 10,
		HTTP: &http.Client{
			Timeout: time.Millisecond * 50,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: time.Millisecond * 50,
				}).Dial,
				TLSHandshakeTimeout:   time.Millisecond * 100,
				ResponseHeaderTimeout: time.Millisecond * 50,
				IdleConnTimeout:       time.Millisecond * 5,
			},
		},
	}

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fastMu.Lock()
			defer fastMu.Unlock()

			if !fast {
				time.Sleep(time.Millisecond * 250)
				fast = true
				w.WriteHeader(500)
				return
			}

			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				errChan <- err
			}
			if err == nil {
				errChan <- errors.New("no error")
			}

			reqChan <- string(b)

			w.WriteHeader(200)
		}),
	)
	defer ts.Close()

	c.urlStr = ts.URL

	c.HandleErr(ErrHandlerFunc(func(e []Event, err error) {
		if err != nil {
			t.Log("sending err", err)
			errChan <- err
		}
		if err == nil {
			t.Log("sending non err")
			errChan <- errors.New("no error")
		}
	}))

	go func() {
		err := c.Start()
		if err != nil && err != ErrClientClosed {
			t.Fatal(err)
		}
	}()

	go func() {
		time.Sleep(time.Millisecond * 10)
		err := c.Report(Event{
			"foo": "baz",
		})
		if err != nil {
			t.Fatal(err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

RESULT_LOOP:
	for {
		select {
		case req := <-reqChan:
			if match, _ := regexp.MatchString("foo=baz&qt=\\d*", req); !match {
				t.Fatal(req)
			}
			receivedReq = true
		case err := <-errChan:
			if err != nil && err.Error() != "no error" {
				t.Fatal("unexpected err", err)
				receivedErr = true
			}
		case <-ctx.Done():
			if receivedErr {
				t.Fatal("unexpected err")
			}
			if !receivedReq {
				t.Fatal("expected req")
			}
			break RESULT_LOOP
		}
	}

	err := c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

}

func Test_Zero_Client_Report(t *testing.T) {

	errChan := make(chan error, 1)
	var receivedErr bool
	reqChan := make(chan string, 1)
	var receivedReq bool

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				errChan <- err
			}
			if err == nil {
				errChan <- errors.New("no error")
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

	err := c.Report(Event{
		"foo":   "baz",
		"alpha": "beta",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Report(Event{
		"fooz":  "&azz",
		"delta": "$amma",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

RESULT_LOOP:
	for {
		select {
		case req := <-reqChan:
			if match, _ := regexp.MatchString("alpha=beta&foo=baz&qt=\\d*\\ndelta=%24amma&fooz=%26azz&qt=\\d*", req); !match {
				t.Fatal(req)
			}
			receivedReq = true
		case err = <-errChan:
			if err != nil && err.Error() != "no error" {
				t.Fatal("unexpected err", err)
				receivedErr = true
			}
		case <-ctx.Done():
			if receivedErr {
				t.Fatal("unexpected err")
			}
			if !receivedReq {
				t.Fatal("expected req")
			}
			break RESULT_LOOP
		}
	}

	err = c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Millisecond * 100)

}

func Test_Zero_Client_Done_Send(t *testing.T) {

	errChan := make(chan error, 1)
	var receivedErr bool
	reqChan := make(chan string, 1)
	var receivedReq bool

	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				errChan <- err
			}
			if err == nil {
				errChan <- errors.New("no error")
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

	err := c.Report(Event{
		"foo":   "baz",
		"alpha": "beta",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Shutdown(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

RESULT_LOOP:
	for {
		select {
		case req := <-reqChan:
			if req != "alpha=beta&foo=baz" {
				t.Fatal(req)
			}
			receivedReq = true
		case err := <-errChan:
			if err != nil && err.Error() != "no error" {
				t.Fatal("unexpected err", err)
				receivedErr = true
			}
		case <-ctx.Done():
			if receivedErr {
				t.Fatal("unexpected err")
			}
			if !receivedReq {
				t.Fatal("expected req")
			}
			break RESULT_LOOP
		}
	}

}
