package screens

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	domainreq "heartless/internal/domain/request"
	domainres "heartless/internal/domain/response"
	domainvar "heartless/internal/domain/variable"
)

// TestPostResponseScriptRun checks the script sees req and res, can log, and
// collects assertion results.
func TestPostResponseScriptRun(t *testing.T) {
	req := &ScriptRequest{ID: "r1", Method: "GET", URL: "https://api.test/users"}
	res := &domainres.Response{
		StatusCode: 200,
		StatusText: "200 OK",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"users":[{"id":7}]}`),
	}
	src := `
console.log(req.method, req.url);
const body = res.json();
console.log("users", body.users.length);
test("has one user", () => { if (body.users.length !== 1) throw new Error("count") });
test("status is 200", () => { if (res.status !== 200) throw new Error("status " + res.status) });
`
	out := RunPostResponseScript(src, req, res, ScriptEnvironment{})
	if out.Err != "" {
		t.Fatalf("script error: %s", out.Err)
	}
	if out.Assertion != 2 || out.Passed != 2 {
		t.Fatalf("assertions = %d/%d, want 2/2", out.Passed, out.Assertion)
	}
	var logged []string
	for _, line := range out.Output {
		if line.Level == "log" {
			logged = append(logged, line.Text)
		}
	}
	if len(logged) != 2 || logged[0] != "GET https://api.test/users" || logged[1] != "users 1" {
		t.Fatalf("log lines = %q", logged)
	}
}

// TestPostResponseBodyIsParsed checks a JSON body can be read with both dot
// and bracket notation, while res.text stays the raw body.
func TestPostResponseBodyIsParsed(t *testing.T) {
	res := &domainres.Response{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"code":0,"anyvalue":"yes","items":[{"id":7}]}`),
	}
	src := `
if (res.body.code !== 0) throw new Error("dot access: " + res.body.code);
if (res.body["anyvalue"] !== "yes") throw new Error("bracket access");
if (res.body.items[0].id !== 7) throw new Error("nested array");
if (typeof res.text !== "string" || res.text.indexOf("anyvalue") < 0) throw new Error("raw text");
console.log("parsed", res.body.code, res.body.anyvalue);
`
	out := RunPostResponseScript(src, &ScriptRequest{}, res, ScriptEnvironment{})
	if out.Err != "" {
		t.Fatalf("script error: %s", out.Err)
	}
	if len(out.Output) != 1 || out.Output[0].Text != "parsed 0 yes" {
		t.Fatalf("output = %+v", out.Output)
	}
}

// TestPostResponseBodyNonJSON checks a plain-text body stays a string.
func TestPostResponseBodyNonJSON(t *testing.T) {
	res := &domainres.Response{Body: []byte("plain text")}
	src := `
if (typeof res.body !== "string") throw new Error("expected string body");
if (res.body !== "plain text") throw new Error("body = " + res.body);
`
	out := RunPostResponseScript(src, &ScriptRequest{}, res, ScriptEnvironment{})
	if out.Err != "" {
		t.Fatalf("script error: %s", out.Err)
	}
}

// TestPostResponseScriptVariableUpdates checks env.set records an update with
// the previous value, and that new names are marked as new.
func TestPostResponseScriptVariableUpdates(t *testing.T) {
	env := ScriptEnvironment{
		EnvironmentID: "env-dev",
		Environment:   []domainvar.Variable{{Name: "token", Value: "old", EnvironmentID: "env-dev"}},
		Workspace:     []domainvar.Variable{{Name: "baseUrl", Value: "https://x"}},
	}
	src := `
env.set("token", "new");
env.set("fresh", String(42));
if (env.get("baseUrl") !== "https://x") throw new Error("workspace variable not visible");
if (env.get("token") !== "new") throw new Error("own write not visible");
`
	out := RunPostResponseScript(src, &ScriptRequest{}, nil, env)
	if out.Err != "" {
		t.Fatalf("script error: %s", out.Err)
	}
	if len(out.Updates) != 2 {
		t.Fatalf("updates = %+v", out.Updates)
	}
	if u := out.Updates[0]; u.Name != "fresh" || u.Value != "42" || u.Existed {
		t.Fatalf("fresh update = %+v", u)
	}
	if u := out.Updates[1]; u.Name != "token" || u.Value != "new" || u.Previous != "old" || !u.Existed {
		t.Fatalf("token update = %+v", u)
	}
}

// TestPostResponseScriptFailingAssertion checks a failed test is reported.
func TestPostResponseScriptFailingAssertion(t *testing.T) {
	src := `test("status is 201", () => { if (res.status !== 201) throw new Error("got " + res.status) });`
	out := RunPostResponseScript(src, &ScriptRequest{}, &domainres.Response{StatusCode: 200}, ScriptEnvironment{})
	if out.Assertion != 1 || out.Passed != 0 {
		t.Fatalf("assertions = %d/%d", out.Passed, out.Assertion)
	}
	last := out.Output[len(out.Output)-1]
	if last.Level != "fail" || !strings.Contains(last.Text, "got 200") {
		t.Fatalf("output = %+v", last)
	}
}

// TestPostResponseScriptError checks a thrown error is captured, not fatal.
func TestPostResponseScriptError(t *testing.T) {
	out := RunPostResponseScript(`throw new Error("boom")`, &ScriptRequest{}, nil, ScriptEnvironment{})
	if !strings.Contains(out.Err, "boom") {
		t.Fatalf("err = %q", out.Err)
	}
}

// TestScriptStyleRanges checks keywords, strings, variables and comments are
// colored, with rune offsets covering the text.
func TestScriptStyleRanges(t *testing.T) {
	tk := theme.Default().Tokens(false)
	src := `const url = "{{baseUrl}}/users"; // note` + "\n" + `env.set("token", res.json().token);`
	ranges := scriptStyleRanges(src, tk, func(name string) bool { return name == "baseUrl" })
	if len(ranges) == 0 {
		t.Fatal("no ranges")
	}

	colorAt := func(offset int) (styleRange, bool) {
		for _, r := range ranges {
			if offset >= r.Start && offset < r.End {
				return r, true
			}
		}
		return styleRange{}, false
	}
	varAt := utf8.RuneCountInString(src[:strings.Index(src, "{{baseUrl}}")])
	var vr *styleRange
	for i := range ranges {
		if ranges[i].Start == varAt {
			vr = &ranges[i]
		}
	}
	if vr == nil || !vr.Bold || vr.End-varAt != utf8.RuneCountInString("{{baseUrl}}") {
		t.Fatalf("variable range = %+v, want a bold {{baseUrl}} at rune %d", vr, varAt)
	}
	// The variable must be colored over the string it sits in, not lost in it.
	strAt := utf8.RuneCountInString(src[:strings.Index(src, `"`)])
	sr, _ := colorAt(strAt)
	if vr.Color == sr.Color {
		t.Fatalf("variable color %v same as its string's", vr.Color)
	}
	kwAt := strings.Index(src, "const")
	if r, ok := colorAt(kwAt); !ok || !r.Bold {
		t.Fatalf("keyword range = %+v, want bold", r)
	}
	commentAt := strings.Index(src, "// note")
	if r, ok := colorAt(utf8.RuneCountInString(src[:commentAt])); !ok || r.Color != tk.TextTertiary {
		t.Fatalf("comment color = %v, want tertiary", r.Color)
	}
	// Every range stays inside the text, in runes.
	total := utf8.RuneCountInString(src)
	for _, r := range ranges {
		if r.Start < 0 || r.End > total || r.Start >= r.End {
			t.Fatalf("range %+v outside 0..%d", r, total)
		}
	}
}

// TestScriptHighlightSpansKeepsText checks the standalone rendering keeps
// every character of the script.
func TestScriptHighlightSpansKeepsText(t *testing.T) {
	tk := theme.Default().Tokens(false)
	src := "const a = 1; // x\nenv.set('k', 'v');"
	var joined string
	for _, s := range scriptHighlightSpans(src, tk, nil) {
		joined += s.Text
	}
	if joined != src {
		t.Fatalf("spans text = %q, want %q", joined, src)
	}
}

// TestScriptCompletion checks word detection, filtering and insertion used by
// the editor's inline autocomplete.
func TestScriptCompletion(t *testing.T) {
	if prefix, start := scriptWordPrefix("console.log(res.bo)", 18); prefix != "res.bo" || start != 12 {
		t.Fatalf("word = %q at %d, want res.bo at 12", prefix, start)
	}
	env := ScriptEnvironment{
		Environment: []domainvar.Variable{{Name: "token"}},
		Workspace:   []domainvar.Variable{{Name: "baseUrl"}},
	}
	got := scriptCompletions("res.bo", env)
	if len(got) == 0 || got[0] != "res.body" {
		t.Fatalf("completions for res.bo = %q", got)
	}
	got = scriptCompletions("to", env)
	if len(got) == 0 || got[0] != "token" {
		t.Fatalf("completions for to = %q", got)
	}
	if len(scriptCompletions("res.body", env)) != 0 {
		t.Fatal("a completed token should not suggest itself")
	}

	text, caret := scriptInsertCompletion("console.log(res.bo)", 18, "res.body")
	if text != "console.log(res.body)" || caret != 20 {
		t.Fatalf("insert = %q at %d", text, caret)
	}
}

// TestScriptCompletionSuggestionsShowsAndInserts checks the editor offers
// matching tokens while a word is typed and clicking one completes it.
func TestScriptCompletionSuggestionsShowsAndInserts(t *testing.T) {
	services := newTestServices(t)
	req := domainreq.NewHTTP("ws-1", "Complete")
	req.ID = "req-complete"
	req.URL = "https://api.test/x"

	st := shellState{workspaceID: "ws-1", environmentID: "env-1"}
	rs := st.requestStateFor(*req)
	rs.Tab = 4
	rs.Script = "res.bo"

	tt := ui.NewTester(func(c *ui.Context) {
		requestPane(c, theme.TokensOf(c), &st, services, *req)
	}, 1200, 900)
	tt.Frame()
	if !tt.HasText("Complete") || !tt.HasText("res.body") {
		t.Fatalf("suggestions not shown, texts %q", tt.Texts())
	}
	if err := tt.Click("res.body"); err != nil {
		t.Fatalf("click suggestion: %v", err)
	}
	if rs.Script != "res.body" {
		t.Fatalf("script after completion = %q, want res.body", rs.Script)
	}
}

// TestScriptVariablesAutoApplied checks env.set updates are saved as soon as a
// run finishes, updating an existing variable and its editor draft.
func TestScriptVariablesAutoApplied(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	existing := domainvar.NewVariable("token", "old", "text", false)
	existing.WorkspaceID, existing.EnvironmentID = "ws-1", "env-1"
	if err := services.Variables.Save(ctx, existing.ID, "ws-1", "env-1", existing); err != nil {
		t.Fatalf("save variable: %v", err)
	}

	st := shellState{workspaceID: "ws-1", environmentID: "env-1"}
	if err := st.reloadVariables(ctx, services); err != nil {
		t.Fatalf("reload variables: %v", err)
	}
	draft := st.varDraft(*existing)
	if draft.Value != "old" {
		t.Fatalf("draft initial = %q", draft.Value)
	}

	req := domainreq.NewHTTP("ws-1", "AutoVars")
	rs := st.requestStateFor(*req)
	rs.script = &scriptRun{result: &ScriptResult{
		Updates: []ScriptVariableUpdate{{Name: "token", Value: "new", Previous: "old", Existed: true}},
	}}

	st.autoApplyScriptVariables(services)

	vars, err := services.Variables.List(ctx, "ws-1", "env-1")
	if err != nil {
		t.Fatalf("list variables: %v", err)
	}
	found := false
	for _, v := range vars {
		if v.Name == "token" {
			found = true
			if v.Value != "new" {
				t.Fatalf("stored value = %q, want new", v.Value)
			}
		}
	}
	if !found {
		t.Fatal("token variable missing")
	}
	if draft.Value != "new" {
		t.Fatalf("draft = %q, want new", draft.Value)
	}
	if !rs.script.applied {
		t.Fatal("run not marked applied")
	}
	// A second pass is a no-op.
	st.autoApplyScriptVariables(services)
	if !rs.script.applied {
		t.Fatal("run lost its applied state")
	}
}

// TestScriptRunsAutomaticallyOnResponse checks the script runs by itself once
// a send finishes, without pressing anything.
func TestScriptRunsAutomaticallyOnResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"t-123"}`))
	}))
	defer srv.Close()

	services := newTestServices(t)
	req := domainreq.NewHTTP("ws-1", "Auto")
	req.ID = "req-auto"
	req.URL = srv.URL + "/token"

	st := shellState{workspaceID: "ws-1", environmentID: "env-1"}
	rs := st.requestStateFor(*req)
	rs.Script = `env.set("token", res.json().token);`

	done := make(chan struct{})
	st.send(services, *req, func() { close(done) })
	<-done

	// The script is queued after the send; wait for its result.
	var run *scriptRun
	for i := 0; i < 400; i++ {
		mu.Lock()
		run = rs.script
		mu.Unlock()
		if run != nil && run.result != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if run == nil || run.result == nil {
		t.Fatal("script did not run after the response")
	}
	if run.result.Err != "" {
		t.Fatalf("script error: %s", run.result.Err)
	}
	if len(run.result.Updates) != 1 || run.result.Updates[0].Name != "token" || run.result.Updates[0].Value != "t-123" {
		t.Fatalf("updates = %+v", run.result.Updates)
	}
}

// TestScriptTabSavesAndRuns drives the Script tab: the text persists with the
// request and Run produces output from the last response.
func TestScriptTabSavesAndRuns(t *testing.T) {
	services := newTestServices(t)
	ctx := context.Background()

	req := domainreq.NewHTTP("ws-1", "Scripted")
	req.ID = "req-script"
	req.URL = "https://api.test/x"
	if err := services.Collections.SaveRequest(ctx, req); err != nil {
		t.Fatalf("save request: %v", err)
	}

	st := shellState{workspaceID: "ws-1", environmentID: "env-1"}
	rs := st.requestStateFor(*req)
	rs.Tab = 4
	rs.Script = `console.log("status", res.status); env.set("token", "abc");`
	rs.response = &domainres.ExecutionResult{
		Response: &domainres.Response{StatusCode: 201, StatusText: "201 Created", Body: []byte(`{}`)},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		requestPane(c, theme.TokensOf(c), &st, services, *req)
	}, 1200, 900)
	tt.Frame()
	tt.Frame()

	// The script is written back to the stored request so it survives reopening.
	saved, err := services.Collections.GetRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if saved.Script != rs.Script {
		t.Fatalf("saved script = %q, want %q", saved.Script, rs.Script)
	}

	if err := tt.Click("Run again"); err != nil {
		t.Fatalf("click Run again: %v", err)
	}
	// The run happens on the runner's goroutine; wait for the result.
	for i := 0; i < 200 && rs.script == nil; i++ {
		tt.Frame()
	}
	if rs.script == nil || rs.script.result == nil {
		t.Fatal("script did not run")
	}
	if out := rs.script.result; out.Err != "" || len(out.Updates) != 1 || out.Updates[0].Name != "token" {
		t.Fatalf("run = %+v (updates %+v)", out, out.Updates)
	}
	if !tt.HasText("status 201") {
		t.Fatalf("console output not shown, texts %q", tt.Texts())
	}

	// Saving the variable writes it to the active environment.
	st.applyScriptVariables(services, rs)
	vars, err := services.Variables.List(ctx, "ws-1", "")
	if err != nil {
		t.Fatalf("list variables: %v", err)
	}
	if len(vars) != 1 {
		t.Fatalf("variables = %d, want 1", len(vars))
	}
	if v := vars[0]; v.Name != "token" || v.Value != "abc" || v.EnvironmentID != "env-1" {
		t.Fatalf("variable = %+v, want token=abc in env-1", v)
	}
}
