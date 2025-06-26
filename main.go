package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Hello GUI")

	greeting := widget.NewLabel("Hello Fyne!")
	button := widget.NewButton("Click me!", func() {
		greeting.SetText("Button clicked!")
	})

	content := container.NewVBox(
		greeting,
		button,
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(300, 200))
	myWindow.ShowAndRun()
}
