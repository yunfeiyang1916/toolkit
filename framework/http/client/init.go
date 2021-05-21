package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
)

// Deprecated: DefaultClient should not use anymore.
var DefaultClient Client

var defaultHttpClient *http.Client

// httpsInsecureSkipVerify control InsecureSkipVerify arguments
var httpsInsecureSkipVerify = false

func init() {
	if insecure, ok := os.LookupEnv("HTTPS_INSECURE"); ok && insecure == "1" {
		httpsInsecureSkipVerify = true
	}

	tp := &http.Transport{
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		MaxIdleConns:        defaultMaxIdleConns,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAliveTimeout,
			DualStack: true,
		}).DialContext,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
		DisableKeepAlives:     false,
	}

	if httpsInsecureSkipVerify {
		tp.TLSClientConfig = &tls.Config{InsecureSkipVerify: httpsInsecureSkipVerify} // #nosec
	}

	defaultHttpClient = &http.Client{
		Transport: tp,
		// Timeout:   defaultRequestTimeout,
	}
	DefaultClient = NewClient(WithClient(defaultHttpClient))
}
