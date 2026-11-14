package http

import (
	"context"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	domainreq "heartless/internal/domain/request"
)

func TestClientExecute(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Query().Get("a") != "1" || r.Header.Get("X-Test") != "v" {
			t.Fatalf("unexpected request: %s %v", r.URL.String(), r.Header)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != "hello" {
			t.Fatalf("body=%q", string(b))
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(201)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient()
	req := domainreq.NewHTTP("ws1", "test")
	req.Method = "POST"
	req.URL = srv.URL
	req.Params = map[string]string{"a": "1"}
	req.Headers = map[string]string{"X-Test": "v"}
	req.Body = "hello"

	res, err := c.Execute(context.Background(), req, domainreq.DefaultHTTPSettings())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.StatusCode != 201 || string(res.Body) != "ok" || res.ContentType != "text/plain" {
		t.Fatalf("unexpected response: %+v body=%q", res, string(res.Body))
	}
}

func TestClientCancel(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		<-block
	}))
	defer func() { close(block); srv.Close() }()

	c := NewClient()
	req := domainreq.NewHTTP("ws1", "test")
	req.URL = srv.URL
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := c.Execute(ctx, req, domainreq.DefaultHTTPSettings())
	if err == nil {
		t.Fatal("expected error")
	}
}
