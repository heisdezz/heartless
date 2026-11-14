package screens

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"heartless/internal/app"
	domainreq "heartless/internal/domain/request"
	domainres "heartless/internal/domain/response"
	domainvar "heartless/internal/domain/variable"
)

// mu guards requestStates and the fields of its values, which background
// execution updates from another goroutine while frames read them.
var mu sync.Mutex

// requestStateFor returns the editor state for a request, seeding it from the
// saved request the first time it is opened.
func (s *shellState) requestStateFor(req domainreq.Request) *requestState {
	if s.requestStates == nil {
		s.requestStates = make(map[string]*requestState)
	}
	rs, ok := s.requestStates[req.ID]
	if !ok {
		rs = &requestState{
			Method:  req.Method,
			URL:     req.URL,
			Body:    req.Body,
			Params:  paramsToRows(req.Params),
			Headers: paramsToRows(req.Headers),
		}
		if rs.Method == "" {
			rs.Method = "GET"
		}
		s.requestStates[req.ID] = rs
	}
	return rs
}

// paramsToRows turns a map into editable rows.
func paramsToRows(m map[string]string) []*kv {
	var out []*kv
	for k, v := range m {
		out = append(out, &kv{Key: k, Value: v, On: true})
	}
	// One blank row to type into.
	out = append(out, &kv{On: true})
	return out
}

// rowsToParams flattens the enabled, named rows into a map.
func rowsToParams(rows []*kv) map[string]string {
	m := map[string]string{}
	for _, r := range rows {
		if r == nil || !r.On || strings.TrimSpace(r.Key) == "" {
			continue
		}
		m[r.Key] = r.Value
	}
	return m
}

// buildDomainRequest turns the request's editor state into a domain request for
// execution. The URL keeps its {{var}} placeholders; the executor resolves them.
func (rs *requestState) buildDomainRequest(req domainreq.Request) *domainreq.Request {
	out := req
	out.Method = rs.Method
	out.URL = strings.TrimSpace(rs.URL)
	out.Body = rs.Body
	out.Params = rowsToParams(rs.Params)
	out.Headers = rowsToParams(rs.Headers)
	domainreq.NormalizeHTTP(&out)
	return &out
}

// send runs the request in the background. The pane only shows this request's
// own result, so a send never affects another request's state.
func (s *shellState) send(services *app.Services, req domainreq.Request, invalidate func()) {
	if services == nil {
		return
	}
	rs := s.requestStateFor(req)

	mu.Lock()
	if rs.running {
		mu.Unlock()
		return
	}
	rs.running = true
	rs.error = ""
	ctx, cancel := context.WithCancel(context.Background())
	rs.cancel = cancel
	domain := rs.buildDomainRequest(req)
	// Resolve {{var}} against this request's active environment and workspace.
	rs.sentBody = domain.Body
	exec := services.ExecutorFor(s.resolverVariables())
	mu.Unlock()

	go func() {
		defer cancel()
		result, err := exec.Execute(ctx, domain)

		mu.Lock()
		rs.running = false
		rs.cancel = nil
		if err != nil {
			rs.error = err.Error()
		} else {
			rs.response = result
			if result != nil && result.ErrorText != "" {
				rs.error = result.ErrorText
			}
		}
		mu.Unlock()

		if invalidate != nil {
			invalidate()
		}
	}()
}

// resolverVariables returns the variables the active environment and workspace
// contribute, environment first.
func (s *shellState) resolverVariables() (env, workspace []*domainvar.Variable) {
	for i := range s.variables {
		v := &s.variables[i]
		if v.EnvironmentID != "" && v.EnvironmentID == s.environmentID {
			env = append(env, v)
		} else if v.EnvironmentID == "" {
			workspace = append(workspace, v)
		}
	}
	return env, workspace
}

// cancelSend stops this request's in-flight execution, if any.
func (s *shellState) cancelSend(reqID string) {
	mu.Lock()
	defer mu.Unlock()
	if rs, ok := s.requestStates[reqID]; ok && rs.cancel != nil {
		rs.cancel()
	}
}

// snapshot returns a copy of the request's state for a frame to render.
func (s *shellState) snapshot(reqID string) (requestState, bool) {
	mu.Lock()
	defer mu.Unlock()
	rs, ok := s.requestStates[reqID]
	if !ok {
		return requestState{}, false
	}
	return *rs, true
}

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

// previewURL resolves {{var}} placeholders using the active environment and
// workspace variables, for display under the URL field.
func (s *shellState) previewURL(raw string) string {
	repl := func(m string) string {
		name := strings.TrimSpace(m[2 : len(m)-2])
		for _, v := range s.activeVariables() {
			if v.Name == name {
				return v.Value
			}
		}
		return m
	}
	return placeholderRe.ReplaceAllStringFunc(raw, repl)
}

// activeVariables returns environment variables scoped to the active
// environment first, then workspace variables.
func (s *shellState) activeVariables() []variableView {
	var out []variableView
	for _, v := range s.variables {
		if v.EnvironmentID == s.environmentID && s.environmentID != "" {
			out = append(out, variableView{Name: v.Name, Value: v.Value})
		}
	}
	for _, v := range s.variables {
		if v.EnvironmentID == "" {
			out = append(out, variableView{Name: v.Name, Value: v.Value})
		}
	}
	return out
}

// variableView is a name/value pair for placeholder previews.
type variableView struct {
	Name  string
	Value string
}

// resultSummary describes a result for the response pane.
func resultSummary(r *domainres.ExecutionResult) (status string, ok bool) {
	if r == nil || r.Response == nil {
		return "", false
	}
	return statusLine(r.Response), true
}

func statusLine(res *domainres.Response) string {
	if res.StatusText != "" {
		return strconv.Itoa(res.StatusCode) + " " + res.StatusText
	}
	return strconv.Itoa(res.StatusCode)
}

// humanDuration formats a duration for the response pane, as "1.2s" or
// "240ms".
func humanDuration(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	if d < time.Second {
		return strconv.Itoa(int(d/time.Millisecond)) + "ms"
	}
	return strconv.FormatFloat(d.Seconds(), 'f', 2, 64) + "s"
}

// humanBytes formats a byte count, as "1.2 KB".
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return strconv.FormatFloat(float64(n)/float64(div), 'f', 1, 64) + " " + "KMGTPE"[exp:exp+1] + "B"
}

// prettyBody indents JSON bodies and leaves anything else as it is.
func prettyBody(body string) string {
	if formatted, ok := formatJSONBody(body); ok {
		return formatted
	}
	return body
}

// formatJSONBody returns an indented JSON body when body is valid JSON.
func formatJSONBody(body string) (string, bool) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return body, false
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return body, false
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(body), "", "  "); err != nil {
		return body, false
	}
	return buf.String(), true
}

// isJSONContentType reports whether a content type indicates JSON.
func isJSONContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return false
	}
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct == "application/json" || strings.HasSuffix(ct, "+json")
}

// looksLikeJSON reports whether body appears to be JSON.
func looksLikeJSON(body string) bool {
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}
