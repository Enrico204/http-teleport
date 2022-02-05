package telescope

import (
	"gitlab.com/enrico204/http-telescope/service/telescope/transport"
	"net/http/httputil"
	"net/url"
)

type Options struct {
	// StoreBody indicates whether to save the body of requests/responses.
	// The body will be saved only if:
	// - it has explicit Content-Length and Content-Type headers
	// - the content length is less than MaxContentLengthSize
	// - the content mime type is text, JSON or XML
	//
	// Important: this flag will increase the consumed RAM as request/response bodies will be saved in memory
	StoreBody bool

	// DisableCaching overwrites the caching header in responses to "no-cache"
	DisableCaching bool

	// EmbeddedUI controls if the embedded web UI should be available at /_telescope/ inside the proxy. If this value is
	// true, the web dashboard is available in /_telescope/. If this value is false, the web dashboard is not available.
	EmbeddedUI bool
}

func New(target *url.URL, options Options) (*Telescope, error) {
	tr := transport.New(options.StoreBody, options.DisableCaching)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = tr
	return &Telescope{
		embeddedUI: options.EmbeddedUI,
		transport:  tr,
		proxy:      proxy,
		target:     target,
	}, nil
}

type Telescope struct {
	embeddedUI bool

	transport transport.ProxyTransport
	proxy     *httputil.ReverseProxy
	target    *url.URL
}
