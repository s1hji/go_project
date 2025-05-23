package main

import (
	"log"
	"todolist/db"
	"todolist/gui"
	"todolist/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {

	err := db.Init()
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer db.Close()

	a := app.New()
	a.Settings().SetTheme(&theme.CustomTheme{})

	w := a.NewWindow("My Tasks")
	w.Resize(fyne.NewSize(400, 600))

	gui.ShowTodoLists(w)

	w.ShowAndRun()
}
