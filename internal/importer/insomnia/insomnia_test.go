package insomnia

import (
	"strings"
	"testing"

	"heartless/internal/importer"
)

const v4Export = `{
  "_type": "export",
  "__export_format": 4,
  "__export_date": "2024-01-01T00:00:00.000Z",
  "resources": [
    {"_type": "workspace", "_id": "ws_1", "name": "Insomnia Workspace", "description": "desc"},
    {"_type": "environment", "_id": "env_1", "parentId": "ws_1", "name": "Base Env", "data": {"baseUrl": "https://x"}},
    {"_type": "request_group", "_id": "grp_1", "parentId": "ws_1", "name": "Auth"},
    {"_type": "request_group", "_id": "grp_2", "parentId": "grp_1", "name": "Tokens"},
    {
      "_type": "request", "_id": "req_1", "parentId": "grp_2",
      "name": "Get token", "description": "token desc", "method": "POST",
      "url": "{{ _.baseUrl }}/token",
      "headers": [
        {"name": "Content-Type", "value": "application/json"},
        {"name": "X-Off", "value": "1", "disabled": true}
      ],
      "parameters": [{"name": "verbose", "value": "1", "disabled": true}],
      "body": {"mimeType": "application/json", "text": "{\"grant\":\"client\"}"},
      "authentication": {"type": "bearer", "token": "abc"}
    },
    {
      "_type": "request", "_id": "req_2", "parentId": "ws_1",
      "name": "List", "method": "GET", "url": "{{ _.baseUrl }}/items",
      "headers": [], "parameters": [{"name": "q", "value": "x"}]
    },
    {
      "_type": "request", "_id": "req_orphan", "parentId": "gone",
      "name": "Orphan", "method": "GET", "url": "https://x/orphan"
    },
    {"_type": "grpc_request", "_id": "grpc_1", "parentId": "ws_1", "name": "RPC", "url": "localhost:50051", "protoMethodName": "pkg.Svc/Do"},
    {"_type": "websocket_request", "_id": "wsr_1", "parentId": "ws_1", "name": "Socket", "url": "wss://x/ws"}
  ]
}`

const v5Export = `{
  "type": "collection",
  "name": "V5 Collection",
  "meta": {"id": "col_1", "description": "v5 desc"},
  "environments": {"id": "env_root", "name": "Base", "data": {"url": "https://x"}, "subEnvironments": []},
  "headers": [{"name": "X-Ws", "value": "1"}],
  "collection": [
    {
      "meta": {"id": "f1"},
      "name": "Folder A",
      "children": [
        {
          "meta": {"id": "r1"},
          "name": "Ping",
          "method": "GET",
          "url": "https://x/ping",
          "headers": [{"name": "Accept", "value": "text/plain"}],
          "parameters": [],
          "authentication": {"type": "basic", "username": "u", "password": "p"}
        },
        {"meta": {"id": "g1"}, "name": "Rpc", "protoFileId": "pf1", "url": "localhost:1", "protoMethodName": "a.B/C"},
        {"meta": {"id": "w1"}, "name": "Socket", "url": "wss://x"}
      ]
    },
    {
      "meta": {"id": "r2"},
      "name": "Root req",
      "method": "POST",
      "url": "https://x/p",
      "body": {"mimeType": "application/x-www-form-urlencoded", "params": [
        {"name": "a", "value": "1"},
        {"name": "b", "value": "2", "disabled": true}
      ]}
    }
  ]
}`

const postmanJSON = `{"info":{"schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json","name":"P"},"item":[]}`

func TestDetect(t *testing.T) {
	imp := Importer{}
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"v4 export", v4Export, true},
		{"v5 export", v5Export, true},
		{"postman collection", postmanJSON, false},
		{"garbage", "{{{", false},
		{"unrelated json", `{"foo": 1}`, false},
	}
	for _, tc := range cases {
		if got := imp.Detect([]byte(tc.data)); got != tc.want {
			t.Errorf("%s: Detect = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestParseV4(t *testing.T) {
	draft, err := Importer{}.Parse([]byte(v4Export))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if draft.Format != importer.FormatInsomnia {
		t.Fatalf("format = %q", draft.Format)
	}
	if len(draft.Collections) != 1 {
		t.Fatalf("collections = %d, want 1", len(draft.Collections))
	}
	col := draft.Collections[0]
	if col.Name != "Insomnia Workspace" || col.Description != "desc" {
		t.Fatalf("collection = %+v", col)
	}

	// Workspace children: [group Auth, request List]; grpc/websocket/environment skipped.
	if len(col.Items) != 2 {
		t.Fatalf("items = %d, want 2 (skipped resources excluded): %+v", len(col.Items), col.Items)
	}
	authFolder := col.Items[0].Folder
	if authFolder == nil || authFolder.Name != "Auth" {
		t.Fatalf("items[0] = %+v, want folder Auth", col.Items[0])
	}
	if len(col.Items[0].Children) != 1 {
		t.Fatalf("Auth children = %d, want 1", len(col.Items[0].Children))
	}
	tokens := col.Items[0].Children[0].Folder
	if tokens == nil || tokens.Name != "Tokens" {
		t.Fatalf("Auth child = %+v, want folder Tokens", col.Items[0].Children[0])
	}
	if len(col.Items[0].Children[0].Children) != 1 {
		t.Fatalf("Tokens children = %d, want 1", len(col.Items[0].Children[0].Children))
	}
	getToken := col.Items[0].Children[0].Children[0].Request
	if getToken == nil {
		t.Fatal("Get token request missing")
	}
	if getToken.Method != "POST" || getToken.URL != "{{ _.baseUrl }}/token" {
		t.Fatalf("Get token = %+v", getToken)
	}
	if getToken.Description != "token desc" {
		t.Fatalf("description = %q", getToken.Description)
	}
	if len(getToken.Headers) != 2 || !getToken.Headers[0].Enabled || getToken.Headers[1].Enabled {
		t.Fatalf("headers = %+v", getToken.Headers)
	}
	if len(getToken.Params) != 1 || getToken.Params[0].Enabled {
		t.Fatalf("params = %+v", getToken.Params)
	}
	if getToken.Body != `{"grant":"client"}` || getToken.BodyType != "application/json" {
		t.Fatalf("body = %q type %q", getToken.Body, getToken.BodyType)
	}

	// Root-level request.
	list := col.Items[1].Request
	if list == nil || list.Name != "List" || list.Method != "GET" {
		t.Fatalf("items[1] = %+v, want request List", col.Items[1])
	}
	if len(list.Params) != 1 || list.Params[0].Name != "q" || !list.Params[0].Enabled {
		t.Fatalf("List params = %+v", list.Params)
	}

	// Unsupported features and orphans are reported.
	wantIssues := []struct{ severity, path, contains string }{
		{importer.SeverityUnsupported, "environment", "environment"},
		{importer.SeverityUnsupported, "grpc_request", "grpc_request"},
		{importer.SeverityUnsupported, "websocket_request", "websocket_request"},
		{importer.SeverityUnsupported, "resources[1][0][0].authentication", "bearer"},
		{importer.SeverityWarning, "resources", "missing parent"},
	}
	for _, want := range wantIssues {
		if !hasIssue(draft.Issues, want.severity, want.path, want.contains) {
			t.Errorf("missing issue severity=%s path=%s containing %q; issues=%+v",
				want.severity, want.path, want.contains, draft.Issues)
		}
	}
}

func TestParseV5(t *testing.T) {
	draft, err := Importer{}.Parse([]byte(v5Export))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(draft.Collections) != 1 {
		t.Fatalf("collections = %d, want 1", len(draft.Collections))
	}
	col := draft.Collections[0]
	if col.Name != "V5 Collection" || col.Description != "v5 desc" {
		t.Fatalf("collection = %+v", col)
	}
	if len(col.Items) != 2 {
		t.Fatalf("items = %d, want 2: %+v", len(col.Items), col.Items)
	}
	folder := col.Items[0].Folder
	if folder == nil || folder.Name != "Folder A" {
		t.Fatalf("items[0] = %+v, want folder Folder A", col.Items[0])
	}
	// grpc + websocket children are skipped, only Ping remains.
	if len(col.Items[0].Children) != 1 {
		t.Fatalf("Folder A children = %d, want 1: %+v", len(col.Items[0].Children), col.Items[0].Children)
	}
	ping := col.Items[0].Children[0].Request
	if ping == nil || ping.Name != "Ping" || ping.Method != "GET" || ping.URL != "https://x/ping" {
		t.Fatalf("Ping = %+v", ping)
	}
	if len(ping.Headers) != 1 || ping.Headers[0].Name != "Accept" {
		t.Fatalf("Ping headers = %+v", ping.Headers)
	}

	root := col.Items[1].Request
	if root == nil || root.Name != "Root req" || root.Method != "POST" {
		t.Fatalf("items[1] = %+v", root)
	}
	if root.Body != "a=1" || root.BodyType != "application/x-www-form-urlencoded" {
		t.Fatalf("Root req body = %q type %q", root.Body, root.BodyType)
	}

	wantIssues := []struct{ severity, path, contains string }{
		{importer.SeverityUnsupported, "environments", "environments"},
		{importer.SeverityUnsupported, "headers", "workspace-level headers"},
		{importer.SeverityUnsupported, "collection[0][0].authentication", "basic"},
		{importer.SeverityUnsupported, "grpc_request", "grpc_request"},
		{importer.SeverityUnsupported, "websocket_request", "websocket_request"},
	}
	for _, want := range wantIssues {
		if !hasIssue(draft.Issues, want.severity, want.path, want.contains) {
			t.Errorf("missing issue severity=%s path=%s containing %q; issues=%+v",
				want.severity, want.path, want.contains, draft.Issues)
		}
	}
}

func TestParseRejectsUnknownDocument(t *testing.T) {
	imp := Importer{}
	if _, err := imp.Parse([]byte(`{"name":"nothing here"}`)); err == nil {
		t.Fatal("expected error for document with neither resources nor collection")
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
