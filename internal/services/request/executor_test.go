package request

import (
	"context"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	domain "heartless/internal/domain/request"
	domainvar "heartless/internal/domain/variable"
	varres "heartless/internal/domain/variable-resolution"
	transport "heartless/internal/transport/http"
)

func TestExecutorResolveValidateExecute(t *testing.T) {
	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path != "/users" || r.Header.Get("X-Token") != "abc" {
			t.Fatalf("unexpected request: %s %v", r.URL.String(), r.Header)
		}
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	env := varres.NewMemoryVariableSet()
	env.Add(*domainvar.NewVariable("base", srv.URL, "text", false))
	env.Add(*domainvar.NewVariable("token", "abc", "text", false))
	resolver := varres.NewResolver(varres.NewMemoryVariableSet(), varres.NewMemoryVariableSet(), varres.NewMemoryVariableSet(), env, varres.NewMemoryVariableSet(), varres.NewMemoryVariableSet())
	exec := NewExecutor(transport.NewClient(), resolver, domain.DefaultHTTPSettings())

	req := domain.NewHTTP("ws1", "list")
	req.URL = "{{base}}/users"
	req.Method = "POST"
	req.Headers = map[string]string{"X-Token": "{{token}}"}
	req.Body = "hi"
	origURL := req.URL

	result, err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Error != nil || result.Response == nil || result.Response.StatusCode != 200 {
		t.Fatalf("unexpected result: %+v resp=%+v", result, result.Response)
	}
	if string(result.Response.Body) != "hi" {
		t.Fatalf("body=%q", string(result.Response.Body))
	}
	if req.URL != origURL {
		t.Fatalf("original request mutated: %q", req.URL)
	}
}

func TestExecutorValidation(t *testing.T) {
	exec := NewExecutor(transport.NewClient(), nil, domain.DefaultHTTPSettings())
	req := &domain.Request{Name: "bad"}
	result, err := exec.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if result == nil || result.Error == nil {
		t.Fatalf("expected validation error result: %+v", result)
	}
}
