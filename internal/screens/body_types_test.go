package screens

import (
	"testing"

	domainreq "heartless/internal/domain/request"
)

// TestJSObjectToJSON checks JavaScript object literals become strict JSON.
func TestJSObjectToJSON(t *testing.T) {
	cases := map[string]string{
		`{"a": 1}`:                          `{"a": 1}`,
		`{a: 1, b: "x"}`:                    `{"a": 1, "b": "x"}`,
		"{a: 1, b: 'x',}":                   `{"a": 1, "b": "x"}`,
		`{a: {b: [1, 2,],},}`:               `{"a": {"b": [1, 2]}}`,
		`{name: "Ada", tags: ["a", "b"]}`:   `{"name": "Ada", "tags": ["a", "b"]}`,
		`not json at all`:                   `not json at all`,
		`{broken: `:                         `{broken: `,
		`{"nested": {"ok": true},}`:         `{"nested": {"ok": true}}`,
		`{count: 3, ratio: 1.5, on: false}`: `{"count": 3, "ratio": 1.5, "on": false}`,
	}
	for in, want := range cases {
		if got := jsObjectToJSON(in); got != want {
			t.Errorf("jsObjectToJSON(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBuildDomainRequestBodyTypes checks each body type sets the body and
// content type sent.
func TestBuildDomainRequestBodyTypes(t *testing.T) {
	mk := func(bodyType, body string) *domainreq.Request {
		rs := &requestState{
			Method:   "POST",
			URL:      "https://api.test/x",
			Body:     body,
			BodyType: bodyType,
			Params:   []*kv{{Key: "limit", Value: "10", On: true}, {Key: "off", Value: "1", On: false}},
			Headers:  []*kv{},
		}
		req := domainreq.New("ws-1", "Req", domainreq.KindHTTP)
		return rs.buildDomainRequest(*req)
	}

	// JSON: JavaScript object is converted and the header set.
	j := mk(string(domainreq.BodyJSON), `{name: "Ada"}`)
	if j.Body != `{"name": "Ada"}` {
		t.Errorf("json body = %q", j.Body)
	}
	if j.Headers["Content-Type"] != "application/json" {
		t.Errorf("json content type = %q", j.Headers["Content-Type"])
	}

	// No body clears the body.
	n := mk(string(domainreq.BodyNone), `{"x":1}`)
	if n.Body != "" {
		t.Errorf("none body = %q, want empty", n.Body)
	}

	// Url encoded: enabled keys only.
	f := mk(string(domainreq.BodyForm), "")
	if f.Body != "limit=10" {
		t.Errorf("form body = %q", f.Body)
	}
	if f.Headers["Content-Type"] != "application/x-www-form-urlencoded" {
		t.Errorf("form content type = %q", f.Headers["Content-Type"])
	}

	// Multipart keeps its header and body as typed.
	m := mk(string(domainreq.BodyMultipart), "raw")
	if m.Headers["Content-Type"] != "multipart/form-data" {
		t.Errorf("multipart content type = %q", m.Headers["Content-Type"])
	}

	// A user-set Content-Type is not overwritten.
	rs := &requestState{
		Method: "POST", URL: "https://api.test/x", Body: "x", BodyType: string(domainreq.BodyJSON),
		Params: []*kv{}, Headers: []*kv{{Key: "content-type", Value: "application/vnd.api+json", On: true}},
	}
	req := domainreq.New("ws-1", "Req", domainreq.KindHTTP)
	custom := rs.buildDomainRequest(*req)
	if custom.Headers["content-type"] != "application/vnd.api+json" {
		t.Errorf("custom content type = %q", custom.Headers["content-type"])
	}
	if _, set := custom.Headers["Content-Type"]; set {
		t.Errorf("Content-Type was overwritten")
	}

	// GraphQL gets the JSON content type.
	g := mk(string(domainreq.BodyGraphQL), "query { a }")
	if g.Headers["Content-Type"] != "application/json" {
		t.Errorf("graphql content type = %q", g.Headers["Content-Type"])
	}
}
