package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// reqCounter zählt fortlaufend alle Request/Response-Paare.
var reqCounter int32

// CLI-Flags
var (
	logRequests  = flag.Bool("requests", true, "log HTTP requests")
	logResponses = flag.Bool("responses", true, "log HTTP responses")
	cliTarget    = flag.String("target", "", "upstream target URL (overrides TARGET)")
	cliPort      = flag.String("port", "", "listen port (overrides PORT)")
)

// DebugTransport loggt Request und Response (mit Dekompression und Hervorhebung).
type DebugTransport struct{}

func decodeBody(enc string, body []byte) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(enc)) {
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

	// Request-Dump
	reqDump, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return nil, err
	}
	var reqBody []byte
	headers := reqDump
	if idx := bytes.Index(reqDump, []byte("\r\n\r\n")); idx != -1 {
		headers = reqDump[:idx]
		reqBody = reqDump[idx+4:]
		reqBody = highlightBody(reqBody, r.Header.Get("Content-Type"))
	}
	headers = append(highlightHeaders(headers, true), []byte("\r\n\r\n")...)
	if *logRequests {
		log.Printf("[REQUEST %d]\n%s%s\n", counter, headers, reqBody)
	}

	// Weiterleiten an Upstream
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}

	// Response-Dump
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(respBytes))

	headerDump, _ := httputil.DumpResponse(resp, false)
	decoded, _ := decodeBody(resp.Header.Get("Content-Encoding"), respBytes)
	decoded = highlightBody(decoded, resp.Header.Get("Content-Type"))
	headerDump = append(highlightHeaders(bytes.TrimSuffix(headerDump, []byte("\r\n\r\n")), false), []byte("\r\n\r\n")...)

	if *logResponses {
		log.Printf("[RESPONSE %d: %s]\n%s%s\n", counter, resp.Status, headerDump, decoded)
	}

	resp.Body = io.NopCloser(bytes.NewReader(respBytes))
	return resp, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getListenAddress() string {
	port := *cliPort
	if port == "" {
		port = getEnv("PORT", "1338")
	}
	return ":" + port
}

func getTarget() string {
	t := *cliTarget
	if t == "" {
		t = getEnv("TARGET", "http://example.com")
	}
	return t
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	targetURL, _ := url.Parse(getTarget())
	log.Printf("Proxy listening on %s → %s\n", getListenAddress(), targetURL)

	// Transport mit Debug-Logging
	transport := DebugTransport{}

	// SingleHost-Proxy bleibt als Basis, aber wir verwenden eigenen Handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Original-Body einlesen
		origBody, _ := io.ReadAll(r.Body)
		r.Body.Close()

		// 1) Erst-Call mit stream=true
		resp, err := doUpstreamCall(r, origBody, true, transport)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// 2) Prüfen: kam jemals delta.content?
		if isEmptyStream(resp.Body) {
			resp.Body.Close()
			// 3) Fallback: stream=false
			resp, err = doUpstreamCall(r, origBody, false, transport)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
		}

		// 4) Antwort 1:1 weiterleiten
		copyResponse(w, resp)
	})

	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		log.Fatal(err)
	}
}

// doUpstreamCall dupliziert das Request-JSON, setzt den stream-Flag und schickt es upstream.
func doUpstreamCall(origReq *http.Request, origBody []byte, useStream bool, transport http.RoundTripper) (*http.Response, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(origBody, &m); err != nil {
		return nil, err
	}
	m["stream"] = useStream
	newBody, _ := json.Marshal(m)

	req := origReq.Clone(origReq.Context())
	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))

	return transport.RoundTrip(req)
}

// isEmptyStream scannt die ersten SSE-Chunks auf non-empty delta.content.
func isEmptyStream(rc io.ReadCloser) bool {
	defer rc.Close()
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimPrefix(line, "data:")
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err == nil {
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					return false
				}
			}
		}
	}
	return true
}

// copyResponse leitet Header und Body der Upstream-Response an den Client weiter.
func copyResponse(w http.ResponseWriter, resp *http.Response) {
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()
}
