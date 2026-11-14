package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"heartless/internal/app"
	"heartless/internal/screens"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(configDir, "heartless", "heartless.db")
	app.SetDefaultDBPath(dbPath)

	services, err := app.Open(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = services.DB.Close()
	}()

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:  "Heartless",
			Width:  1200,
			Height: 800,
			Content: ui.View(func(c *ui.Context) {
				theme.Apply(c)
				screens.App(c, services)
			}),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
