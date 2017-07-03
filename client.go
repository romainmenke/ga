package ga

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

// Client reports Events to GA.
type Client struct {
	// How long the client waits before reporting an Event to GA.
	// The default is 15 seconds.
	BatchWait time.Duration
	// HTTP is the http.Client used to make POST calls to GA.
	HTTP *http.Client
	// The time to wait for batch sends to complete.
	// If the timeout is exceeded the remaining items will be reported later.
	// Client.Shutdown will wait in steps of SendTimeout until all Events have been submitted.
	// The defaults to http.Client.Timeout if this is zero it will default to 5 seconds.
	SendTimeout time.Duration
	// The GA ID for Events.
	// This is only used by Client.DefaultHTTPHandler.
	TID string

	doneChan     chan struct{}
	errHandler   ErrHandler // useful for logging errors occurring on ga go routines
	eventChan    chan *Event
	eventCounter int32 // accessed atomically
	events       Events
	inShutdown   int32 // accessed atomically (non-zero means we're in Shutdown).
	mu           sync.Mutex
	started      int32  // accessed atomically (non-zero means we've Started).
	urlStr       string // set to httptest.NewServer().URL during tests.
}

func (c *Client) getDoneChan() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getDoneChanLocked()
}

func (c *Client) getDoneChanLocked() chan struct{} {
	if c.doneChan == nil {
		c.doneChan = make(chan struct{})
	}
	return c.doneChan
}

func (c *Client) closeDoneChanLocked() {
	ch := c.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by c.mu.
		close(ch)
	}
}

func (c *Client) closeEventChanLocked() {
	close(c.eventChan)
}

// Shutdown the Client.
// This will block until all Events reported before calling Shutdown have been submitted.
func (c *Client) Shutdown(ctx context.Context) error {
	atomic.AddInt32(&c.inShutdown, 1)
	defer atomic.AddInt32(&c.inShutdown, -1)

	c.mu.Lock()
	c.closeDoneChanLocked()
	c.closeEventChanLocked()
	c.mu.Unlock()

	for c.sending() {
		time.Sleep(c.HTTP.Timeout)
	}

	return nil
}

func (c *Client) sending() bool {
	if atomic.LoadInt32(&c.eventCounter) > 0 {
		return true
	}

	return false
}

// Start makes the Client receive Events and submit them to GA.
// This call will block until Client.Shutdown is called.
func (c *Client) Start() error {
	if atomic.LoadInt32(&c.started) > 0 {
		return ErrAlreadyStarted
	}

	atomic.AddInt32(&c.started, 1)
	defer atomic.AddInt32(&c.started, -1)

	c.events = make([]*Event, 0, 256)

	c.eventChan = make(chan *Event)

	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}

	if c.BatchWait == 0 {
		c.BatchWait = time.Second * 15
	}

	if c.SendTimeout == 0 {
		c.SendTimeout = c.HTTP.Timeout
	}

	if c.SendTimeout == 0 {
		c.SendTimeout = time.Second * 5
	}

	if c.urlStr == "" {
		c.urlStr = "https://www.google-analytics.com/batch"
	}

	if c.errHandler == nil {
		c.HandleErr(ErrHandlerFunc(func(e Events, err error) {}))
	}

	ticker := time.NewTicker(c.BatchWait)
	defer ticker.Stop()

	for {
		select {
		case e := <-c.eventChan:
			if e == nil {
				continue
			}

			atomic.AddInt32(&c.eventCounter, 1)

			c.events = append(c.events, e)
			if len(c.events) >= 20 {
				n, _ := c.send(c.events)
				c.events = c.events[:len(c.events)-int(n)]
				atomic.AddInt32(&c.eventCounter, -n)
			}

		case <-ticker.C:
			if len(c.events) > 0 {
				n, _ := c.send(c.events)
				c.events = c.events[:len(c.events)-int(n)]
				atomic.AddInt32(&c.eventCounter, -n)
			}
		case <-c.getDoneChan():
			if len(c.events) > 0 {
				n, _ := c.send(c.events)
				c.events = c.events[:len(c.events)-int(n)]
				atomic.AddInt32(&c.eventCounter, -n)
			}
			return ErrClientClosed
		}
	}
}

// Report is used to submit an Event to GA.
// This can be safely called by multiple go routines.
func (c *Client) Report(e *Event) error {
	select {
	case <-c.getDoneChan():
		return ErrClientClosed
	default:
	}

	c.eventChan <- e
	return nil
}

// ErrHandler is used to handle errors that occur while submitting to GA.
type ErrHandler interface {
	// Err receives the Events that erred and the corresponding error.
	// This is useful for logging and retry submitting Events.
	Err(Events, error)
}

// The ErrHandlerFunc type is an adapter to allow the use of ordinary functions as ErrHandlers.
// If f is a function with the appropriate signature, ErrHandlerFunc(f) is a ErrHandler that calls f.
type ErrHandlerFunc func(e Events, err error)

// Err receives the Events that erred and the corresponding error.
// This is useful for logging and retry submitting Events.
func (f ErrHandlerFunc) Err(e Events, err error) {
	f(e, err)
}

// HandleErr sets the ErrHandler to be used by the Client.
// Multiple calls will override.
func (c *Client) HandleErr(h ErrHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.errHandler != nil {
		return
	}

	c.errHandler = h
}

func (c *Client) send(events Events) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.SendTimeout)
	defer cancel()

	var sumN int32

	for len(events) >= 20 {
		n, err := c.sendBatch(ctx, events[:20])
		sumN += n
		if err != nil {
			return sumN, err
		}

		events = events[20:]
	}

	if len(events) > 0 {
		n, err := c.sendBatch(ctx, events)
		sumN += n
		if err != nil {
			return sumN, err
		}
	}

	return sumN, nil
}

func (c *Client) sendBatch(ctx context.Context, events Events) (int32, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		// execute
	}

	buf := bytes.NewBuffer(nil)
	_, err := events.WriteTo(buf)
	if err != nil {
		c.errHandler.Err(events, err)
		return int32(len(events)), nil // can't recover this
	}

	req, err := http.NewRequest("POST", c.urlStr, buf)
	if err != nil {
		c.errHandler.Err(events, err)
		return int32(len(events)), nil // can't recover this
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		c.errHandler.Err(events, err)
		return int32(len(events)), nil // can't recover this
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			err = errors.Wrap(ErrGoogleAnalytics, err.Error())
			c.errHandler.Err(events, err)
			return int32(len(events)), nil // can't recover this
		}
		err = errors.Wrap(ErrGoogleAnalytics, fmt.Sprintf("%d %s", resp.StatusCode, b))
		c.errHandler.Err(events, err)
		return int32(len(events)), nil // can't recover this
	}

	return int32(len(events)), nil
}
