package ga

import (
	"net/http"

	"github.com/pborman/uuid"
)

// DefaultHTTPHandler attempts to provide a sane default HTTPHandler to report pageview events.
// For this to work the TID of the Client must be set.
func (c *Client) DefaultHTTPHandler(h http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := Event{
			"tid": c.TID,
			"cid": uuid.New(),
			"t":   "pageview",
			"v":   "1",
			"dh":  r.Host,
			"dp":  r.URL.RequestURI(),
			"dr":  r.Referer(),
			"ua":  r.UserAgent(),
		}

		c.Report(e)

		h.ServeHTTP(w, r)
	})
}
