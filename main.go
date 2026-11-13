package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var count int

func view(c *ui.Context) {

	theme.Apply(c)
	bar := c.TitleBar()
	ui.Row(c).Height(max(bar.Height, 32)).Padding(0, bar.Right+12, 0, bar.Left+12).DragWindow().Children(func() {
		ui.Text(c, "Inbox").Bold()
	})
	ui.Column(c).Fill().Center().Gap(12).Background(theme.TokensOf(c).BgContainer).Children(func() {
		if action.Button(c, "Hello Lobe", action.ButtonProps{Variant: action.Primary}).Clicked() {
			count++
		}
		ui.Text(c, fmt.Sprintf("Clicked %d times", count))
	})
}

func main() {

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Hello Lobe", Width: 480, Height: 320, Content: ui.View(view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
