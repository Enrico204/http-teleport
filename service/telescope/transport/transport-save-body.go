package transport

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func (tr *proxyTransport) saveBody(reader io.ReadCloser, headers http.Header, contentLength int64) (io.ReadCloser, []byte, error) {
	if !tr.storeBody ||
		!(contentLength > 0 && contentLength < MaxContentLengthSize) ||
		!isTextMimeType(headers.Get("Content-Type")) {
		return reader, nil, nil
	}

	// We can't read the streaming while it's flowing, so we do "store and forward"
	// Read everything
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}
	err = reader.Close()
	if err != nil {
		return nil, nil, err
	}

	// Replace body reader with in-memory reader
	if headers.Get("Content-Encoding") == "gzip" {
		b, err = gunzip(b)
		if err != nil {
			return nil, nil, err
		}
	}
	return ioutil.NopCloser(bytes.NewReader(b)), b, nil
}

func (tr *proxyTransport) saveRequestBody(track *HTTPTrack, request *http.Request) error {
	var err error
	var body []byte
	request.Body, body, err = tr.saveBody(request.Body, request.Header, request.ContentLength)
	if err != nil {
		return err
	}

	// Save
	track.Body = body
	track.IsJSON = request.Header.Get("Content-Type") == "application/json"
	return nil
}

func (tr *proxyTransport) saveResponseBody(track *HTTPTrack, response *http.Response) error {
	var err error
	var body []byte
	response.Body, body, err = tr.saveBody(response.Body, response.Header, response.ContentLength)
	if err != nil {
		return err
	}

	// Save
	track.Response.Body = body
	track.Response.IsJSON = response.Header.Get("Content-Type") == "application/json"
	return nil
}

func isTextMimeType(contentType string) bool {
	switch {
	case strings.HasPrefix(contentType, "text/"):
		return true
	case strings.HasPrefix(contentType, "application/json"):
		return true
	case strings.HasPrefix(contentType, "application/xml"):
		return true
	case strings.HasPrefix(contentType, "application/atom+xml"):
		return true
	case strings.HasPrefix(contentType, "application/javascript"):
		return true
	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
		return true
	default:
		return false
	}
}

func gunzip(b []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	return ioutil.ReadAll(gz)
}
