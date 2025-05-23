package gui

import (
	"fmt"
	"time"
	"todolist/db"
	"todolist/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const dateFormat = "02.01.2006"

func ShowTodoLists(w fyne.Window) {
	userID, err := db.GetOrCreateDefaultUser()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка получения пользователя: %v", err), w)
		return
	}

	lists, err := db.GetTodoLists(userID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки списков: %v", err), w)
		return
	}

	mainContainer := container.NewVBox(
		layout.NewSpacer(),
		widget.NewLabelWithStyle("My Tasks", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
	)

	listsContainer := container.NewVBox()
	for _, list := range lists {
		currentList := list

		// Чекбокс для отметки выполнения
		check := widget.NewCheck("", func(checked bool) {
			// Здесь можно добавить логику отметки списка как выполненного
			// Например, обновить в базе данных
			fmt.Printf("Список %s отмечен как выполненный: %v\n", currentList.Title, checked)
		})

		// Кнопка списка
		listBtn := widget.NewButton(currentList.Title, func() {
			ShowTodoItems(w, currentList)
		})
		listBtn.Alignment = widget.ButtonAlignLeading

		// Кнопка удаления
		deleteBtn := widget.NewButton("✕", func() {
			dialog.ShowConfirm(
				"Удаление списка",
				"Удалить этот список и все его задачи?",
				func(ok bool) {
					if ok {
						if err := db.DeleteTodoList(currentList.ID); err != nil {
							dialog.ShowError(err, w)
							return
						}
						ShowTodoLists(w)
					}
				},
				w,
			)
		})

		// Собираем строку списка
		listRow := container.NewHBox(
			check,
			listBtn,
			layout.NewSpacer(),
			deleteBtn,
		)

		listsContainer.Add(listRow)
		listsContainer.Add(layout.NewSpacer()) // Отступ между списками
	}

	scrollContainer := container.NewVScroll(listsContainer)
	scrollContainer.SetMinSize(fyne.NewSize(w.Canvas().Size().Width*0.9, 300))

	mainContainer.Add(scrollContainer)
	mainContainer.Add(layout.NewSpacer())

	// Кнопка добавления
	addButton := widget.NewButton("+ Добавить список", func() {
		showAddListDialog(w, userID)
	})
	addButton.Importance = widget.HighImportance
	addButton.Resize(fyne.NewSize(300, 50))

	addButtonContainer := container.NewHBox(
		layout.NewSpacer(),
		addButton,
		layout.NewSpacer(),
	)

	mainContainer.Add(addButtonContainer)
	mainContainer.Add(layout.NewSpacer())

	w.SetContent(mainContainer)
}

func showAddListDialog(w fyne.Window, userID int) {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Введите название списка")

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Введите описание")
	descEntry.MultiLine = true
	descEntry.Wrapping = fyne.TextWrapWord

	// Создаем контейнер с формой и кнопками
	formContent := container.NewVBox(
		widget.NewLabelWithStyle("Новый список", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Название:"),
		titleEntry,
		widget.NewLabel("Описание:"),
		descEntry,
	)

	buttons := container.NewHBox(
		layout.NewSpacer(),
		widget.NewButton("Отмена", nil),
		widget.NewButton("Создать", nil),
		layout.NewSpacer(),
	)

	// Объединяем все в один контейнер
	content := container.NewBorder(
		nil, buttons, nil, nil,
		formContent,
	)

	// Создаем диалог с нашим контентом
	d := dialog.NewCustomWithoutButtons("",
		container.NewPadded(content),
		w,
	)

	// Назначаем действия для кнопок
	buttons.Objects[1].(*widget.Button).OnTapped = d.Hide
	buttons.Objects[2].(*widget.Button).OnTapped = func() {
		if titleEntry.Text != "" {
			newList := models.TodoList{
				UserID:      userID,
				Title:       titleEntry.Text,
				Description: descEntry.Text,
				CreatedAt:   time.Now(),
			}

			if err := db.CreateTodoList(&newList); err != nil {
				dialog.ShowError(err, w)
				return
			}
			ShowTodoLists(w)
			d.Hide()
		}
	}

	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}

func ShowTodoItems(w fyne.Window, list models.TodoList) {
	tasks, err := db.GetTasksByList(list.ID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки задач: %v", err), w)
		return
	}

	tasksContainer := container.NewVBox()
	for _, task := range tasks {
		currentTask := task
		taskRow := createTaskRow(w, &currentTask, list)
		tasksContainer.Add(taskRow)
	}

	addButton := widget.NewButton("+ Добавить задачу", func() {
		showAddTaskDialog(w, list)
	})

	backButton := widget.NewButton("← Назад", func() {
		ShowTodoLists(w)
	})

	deleteListButton := widget.NewButton("Удалить список", func() {
		dialog.ShowConfirm(
			"Удаление списка",
			"Вы уверены, что хотите удалить этот список и все его задачи?",
			func(ok bool) {
				if ok {
					if err := db.DeleteTodoList(list.ID); err != nil {
						dialog.ShowError(err, w)
						return
					}
					ShowTodoLists(w)
				}
			},
			w,
		)
	})

	controls := container.NewHBox(
		backButton,
		layout.NewSpacer(),
		deleteListButton,
	)

	w.SetContent(container.NewVBox(
		widget.NewLabelWithStyle(list.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(list.Description),
		tasksContainer,
		addButton,
		controls,
	))
}

func createTaskRow(w fyne.Window, task *models.Task, list models.TodoList) *fyne.Container {
	taskBtn := widget.NewButton("", nil)
	taskBtn.Alignment = widget.ButtonAlignLeading

	updateTaskText := func() {
		text := task.Title
		if !task.DueDate.IsZero() {
			text += " (" + task.DueDate.Format(dateFormat) + ")"
		}
		taskBtn.SetText(text)
	}
	updateTaskText()

	taskBtn.OnTapped = func() {
		showTaskDetails(w, task, updateTaskText)
	}

	check := widget.NewCheck("", func(done bool) {
		task.IsDone = done
		if err := db.UpdateTask(task); err != nil {
			dialog.ShowError(err, w)
		}
		updateTaskText()
	})
	check.SetChecked(task.IsDone)

	deleteBtn := widget.NewButton("✕", func() {
		dialog.ShowConfirm(
			"Удаление",
			"Удалить задачу?",
			func(ok bool) {
				if ok {
					if err := db.DeleteTask(task.ID); err != nil {
						dialog.ShowError(err, w)
						return
					}
					ShowTodoItems(w, list)
				}
			},
			w,
		)
	})

	return container.NewHBox(
		check,
		taskBtn,
		layout.NewSpacer(),
		deleteBtn,
	)
}

func showAddTaskDialog(w fyne.Window, list models.TodoList) {
	// Поля формы
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Название задачи")

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Описание задачи")
	descEntry.MultiLine = true
	descEntry.Wrapping = fyne.TextWrapWord

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("дд.мм.гггг")

	// Форма
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Название:", Widget: titleEntry},
			{Text: "Описание:", Widget: descEntry},
			{Text: "Срок:", Widget: dateEntry},
		},
	}

	// Создаем диалог
	d := dialog.NewForm(
		"Новая задача",
		"Добавить",
		"Отмена",
		form.Items,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// Обработка добавления задачи
			var dueDate time.Time
			if dateText := dateEntry.Text; dateText != "" {
				if parsed, err := time.Parse(dateFormat, dateText); err == nil {
					dueDate = parsed
				}
			}

			task := models.Task{
				ListID:      list.ID,
				Title:       titleEntry.Text,
				Description: descEntry.Text,
				DueDate:     dueDate,
				CreatedAt:   time.Now(),
			}

			if err := db.CreateTask(&task); err != nil {
				dialog.ShowError(err, w)
				return
			}
			ShowTodoItems(w, list)
		},
		w,
	)

	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}
func showTaskDetails(w fyne.Window, task *models.Task, onUpdate func()) {
	titleLabel := widget.NewLabelWithStyle(task.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	descLabel := widget.NewLabel(task.Description)
	descLabel.Wrapping = fyne.TextWrapWord

	dateText := "Срок не установлен"
	if !task.DueDate.IsZero() {
		dateText = "Срок: " + task.DueDate.Format(dateFormat)
	}
	dateLabel := widget.NewLabel(dateText)

	editBtn := widget.NewButton("Редактировать", func() {
		editTaskDialog(w, task, func() {
			onUpdate()
			showTaskDetails(w, task, onUpdate)
		})
	})

	dialog.ShowCustom(
		"Детали задачи",
		"Закрыть",
		container.NewVBox(
			titleLabel,
			widget.NewSeparator(),
			descLabel,
			dateLabel,
			layout.NewSpacer(),
			editBtn,
		),
		w,
	)
}

func editTaskDialog(w fyne.Window, task *models.Task, onSave func()) {
	titleEntry := widget.NewEntry()
	titleEntry.SetText(task.Title)

	descEntry := widget.NewEntry()
	descEntry.SetText(task.Description)
	descEntry.MultiLine = true

	dateEntry := widget.NewEntry()
	if !task.DueDate.IsZero() {
		dateEntry.SetText(task.DueDate.Format(dateFormat))
	}

	dialog.ShowForm(
		"Редактировать",
		"Сохранить",
		"Отмена",
		[]*widget.FormItem{
			{Text: "Название:", Widget: titleEntry},
			{Text: "Описание:", Widget: descEntry},
			{Text: "Срок:", Widget: dateEntry},
		},
		func(b bool) {
			if !b {
				return
			}

			task.Title = titleEntry.Text
			task.Description = descEntry.Text

			if dateText := dateEntry.Text; dateText != "" {
				if parsed, err := time.Parse(dateFormat, dateText); err == nil {
					task.DueDate = parsed
				}
			} else {
				task.DueDate = time.Time{}
			}

			if err := db.UpdateTask(task); err != nil {
				dialog.ShowError(err, w)
				return
			}

			onSave()
		},
		w,
	)
}
