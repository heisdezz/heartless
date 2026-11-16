package request

import (
	"context"
	"encoding/base64"
	"maps"
	"strings"
	"sync"
	"time"

	domainhist "heartless/internal/domain/history"
	domain "heartless/internal/domain/request"
	"heartless/internal/domain/response"
	varres "heartless/internal/domain/variable-resolution"
	transport "heartless/internal/transport/http"
)

// HistoryRecorder persists execution history entries.
type HistoryRecorder interface {
	Record(ctx context.Context, entry *domainhist.Entry) error
}

// Executor executes HTTP requests with validation, variable resolution, and transport.
type Executor struct {
	transport *transport.Client
	resolver  *varres.Resolver
	settings  domain.HTTPSettings
	history   HistoryRecorder

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// NewExecutor creates a request executor.
func NewExecutor(client *transport.Client, resolver *varres.Resolver, settings domain.HTTPSettings) *Executor {
	if client == nil {
		client = transport.NewClient()
	}
	return &Executor{
		transport: client,
		resolver:  resolver,
		settings:  settings,
		cancels:   map[string]context.CancelFunc{},
	}
}

// WithHistory attaches a history recorder.
func (e *Executor) WithHistory(recorder HistoryRecorder) *Executor {
	e.history = recorder
	return e
}

func (e *Executor) Execute(ctx context.Context, req *domain.Request) (*response.ExecutionResult, error) {
	start := time.Now()
	if req == nil {
		res := &response.ExecutionResult{Error: errString("request is required"), ErrorText: "request is required", StartedAt: start, EndedAt: start}
		return res, nil
	}
	resolved := cloneRequest(req)
	domain.NormalizeHTTP(resolved)

	if e.resolver != nil {
		resolved.URL = e.resolver.Resolve(ctx, resolved.URL)
		resolved.Body = e.resolver.Resolve(ctx, resolved.Body)
		for k, v := range resolved.Headers {
			resolved.Headers[k] = e.resolver.Resolve(ctx, v)
		}
		for k, v := range resolved.Params {
			resolved.Params[k] = e.resolver.Resolve(ctx, v)
		}
		resolved.Auth = resolveAuth(ctx, resolved.Auth, e.resolver)
	}

	applyAuth(resolved)

	settings := e.settings
	if settings.Timeout == 0 {
		settings = domain.DefaultHTTPSettings()
	}

	if errs := domain.ValidateHTTP(resolved, settings); errs.Has() {
		err := errs
		end := time.Now()
		res := &response.ExecutionResult{RequestID: resolved.ID, Error: err, ErrorText: err.Error(), StartedAt: start, EndedAt: end, Duration: end.Sub(start)}
		e.record(ctx, resolved, res)
		return res, nil
	}

	execCtx, cancel := context.WithCancel(ctx)
	execID := resolved.ID
	e.mu.Lock()
	e.cancels[execID] = cancel
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.cancels, execID)
		e.mu.Unlock()
		cancel()
	}()

	res, err := e.transport.Execute(execCtx, resolved, settings)
	end := time.Now()
	result := &response.ExecutionResult{
		RequestID: resolved.ID,
		Response:  res,
		Duration:  end.Sub(start),
		StartedAt: start,
		EndedAt:   end,
	}
	if err != nil {
		result.Error = err
		result.ErrorText = err.Error()
		if res != nil {
			res.Error = err.Error()
		}
	}
	if res != nil {
		if res.RequestID == "" {
			res.RequestID = resolved.ID
		}
		if res.Duration == 0 {
			res.Duration = end.Sub(start)
		}
	}
	if result.Duration == 0 {
		result.Duration = end.Sub(start)
	}
	e.record(ctx, resolved, result)
	return result, nil
}

func (e *Executor) record(ctx context.Context, req *domain.Request, result *response.ExecutionResult) {
	if e == nil || e.history == nil || req == nil || result == nil {
		return
	}
	entry := domainhist.New(req.WorkspaceID)
	entry.RequestID = req.ID
	entry.Method = req.Method
	entry.URL = req.URL
	if result.Response != nil {
		entry.StatusCode = result.Response.StatusCode
		// Keep the body, headers and metadata so the result is reproducible
		// from the database.
		entry.Response = result.Response
	}
	entry.DurationMS = result.Duration.Milliseconds()
	if result.Error != nil {
		entry.Error = result.Error.Error()
	} else if result.ErrorText != "" {
		entry.Error = result.ErrorText
	}
	_ = e.history.Record(ctx, entry)
}

func (e *Executor) Cancel(_ context.Context, executionID string) error {
	e.mu.Lock()
	cancel, ok := e.cancels[executionID]
	e.mu.Unlock()
	if ok {
		cancel()
	}
	return nil
}

func cloneRequest(r *domain.Request) *domain.Request {
	if r == nil {
		return nil
	}
	c := *r
	c.Headers = maps.Clone(r.Headers)
	if c.Headers == nil {
		c.Headers = map[string]string{}
	}
	c.Params = maps.Clone(r.Params)
	if c.Params == nil {
		c.Params = map[string]string{}
	}
	return &c
}

type simpleErr string

// resolveAuth resolves {{var}} placeholders in an auth's credentials so a token
// can come from the active environment.
func resolveAuth(ctx context.Context, a domain.Auth, r *varres.Resolver) domain.Auth {
	a.Token = r.Resolve(ctx, a.Token)
	a.Username = r.Resolve(ctx, a.Username)
	a.Password = r.Resolve(ctx, a.Password)
	a.Key = r.Resolve(ctx, a.Key)
	return a
}

// applyAuth adds the request's credentials to its headers. Only auth types the
// transport supports set a header; the rest leave the request unchanged.
func applyAuth(r *domain.Request) {
	if r == nil {
		return
	}
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	switch r.Auth.Type {
	case domain.AuthBearer:
		if strings.TrimSpace(r.Auth.Token) != "" {
			r.Headers["Authorization"] = "Bearer " + strings.TrimSpace(r.Auth.Token)
		}
	case domain.AuthBasic:
		if r.Auth.Username != "" || r.Auth.Password != "" {
			token := base64.StdEncoding.EncodeToString([]byte(r.Auth.Username + ":" + r.Auth.Password))
			r.Headers["Authorization"] = "Basic " + token
		}
	case domain.AuthAPIKey:
		header := r.Auth.Header
		if header == "" {
			header = "X-API-Key"
		}
		if r.Auth.Key != "" {
			r.Headers[header] = r.Auth.Prefix + r.Auth.Key
		}
	}
}

func (e simpleErr) Error() string { return string(e) }

func errString(s string) error { return simpleErr(s) }
