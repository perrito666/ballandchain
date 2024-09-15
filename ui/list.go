package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func ListTimeEntries() {
	a := app.New()
	w := a.NewWindow("Gotta work")
	entry := widget.NewEntry()
	entry.MultiLine = true
	entry.Resize(fyne.NewSize(400, 400))

	rt := widget.NewRichTextFromMarkdown("")
	entry.OnChanged = func(s string) {
		rt.ParseMarkdown(s)
	}
	rt.Resize(fyne.NewSize(400, 400))

	outerBox := container.New(layout.NewGridLayout(2), entry, rt)
	w.Resize(fyne.NewSize(800, 400))
	w.SetContent(outerBox)
	w.ShowAndRun()
}
