package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

/*
Store in JSON

One Folder with customers
 One Folder Per Customer
   One JSON with meta
   One JSON per task
One Folder For Years
  One Folder Per Month
   One JSON per day
Entry is:
{
"id": "uuid",
"customer": "uuid",
"startTS": "timestamp"
"endTS": "maybe timestamp"
}

One folder for entry text
*/

func main() {
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
