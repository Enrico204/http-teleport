package transport

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPTrackResponse struct {
	Error         error
	Status        string
	StatusCode    int
	Proto         string
	Header        http.Header
	ContentLength int64
	When          time.Time
	Body          []byte
	IsJSON        bool

	// httputil.DumpResponse ?
}

func (h *HTTPTrackResponse) String() string {
	var buf = strings.Builder{}
	buf.WriteString(h.Proto + " " + h.Status + "\n")
	for k, h := range h.Header {
		for _, v := range h {
			buf.WriteString(k)
			buf.WriteString(": ")
			buf.WriteString(v)
			buf.WriteString("\n")
		}
	}
	return strings.TrimSpace(buf.String())
}

type HTTPTrack struct {
	Method        string
	URL           *url.URL
	Proto         string
	Header        http.Header
	ContentLength int64
	When          time.Time
	RemoteAddr    string
	Body          []byte
	IsJSON        bool

	Response *HTTPTrackResponse
}

func (h *HTTPTrack) Duration() time.Duration {
	if h.Response == nil {
		return -1
	}
	return h.Response.When.Sub(h.When)
}

func (h *HTTPTrack) String() string {
	var buf = strings.Builder{}
	buf.WriteString(h.Method)
	buf.WriteString(" ")
	buf.WriteString(h.URL.Path)
	if len(h.URL.Query()) > 0 {
		buf.WriteString("?")
		buf.WriteString(h.URL.RawQuery)
	}
	buf.WriteString(" ")
	buf.WriteString(h.Proto)
	buf.WriteString("\n")
	for k, h := range h.Header {
		for _, v := range h {
			buf.WriteString(k)
			buf.WriteString(": ")
			buf.WriteString(v)
			buf.WriteString("\n")
		}
	}
	return strings.TrimSpace(buf.String())
}
