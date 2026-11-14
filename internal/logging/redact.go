package logging

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

var sensitiveKeyParts = []string{
	"authorization", "cookie", "set-cookie", "proxy-authorization",
	"token", "access_token", "refresh_token", "id_token",
	"api_key", "apikey", "secret", "password", "passwd", "pwd",
	"credential", "private_key", "client_secret", "session",
}

// IsSensitiveKey reports whether a key name likely contains sensitive data.
func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

// RedactString replaces known secret-bearing patterns in free-form strings.
func RedactString(s string) string {
	if s == "" {
		return s
	}
	out := bearerRE.ReplaceAllString(s, "Bearer "+redacted)
	out = kvSecretRE.ReplaceAllStringFunc(out, func(m string) string {
		parts := kvSecretRE.FindStringSubmatch(m)
		if len(parts) != 3 {
			return m
		}
		return parts[1] + "=" + redacted
	})
	return out
}

var (
	bearerRE   = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._\-~+/]+=*`)
	kvSecretRE = regexp.MustCompile(`(?i)\b(token|access_token|refresh_token|api[_-]?key|secret|password|passwd|client_secret)\s*=\s*([^\s,;]+)`)
)

// RedactValue redacts a slog.Value recursively.
func RedactValue(v slog.Value) slog.Value {
	switch v.Kind() {
	case slog.KindString:
		return slog.StringValue(RedactString(v.String()))
	case slog.KindAny:
		if err, ok := v.Any().(error); ok {
			return slog.StringValue(RedactString(err.Error()))
		}
		return v
	case slog.KindGroup:
		attrs := v.Group()
		out := make([]slog.Attr, 0, len(attrs))
		for _, a := range attrs {
			out = append(out, RedactAttr(a))
		}
		return slog.GroupValue(out...)
	case slog.KindLogValuer:
		return RedactValue(v.Resolve())
	default:
		return v
	}
}

// RedactAttr redacts sensitive attributes by key and value.
func RedactAttr(a slog.Attr) slog.Attr {
	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	return slog.Attr{Key: a.Key, Value: RedactValue(a.Value)}
}

type redactHandler struct{ next slog.Handler }

// Wrap returns a redacting slog.Handler.
func Wrap(next slog.Handler) slog.Handler { return &redactHandler{next: next} }

func (h *redactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *redactHandler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, RedactString(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(RedactAttr(a))
		return true
	})
	return h.next.Handle(ctx, nr)
}

func (h *redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, RedactAttr(a))
	}
	return &redactHandler{next: h.next.WithAttrs(out)}
}

func (h *redactHandler) WithGroup(name string) slog.Handler {
	return &redactHandler{next: h.next.WithGroup(name)}
}
