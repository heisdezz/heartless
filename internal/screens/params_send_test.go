package screens

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	domainreq "heartless/internal/domain/request"
)

// TestParamsPassedOnSend checks the Params tab reaches the server as the query
// string.
func TestParamsPassedOnSend(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	services := newTestServices(t)

	st := shellState{workspaceID: "ws-1"}
	req := domainreq.New("ws-1", "P", domainreq.KindHTTP)
	req.ID = "req-params"
	req.Method = "GET"
	req.URL = srv.URL + "/x"
	st.requestStates = map[string]*requestState{
		req.ID: {
			Method:  "GET",
			URL:     srv.URL + "/x",
			Params:  []*kv{{Key: "page", Value: "2", On: true}, {Key: "q", Value: "hello world", On: true}},
			Headers: []*kv{},
		},
	}

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	if gotQuery != "page=2&q=hello+world" {
		t.Fatalf("server got query %q", gotQuery)
	}
}

// TestFormFieldsPassedOnSend checks a form request sends its form fields as the
// urlencoded body while the query params still reach the URL.
func TestFormFieldsPassedOnSend(t *testing.T) {
	var gotQuery, gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	services := newTestServices(t)

	st := shellState{workspaceID: "ws-1"}
	req := domainreq.New("ws-1", "P", domainreq.KindHTTP)
	req.ID = "req-form"
	req.Method = "POST"
	req.URL = srv.URL + "/login"
	st.requestStates = map[string]*requestState{
		req.ID: {
			Method:   "POST",
			URL:      srv.URL + "/login",
			BodyType: string(domainreq.BodyForm),
			Params:   []*kv{{Key: "next", Value: "/home", On: true}},
			Form:     []*kv{{Key: "user", Value: "ada", On: true}, {Key: "skipped", Value: "x", On: false}},
			Headers:  []*kv{},
		},
	}

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	if gotQuery != "next=%2Fhome" {
		t.Fatalf("server got query %q", gotQuery)
	}
	if gotBody != "user=ada" {
		t.Fatalf("server got body %q", gotBody)
	}
	if gotCT != "application/x-www-form-urlencoded" {
		t.Fatalf("server got content type %q", gotCT)
	}
}

// TestPathParamsPassedOnSend checks a parameter named ":id" fills the matching
// token in the URL path instead of being appended to the query string.
func TestPathParamsPassedOnSend(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	services := newTestServices(t)

	st := shellState{workspaceID: "ws-1"}
	req := domainreq.New("ws-1", "P", domainreq.KindHTTP)
	req.ID = "req-path"
	req.Method = "GET"
	req.URL = srv.URL + "/v1/contacts/:id"
	st.requestStates = map[string]*requestState{
		req.ID: {
			Method:  "GET",
			URL:     srv.URL + "/v1/contacts/:id",
			Params:  []*kv{{Key: ":id", Value: "42", On: true}, {Key: "page", Value: "2", On: true}},
			Headers: []*kv{},
		},
	}

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	if gotPath != "/v1/contacts/42" {
		t.Fatalf("server got path %q", gotPath)
	}
	if gotQuery != "page=2" {
		t.Fatalf("server got query %q", gotQuery)
	}
}
