package postman

import (
	"strings"
	"testing"

	"heartless/internal/importer"
)

const sampleCollection = `{
  "info": {
    "name": "Sample API",
    "description": {"content": "A sample collection", "type": "text/plain"},
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "variable": [{"key": "baseUrl", "value": "https://api.example.com"}],
  "auth": {"type": "bearer", "bearer": [{"key": "token", "value": "{{token}}"}]},
  "event": [{"listen": "prerequest", "script": {"type": "text/javascript", "exec": []}}],
  "item": [
    {
      "name": "Users",
      "item": [
        {
          "name": "List users",
          "request": {
            "method": "GET",
            "auth": {"type": "noauth"},
            "header": [
              {"key": "Accept", "value": "application/json"},
              {"key": "X-Debug", "value": "1", "disabled": true},
              {"key": "X-Dup", "value": "first"},
              {"key": "X-Dup", "value": "second"}
            ],
            "url": {
              "protocol": "https",
              "host": ["api", "example", "com"],
              "port": "8443",
              "path": ["api", "v1", "users"],
              "query": [
                {"key": "page", "value": "1"},
                {"key": "debug", "value": "true", "disabled": true}
              ],
              "variable": [{"key": "userId", "value": "42"}]
            }
          }
        },
        {
          "name": "Create user",
          "request": {
            "method": "POST",
            "header": [{"key": "Content-Type", "value": "application/json"}],
            "body": {"mode": "raw", "raw": "{\"name\":\"x\"}", "options": {"raw": {"language": "json"}}},
            "url": "https://api.example.com/api/v1/users"
          }
        }
      ]
    },
    {
      "name": "Health",
      "request": {
        "method": "GET",
        "url": "https://api.example.com/health",
        "auth": {"type": "basic", "basic": [{"key": "username", "value": "u"}]}
      }
    },
    {"name": "Empty folder", "item": []}
  ]
}`

const foreignJSON = `{"_type":"export","__export_format":4,"resources":[]}`

func TestDetect(t *testing.T) {
	imp := Importer{}
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"v2.1 collection", sampleCollection, true},
		{"insomnia export", foreignJSON, false},
		{"garbage", "not json at all", false},
		{"json without schema", `{"info":{"name":"x"},"item":[]}`, false},
		{"schema but no item", `{"info":{"schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"}}`, false},
	}
	for _, tc := range cases {
		if got := imp.Detect([]byte(tc.data)); got != tc.want {
			t.Errorf("%s: Detect = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestParseSampleCollection(t *testing.T) {
	draft, err := Importer{}.Parse([]byte(sampleCollection))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if draft.Format != importer.FormatPostman {
		t.Fatalf("format = %q", draft.Format)
	}
	if draft.Name != "Sample API" {
		t.Fatalf("draft name = %q", draft.Name)
	}
	if draft.Description != "A sample collection" {
		t.Fatalf("description = %q", draft.Description)
	}
	if len(draft.Collections) != 1 {
		t.Fatalf("collections = %d, want 1", len(draft.Collections))
	}
	col := draft.Collections[0]
	if col.Name != "Sample API" {
		t.Fatalf("collection name = %q", col.Name)
	}
	if len(col.Items) != 3 {
		t.Fatalf("items = %d, want 3", len(col.Items))
	}

	// Folder "Users" with two nested requests.
	if col.Items[0].Folder == nil || col.Items[0].Folder.Name != "Users" {
		t.Fatalf("items[0] = %+v, want folder Users", col.Items[0].Folder)
	}
	if len(col.Items[0].Children) != 2 {
		t.Fatalf("folder children = %d, want 2", len(col.Items[0].Children))
	}
	list := col.Items[0].Children[0].Request
	if list == nil {
		t.Fatal("first child is not a request")
	}
	if list.Method != "GET" {
		t.Fatalf("method = %q", list.Method)
	}
	if want := "https://api.example.com:8443/api/v1/users"; list.URL != want {
		t.Fatalf("url = %q, want %q", list.URL, want)
	}
	if len(list.Headers) != 4 {
		t.Fatalf("headers = %d, want 4 (duplicates preserved for service layer)", len(list.Headers))
	}
	if !list.Headers[0].Enabled || list.Headers[1].Enabled {
		t.Fatalf("header enabled flags = %v/%v", list.Headers[0].Enabled, list.Headers[1].Enabled)
	}
	if list.Headers[2].Value != "first" {
		t.Fatalf("first duplicate header value = %q", list.Headers[2].Value)
	}
	if len(list.Params) != 3 {
		t.Fatalf("params = %d, want query(2) + path variable(1)", len(list.Params))
	}
	if list.Params[0].Name != "page" || !list.Params[0].Enabled {
		t.Fatalf("page param = %+v", list.Params[0])
	}
	if list.Params[1].Enabled {
		t.Fatalf("disabled query param must be disabled: %+v", list.Params[1])
	}
	if list.Params[2].Name != ":userId" || list.Params[2].Value != "42" {
		t.Fatalf("path variable param = %+v", list.Params[2])
	}

	create := col.Items[0].Children[1].Request
	if create == nil {
		t.Fatal("second child is not a request")
	}
	if create.BodyType != "application/json" || create.Body != `{"name":"x"}` {
		t.Fatalf("body = %q type %q", create.Body, create.BodyType)
	}

	// Top-level request and empty folder.
	if col.Items[1].Request == nil || col.Items[1].Request.Name != "Health" {
		t.Fatalf("items[1] = %+v, want request Health", col.Items[1].Request)
	}
	if col.Items[2].Folder == nil || col.Items[2].Folder.Name != "Empty folder" || len(col.Items[2].Children) != 0 {
		t.Fatalf("items[2] = %+v, want empty folder", col.Items[2])
	}

	// Unsupported features must be reported.
	wantIssues := []struct{ severity, path, contains string }{
		{importer.SeverityUnsupported, "variable", "collection variable"},
		{importer.SeverityUnsupported, "event", "scripts"},
		{importer.SeverityUnsupported, "auth", `bearer`},
		{importer.SeverityUnsupported, "item[1].request.auth", `basic`},
	}
	for _, want := range wantIssues {
		if !hasIssue(draft.Issues, want.severity, want.path, want.contains) {
			t.Errorf("missing issue severity=%s path=%s containing %q; issues=%+v",
				want.severity, want.path, want.contains, draft.Issues)
		}
	}
	// noauth must not be reported.
	for _, is := range draft.Issues {
		if strings.Contains(is.Message, "noauth") {
			t.Errorf("noauth must not produce an issue: %+v", is)
		}
	}
}

func TestParseBodyModes(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantBody  string
		wantType  string
		wantIssue string
	}{
		{
			name:     "raw json",
			body:     `{"mode":"raw","raw":"{\"a\":1}","options":{"raw":{"language":"json"}}}`,
			wantBody: `{"a":1}`, wantType: "application/json",
		},
		{
			name:     "raw text",
			body:     `{"mode":"raw","raw":"hello"}`,
			wantBody: "hello", wantType: "text/plain",
		},
		{
			name:     "urlencoded skips disabled",
			body:     `{"mode":"urlencoded","urlencoded":[{"key":"a","value":"1"},{"key":"b","value":"2","disabled":true}]}`,
			wantBody: "a=1", wantType: "application/x-www-form-urlencoded",
		},
		{
			name:      "formdata keeps enabled flags, drops file fields",
			body:      `{"mode":"formdata","formdata":[{"key":"f","value":"v"},{"key":"file","src":"/tmp/x"}]}`,
			wantBody:  `[{"name":"f","value":"v","enabled":true}]`,
			wantType:  "multipart/form-data",
			wantIssue: "file form field",
		},
		{
			name:     "graphql envelope",
			body:     `{"mode":"graphql","graphql":{"query":"{me{id}}","variables":"{\"id\":1}"}}`,
			wantBody: `{"query":"{me{id}}","variables":{"id":1}}`, wantType: "graphql",
		},
		{
			name:      "file body unsupported",
			body:      `{"mode":"file","file":{"src":"/tmp/x"}}`,
			wantIssue: "binary file body",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := `{"info":{"schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json","name":"T"},` +
				`"item":[{"name":"R","request":{"method":"GET","url":"http://x","body":` + tc.body + `}}]}`
			draft, err := Importer{}.Parse([]byte(doc))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			req := draft.Collections[0].Items[0].Request
			if req == nil {
				t.Fatal("request missing")
			}
			if tc.wantIssue == "" {
				if req.Body != tc.wantBody || req.BodyType != tc.wantType {
					t.Fatalf("body = %q type %q, want %q / %q", req.Body, req.BodyType, tc.wantBody, tc.wantType)
				}
			}
			if tc.wantIssue != "" && !issueContains(draft.Issues, tc.wantIssue) {
				t.Fatalf("want issue containing %q, got %+v", tc.wantIssue, draft.Issues)
			}
		})
	}
}

func hasIssue(issues []importer.Issue, severity, path, contains string) bool {
	for _, is := range issues {
		if is.Severity == severity && is.Path == path && strings.Contains(is.Message, contains) {
			return true
		}
	}
	return false
}

func issueContains(issues []importer.Issue, contains string) bool {
	for _, is := range issues {
		if strings.Contains(is.Message, contains) {
			return true
		}
	}
	return false
}
