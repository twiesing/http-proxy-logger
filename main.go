package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

// Request counter
var reqCounter int32

type DebugTransport struct{}

func decodeBody(encoding string, body []byte) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip":
		r, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	case "deflate":
		r, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(r)
	default:
		return body, nil
	}
}

func (DebugTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	counter := atomic.AddInt32(&reqCounter, 1)

	requestDump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return nil, err
	}
	headers := requestDump
	var body []byte
	if idx := bytes.Index(requestDump, []byte("\r\n\r\n")); idx != -1 {
		headers = requestDump[:idx]
		body = requestDump[idx+4:]
		body = highlightBody(body, r.Header.Get("Content-Type"))
	}
	headers = append(highlightHeaders(headers, true), []byte("\r\n\r\n")...)
	log.Printf("---REQUEST %d---\n\n%s%s\n\n", counter, string(headers), string(body))

	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	// restore body for client
	response.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	headerDump, err := httputil.DumpResponse(response, false)
	if err != nil {
		return nil, err
	}

	decoded, err := decodeBody(response.Header.Get("Content-Encoding"), bodyBytes)
	if err != nil {
		decoded = bodyBytes
	}
	decoded = highlightBody(decoded, response.Header.Get("Content-Type"))

	headerDump = append(highlightHeaders(bytes.TrimSuffix(headerDump, []byte("\r\n\r\n")), false), []byte("\r\n\r\n")...)

	log.Printf("---RESPONSE %d---\n\n%s%s\n\n", counter, string(headerDump), string(decoded))
	// restore body again for proxying
	response.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return response, nil
}

// Get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Get the port to listen on
func getListenAddress() string {
	port := getEnv("PORT", "1338")
	return ":" + port
}

func getTarget() string {
	target := getEnv("TARGET", "http://example.com")
	return target
}

func main() {
	target, _ := url.Parse(getTarget())
	log.Printf("Forwarding %s -> %s\n", getListenAddress(), target)

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Transport = DebugTransport{}

	d := proxy.Director
	proxy.Director = func(r *http.Request) {
		d(r) // call default director

		r.Host = target.Host // set Host header as expected by target
	}

	if err := http.ListenAndServe(getListenAddress(), proxy); err != nil {
		panic(err)
	}
}
