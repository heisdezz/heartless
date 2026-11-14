package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactStringAndKey(t *testing.T) {
	if !IsSensitiveKey("Authorization") || !IsSensitiveKey("api_key") || IsSensitiveKey("method") {
		t.Fatal("sensitive key detection mismatch")
	}
	s := RedactString("Bearer abc.def.ghi token=secret123 password=hunter2")
	if strings.Contains(s, "abc.def.ghi") || strings.Contains(s, "secret123") || strings.Contains(s, "hunter2") {
		t.Fatalf("expected redaction, got %q", s)
	}
}

func TestWrapRedactsRecord(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(Wrap(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	l.Info("request", "authorization", "Bearer supersecret", "url", "https://x.test?token=abc", "ok", true)
	out := buf.String()
	if strings.Contains(out, "supersecret") || strings.Contains(out, "token=abc") {
		t.Fatalf("expected redacted output, got %s", out)
	}
	if !strings.Contains(out, "REDACTED") {
		t.Fatalf("expected REDACTED marker, got %s", out)
	}
}
