package update

import (
	"net"
	"net/http"
	"time"
)

const (
	updateConnectTimeout        = 30 * time.Second
	updateTLSHandshakeTimeout   = 30 * time.Second
	updateResponseHeaderTimeout = 2 * time.Minute
)

// updateHTTPClient returns an HTTP client suitable for GitHub release downloads.
// Wails' default github provider uses a 30s client timeout, which aborts large
// installer downloads well before they finish. Timeout 0 allows streaming until
// the caller's context expires.
func updateHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   updateConnectTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			TLSHandshakeTimeout:   updateTLSHandshakeTimeout,
			ResponseHeaderTimeout: updateResponseHeaderTimeout,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}
