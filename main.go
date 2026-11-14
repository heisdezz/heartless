package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"heartless/internal/app"
	"heartless/internal/config"
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
		// Apply the saved theme before the first frame.
		appTheme := config.DefaultApplicationSettings()
		if saved, err := services.Settings.Load(context.Background()); err == nil {
			appTheme = saved
		}
		themeCfg := screens.ThemeConfig(appTheme)
		mygo.NewWindow(mygo.WindowOptions{
			Title:  "Heartless",
			Width:  1200,
			Height: 800,
			// The view draws the whole window decoration (screens.titleBar): the
			// bar, its window buttons and the drag area. Frameless rather than a
			// hidden title bar because a compositor that decorates windows itself
			// (KWin on Wayland) keeps its own title bar for one, and GTK keeps
			// the buttons out of the view's hands. GtkWindow stays resizable, so
			// the window still resizes from its edges and drags by its bar.
			// A platform that draws the controls anyway (macOS) still reports
			// their room, which the bar keeps clear of.
			Frameless:      true,
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: int(screens.TitleBarHeight),
			Content: ui.View(func(c *ui.Context) {
				themeCfg.Apply(c)
				screens.App(c, services)
			}),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
