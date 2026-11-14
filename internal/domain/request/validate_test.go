package request

import "testing"

func TestValidateHTTP_OK(t *testing.T) {
	r := NewHTTP("ws1", "Get users")
	r.URL = "https://api.example.com/users"
	errs := ValidateHTTP(r, DefaultHTTPSettings())
	if errs.Has() {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateHTTP_Errors(t *testing.T) {
	r := &Request{Name: " ", Method: "FETCH", URL: "notaurl", Kind: KindGRPC}
	errs := ValidateHTTP(r, HTTPSettings{Timeout: -1, MaxRedirects: -1})
	if !errs.Has() {
		t.Fatal("expected validation errors")
	}
	fields := map[string]bool{}
	for _, e := range errs {
		fields[e.Field] = true
	}
	for _, f := range []string{"workspace_id", "name", "kind", "method", "url", "settings.timeout", "settings.max_redirects"} {
		if !fields[f] {
			t.Fatalf("expected error for field %q, got %+v", f, errs)
		}
	}
	if errs.Error() == "" {
		t.Fatal("expected joined error string")
	}
}
