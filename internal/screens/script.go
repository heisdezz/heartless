package screens

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"

	domainreq "heartless/internal/domain/request"
	domainres "heartless/internal/domain/response"
	domainvar "heartless/internal/domain/variable"
)

// scriptTimeout bounds a post-response script. A runaway loop is stopped and
// reported rather than freezing the app.
const scriptTimeout = 2 * time.Second

// scriptMaxLog bounds the captured console output kept from one run.
const scriptMaxLog = 200

// ScriptRequest is the `req` object a post-response script sees: the request
// as it was sent, with its placeholders already resolved.
type ScriptRequest struct {
	ID        string            `json:"id"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Body      string            `json:"body"`
	BodyType  string            `json:"bodyType"`
	Params    map[string]string `json:"params"`
	Headers   map[string]string `json:"headers"`
	AuthType  string            `json:"authType"`
	Workspace string            `json:"workspaceId"`
}

// ScriptEnvironment carries the variables the script may read and update.
type ScriptEnvironment struct {
	// EnvironmentID is the active environment; variables the script adds belong
	// to it (or to the workspace when it is empty).
	EnvironmentID string
	Environment   []domainvar.Variable
	Workspace     []domainvar.Variable
}

// ScriptOutput is one console line from a run.
type ScriptOutput struct {
	Level string // log, info, debug, warn, error, pass, fail
	Text  string
}

// ScriptVariableUpdate records a variable the script added or changed.
type ScriptVariableUpdate struct {
	Name     string
	Value    string
	Previous string
	Existed  bool
	// Workspace marks a name that belongs to the workspace scope rather than
	// the active environment.
	Workspace bool
}

// ScriptResult is what a post-response script run produced.
type ScriptResult struct {
	Output    []ScriptOutput
	Updates   []ScriptVariableUpdate
	Err       string
	Duration  time.Duration
	Assertion int
	Passed    int
}

// RunPostResponseScript executes src in an isolated JavaScript runtime with
// `req`, `res`, `env.get`/`env.set`/`env.all`, `test` and `console` available.
// It has no network, filesystem or process access. Updated variables are
// returned in the result rather than persisted here, so the caller can show
// them before saving.
func RunPostResponseScript(src string, req *ScriptRequest, res *domainres.Response, environment ScriptEnvironment) *ScriptResult {
	result := &ScriptResult{}
	src = strings.TrimSpace(src)
	if src == "" {
		return result
	}
	start := time.Now()

	lookup := newScriptVarLookup(environment)
	vm := goja.New()

	// Interrupt from another goroutine so an endless loop ends.
	timer := time.AfterFunc(scriptTimeout, func() {
		vm.Interrupt("script timed out")
	})
	defer timer.Stop()

	logLine := func(level string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			if len(result.Output) < scriptMaxLog {
				parts := make([]string, 0, len(call.Arguments))
				for _, a := range call.Arguments {
					parts = append(parts, scriptFormatValue(a))
				}
				result.Output = append(result.Output, ScriptOutput{Level: level, Text: strings.Join(parts, " ")})
			}
			return goja.Undefined()
		}
	}

	console := vm.NewObject()
	for _, level := range []string{"log", "info", "debug", "warn", "error"} {
		_ = console.Set(level, logLine(level))
	}
	_ = vm.Set("console", console)
	_ = vm.Set("req", scriptRequestValue(vm, req))
	_ = vm.Set("res", scriptResponseValue(vm, res))
	_ = vm.Set("env", scriptEnvValue(vm, lookup))
	_ = vm.Set("test", scriptTestFunc(vm, result))

	_, err := vm.RunScript("post-response.js", src)
	if err != nil {
		if strings.Contains(err.Error(), "script timed out") || err.Error() == "interrupted" {
			result.Err = fmt.Sprintf("script timed out after %s", scriptTimeout)
		} else {
			result.Err = err.Error()
		}
	}
	result.Duration = time.Since(start)
	result.Updates = lookup.updates()
	return result
}

// scriptEnvValue builds `env`, the variable accessor. Reads see the script's
// own writes, and writes are collected as updates.
func scriptEnvValue(vm *goja.Runtime, lookup *scriptVarLookup) goja.Value {
	obj := vm.NewObject()
	_ = obj.Set("get", func(name string) goja.Value {
		if v, ok := lookup.get(name); ok {
			return vm.ToValue(v)
		}
		return goja.Undefined()
	})
	_ = obj.Set("has", func(name string) bool {
		_, ok := lookup.get(name)
		return ok
	})
	_ = obj.Set("set", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			return goja.Undefined()
		}
		name := strings.TrimSpace(call.Argument(0).String())
		if name == "" {
			return goja.Undefined()
		}
		lookup.set(name, call.Argument(1).String())
		return goja.Undefined()
	})
	_ = obj.Set("all", func() goja.Value {
		out := vm.NewObject()
		for name, value := range lookup.snapshot() {
			_ = out.Set(name, value)
		}
		return out
	})
	return obj
}

// scriptTestFunc builds `test(name, fn)`: a named assertion whose thrown error
// means it failed. Results land in the run's output so the pane lists them.
func scriptTestFunc(vm *goja.Runtime, result *ScriptResult) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Undefined()
		}
		name := call.Argument(0).String()
		passed, detail := true, ""
		if fn, ok := goja.AssertFunction(call.Argument(1)); ok && len(call.Arguments) > 1 {
			if _, err := fn(goja.Undefined()); err != nil {
				passed = false
				detail = " — " + err.Error()
			}
		}
		result.Assertion++
		level := "fail"
		if passed {
			result.Passed++
			level = "pass"
		}
		result.Output = append(result.Output, ScriptOutput{Level: level, Text: name + detail})
		return goja.Undefined()
	}
}

// scriptVarLookup overlays script-set values on the variables it started with.
type scriptVarLookup struct {
	original  map[string]string
	values    map[string]string
	workspace map[string]bool // names that came from the workspace
}

func newScriptVarLookup(environment ScriptEnvironment) *scriptVarLookup {
	l := &scriptVarLookup{
		original:  map[string]string{},
		values:    map[string]string{},
		workspace: map[string]bool{},
	}
	for _, v := range environment.Workspace {
		l.original[v.Name] = v.Value
		l.values[v.Name] = v.Value
		l.workspace[v.Name] = true
	}
	// An environment variable wins over a workspace one of the same name.
	for _, v := range environment.Environment {
		l.original[v.Name] = v.Value
		l.values[v.Name] = v.Value
		delete(l.workspace, v.Name)
	}
	return l
}

func (l *scriptVarLookup) get(name string) (string, bool) {
	v, ok := l.values[name]
	return v, ok
}

func (l *scriptVarLookup) set(name, value string) {
	l.values[name] = value
}

func (l *scriptVarLookup) snapshot() map[string]string {
	out := make(map[string]string, len(l.values))
	for k, v := range l.values {
		out[k] = v
	}
	return out
}

// updates lists the variables the script changed, sorted by name so the pane
// is stable between runs.
func (l *scriptVarLookup) updates() []ScriptVariableUpdate {
	var out []ScriptVariableUpdate
	for name, value := range l.values {
		prev, existed := l.original[name]
		if existed && prev == value {
			continue
		}
		out = append(out, ScriptVariableUpdate{
			Name:      name,
			Value:     value,
			Previous:  prev,
			Existed:   existed,
			Workspace: l.workspace[name],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// scriptRequestValue builds `req` from the request.
func scriptRequestValue(vm *goja.Runtime, req *ScriptRequest) goja.Value {
	obj := vm.NewObject()
	if req == nil {
		return obj
	}
	_ = obj.Set("id", req.ID)
	_ = obj.Set("method", req.Method)
	_ = obj.Set("url", req.URL)
	_ = obj.Set("body", req.Body)
	_ = obj.Set("bodyType", req.BodyType)
	_ = obj.Set("params", req.Params)
	_ = obj.Set("headers", req.Headers)
	_ = obj.Set("authType", req.AuthType)
	_ = obj.Set("workspaceId", req.Workspace)
	return obj
}

// scriptResponseValue builds `res` from the response. The body is the raw text;
// `res.json()` parses it into an object.
func scriptResponseValue(vm *goja.Runtime, res *domainres.Response) goja.Value {
	obj := vm.NewObject()
	if res == nil {
		_ = obj.Set("status", 0)
		_ = obj.Set("body", "")
		return obj
	}
	_ = obj.Set("id", res.ID)
	_ = obj.Set("status", res.StatusCode)
	_ = obj.Set("statusText", res.StatusText)
	_ = obj.Set("headers", res.Headers)
	_ = obj.Set("contentType", res.ContentType)
	_ = obj.Set("size", res.Size)
	_ = obj.Set("url", res.EffectiveURL)
	_ = obj.Set("body", string(res.Body))
	if res.Error != "" {
		_ = obj.Set("error", res.Error)
	}
	_ = obj.Set("json", func() goja.Value {
		var out any
		if err := json.Unmarshal(res.Body, &out); err != nil {
			panic(vm.NewTypeError("response body is not JSON: " + err.Error()))
		}
		return vm.ToValue(out)
	})
	return obj
}

// scriptFormatValue renders a logged argument as console output, encoding
// objects as JSON so they read like data rather than "[object Object]".
func scriptFormatValue(v goja.Value) string {
	if v == nil {
		return "undefined"
	}
	switch v.Export().(type) {
	case map[string]any, []any:
		if b, err := json.Marshal(v.Export()); err == nil {
			return string(b)
		}
	}
	return v.String()
}

// scriptRequestFromDomain builds the `req` view of a request for a script.
func scriptRequestFromDomain(req domainreq.Request) *ScriptRequest {
	return &ScriptRequest{
		ID:        req.ID,
		Method:    req.Method,
		URL:       req.URL,
		Body:      req.Body,
		BodyType:  string(req.BodyType),
		Params:    req.Params,
		Headers:   req.Headers,
		AuthType:  string(req.Auth.Type),
		Workspace: req.WorkspaceID,
	}
}
