package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"heartless/internal/app"
	"heartless/internal/screens"
)

// The view runs without a window in tests, which click and type as a user
// would.
func TestAppRenders(t *testing.T) {
	app.SetDefaultDBPath(filepath.Join(t.TempDir(), "heartless.db"))
	services, err := app.Open(context.Background())
	if err != nil {
		t.Fatalf("open services: %v", err)
	}
	defer services.DB.Close()

	tt := ui.NewTester(func(c *ui.Context) { screens.App(c, services) }, 1000, 700)
	if !tt.HasText("Heartless") {
		t.Fatalf("title bar missing, texts %q", tt.Texts())
	}
	for _, want := range []string{"Environments", "Settings", "Request", "Response"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q, texts %q", want, tt.Texts())
		}
	}
	if err := tt.Click("Settings"); err != nil {
		t.Fatalf("open settings: %v", err)
	}
}
