package transport

import (
	"net/http"
	"sync"
	"time"
)

const (
	MaxContentLengthSize = 10 * 1024
	CircularBufferSize   = 10
)

type ProxyTransport interface {
	GetTracks() []*HTTPTrack
	RoundTrip(request *http.Request) (*http.Response, error)
}

func New(storeBody bool, disableCaching bool) ProxyTransport {
	return &proxyTransport{
		storeBody:      storeBody,
		disableCaching: disableCaching,
		tracks:         make([]*HTTPTrack, CircularBufferSize),
	}
}

type proxyTransport struct {
	disableCaching bool
	storeBody      bool
	tracks         []*HTTPTrack
	tracksIndex    int
	tracksMutex    sync.RWMutex
}

func (tr *proxyTransport) GetTracks() []*HTTPTrack {
	tr.tracksMutex.RLock()
	defer tr.tracksMutex.RUnlock()

	// Create a new slice with ordered items from the circular buffer
	// Iterate until "i" reaches the size of the buffer. Skip positions with nil
	var ret = make([]*HTTPTrack, 0, len(tr.tracks))
	for i := 1; i <= CircularBufferSize; i++ {
		idx := (tr.tracksIndex - i) % CircularBufferSize
		if idx < 0 {
			idx *= -1
		}
		if tr.tracks[idx] != nil {
			ret = append(ret, tr.tracks[idx])
		}
	}
	return ret
}

func (tr *proxyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	var track = HTTPTrack{
		Method:        request.Method,
		URL:           request.URL,
		Proto:         request.Proto,
		Header:        request.Header.Clone(),
		ContentLength: request.ContentLength,
		RemoteAddr:    request.RemoteAddr,
		When:          time.Now(),
	}
	err := tr.saveRequestBody(&track, request)
	if err != nil {
		return nil, err
	}

	tr.tracksMutex.Lock()
	tr.tracks[tr.tracksIndex] = &track
	tr.tracksIndex = (tr.tracksIndex + 1) % CircularBufferSize
	tr.tracksMutex.Unlock()

	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		track.Response = &HTTPTrackResponse{
			Error: err,
			When:  time.Now(),
		}
		return nil, err
	}

	track.Response = &HTTPTrackResponse{
		Status:        response.Status,
		StatusCode:    response.StatusCode,
		Proto:         response.Proto,
		Header:        response.Header.Clone(),
		ContentLength: response.ContentLength,
		When:          time.Now(),
	}
	err = tr.saveResponseBody(&track, response)
	if err != nil {
		return nil, err
	}

	if tr.disableCaching {
		response.Header.Set("Cache-control", "no-cache")
	}

	return response, err
}
