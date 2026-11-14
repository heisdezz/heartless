package screens

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domainreq "heartless/internal/domain/request"
)

// TestRequestPaneTypingKeepsEdits checks that typing into the URL and a param row
// changes the request's persistent state across frames.
func TestRequestPaneTypingKeepsEdits(t *testing.T) {
	services := newTestServices(t)
	st := shellState{workspaceID: "ws-1"}
	req := *domainreq.New("ws-1", "Alpha", domainreq.KindHTTP)
	req.ID = "req-1"
	req.Method = "GET"

	rs := st.requestStateFor(req)
	rs.URL = ""

	tt := ui.NewTester(func(c *ui.Context) {
		requestPane(c, theme.TokensOf(c), &st, services, req)
	}, 900, 700)
	tt.Frame()

	// The URL field sits right of the method select in the top row.
	sendRect, _ := tt.Find("Send")
	tt.ClickAt((sendRect.X)/2, sendRect.Y+10)
	tt.Type("{{baseUrl}}/users")
	tt.Frame()
	tt.Frame()
	if rs.URL != "{{baseUrl}}/users" {
		t.Fatalf("URL = %q, want typed text", rs.URL)
	}

	// The first param row's key field is left of its value field, same row.
	if len(rs.Params) == 0 {
		t.Fatalf("no param rows")
	}
	row := rs.Params[0]
	checkRect, _ := tt.Find("Params")
	_ = checkRect
	// Find the row: the "+ Add" button sits just below it.
	addRect, ok := tt.Find("+ Add")
	if !ok {
		t.Fatalf("no + Add button")
	}
	tt.ClickAt(150, addRect.Y-24)
	tt.Type("limit")
	tt.Frame()
	tt.Frame()
	if row.Key != "limit" {
		t.Fatalf("param key = %q, want limit", row.Key)
	}
}
