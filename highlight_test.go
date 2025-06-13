package main

import (
	"strings"
	"testing"
)

func TestHighlightBodyJSON(t *testing.T) {
	data := []byte(`{"name":"Alice"}`)
	out := string(highlightBody(data, "application/json"))
	if out == string(data) {
		t.Fatalf("expected JSON body to be highlighted")
	}
	if !strings.Contains(out, colorKey) || !strings.Contains(out, colorString) {
		t.Errorf("highlighted JSON missing colors: %q", out)
	}
}

func TestHighlightBodyXML(t *testing.T) {
	data := []byte(`<p>Hello</p>`)
	out := string(highlightBody(data, "application/xml"))
	if out == string(data) {
		t.Fatalf("expected XML body to be highlighted")
	}
	if !strings.Contains(out, colorTag) {
		t.Errorf("highlighted XML missing tag color: %q", out)
	}
}

func TestHighlightHeadersRequest(t *testing.T) {
	headers := []byte("POST /foo HTTP/1.1\r\nHost: example.com\r\n\r\n")
	out := string(highlightHeaders(headers, true))
	if !strings.Contains(out, colorMethod) || !strings.Contains(out, colorURL) {
		t.Errorf("request line not highlighted: %q", out)
	}
	if !strings.Contains(out, colorHeader) {
		t.Errorf("header key not highlighted: %q", out)
	}
}

func TestHighlightHeadersResponse(t *testing.T) {
	headers := []byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\n\r\n")
	out := string(highlightHeaders(headers, false))
	if !strings.Contains(out, colorStatus4xx) {
		t.Errorf("status not colorized: %q", out)
	}
	if !strings.Contains(out, colorHeader) {
		t.Errorf("header key not highlighted: %q", out)
	}
}

func TestColorStatus(t *testing.T) {
	if colorStatus(201) != colorStatus2xx {
		t.Errorf("expected 2xx color")
	}
	if colorStatus(302) != colorStatus3xx {
		t.Errorf("expected 3xx color")
	}
	if colorStatus(404) != colorStatus4xx {
		t.Errorf("expected 4xx color")
	}
	if colorStatus(500) != colorStatus5xx {
		t.Errorf("expected 5xx color")
	}
}
