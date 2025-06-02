// gui.go
package gui

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"
	"strings"
	"sync"
	"time"
	"todolist/db"
	"todolist/models"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	dateFormat    = "02.01.2006"
	userNamesFile = "user_names.json"
)

var (
	userNames   = make(map[int]string)
	userNamesMu sync.RWMutex
)

func addEnterHandler(entry *widget.Entry, callback func()) {
	entry.OnSubmitted = func(s string) {
		callback()
	}
}

func init() {
	loadUserNames()
}

func loadUserNames() {
	data, err := os.ReadFile(userNamesFile)
	if err != nil {
		return
	}

	userNamesMu.Lock()
	defer userNamesMu.Unlock()
	json.Unmarshal(data, &userNames)
}

func saveUserNames() {
	userNamesMu.RLock()
	defer userNamesMu.RUnlock()

	data, err := json.Marshal(userNames)
	if err != nil {
		return
	}
	os.WriteFile(userNamesFile, data, 0644)
}

func ShowUserSelection(w fyne.Window) {
	users, err := db.GetAllUsers()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Ошибка загрузки пользователей: %v", err), w)
		return
	}

	mainContainer := container.NewVBox(
		widget.NewLabelWithStyle("Выберите пользователя", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
	)

	usersContainer := container.NewVBox()
	for _, user := range users {
		u := user
		userName := getUserName(u.ID)

		userRow := container.NewHBox(
			widget.NewButton(userName, func() {
				ShowTodoLists(w, u.ID)
			}),
			layout.NewSpacer(),
			widget.NewButton("✕", func() {
				showDeleteUserDialog(w, u)
			}),
		)
		usersContainer.Add(userRow)
		usersContainer.Add(layout.NewSpacer())
	}

	addButton := widget.NewButton("+ Создать нового пользователя", func() {
		showCreateUserDialog(w)
	})
	addButtonContainer := container.NewHBox(
		layout.NewSpacer(),
		addButton,
		layout.NewSpacer(),
	)

	mainContainer.Add(usersContainer)
	mainContainer.Add(layout.NewSpacer())
	mainContainer.Add(addButtonContainer)
	mainContainer.Add(layout.NewSpacer())

	w.SetContent(mainContainer)
}

func showDeleteUserDialog(w fyne.Window, user models.User) {
	dialog.ShowConfirm(
		"Удаление пользователя",
		fmt.Sprintf("Удалить пользователя '%s' и все его данные?", getUserName(user.ID)),
		func(ok bool) {
			if !ok {
				return
			}

			if err := db.DeleteUser(user.ID); err != nil {
				dialog.ShowError(err, w)
				return
			}

			userNamesMu.Lock()
			delete(userNames, user.ID)
			userNamesMu.Unlock()
			saveUserNames()

			ShowUserSelection(w)
		},
		w,
	)
}

func showCreateUserDialog(w fyne.Window) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Имя пользователя")

	// Создаем диалог
	d := dialog.NewForm(
		"Новый пользователь",
		"Создать",
		"Отмена",
		[]*widget.FormItem{
			{Text: "Имя:", Widget: entry},
		},
		func(confirmed bool) {
			if !confirmed || entry.Text == "" {
				return
			}

			user := models.User{
				TgID: int64(crc32.ChecksumIEEE([]byte(entry.Text))),
			}

			if err := db.CreateUser(&user); err != nil {
				dialog.ShowError(err, w)
				return
			}

			userNamesMu.Lock()
			userNames[user.ID] = entry.Text
			userNamesMu.Unlock()
			saveUserNames()

			ShowTodoLists(w, user.ID)
		},
		w,
	)

	// Показываем диалог
	d.Show()

	// Обработчик Enter
	addEnterHandler(entry, func() {
		if entry.Text == "" {
			return
		}

		user := models.User{
			TgID: int64(crc32.ChecksumIEEE([]byte(entry.Text))),
		}

		if err := db.CreateUser(&user); err != nil {
			dialog.ShowError(err, w)
			return
		}

		userNamesMu.Lock()
		userNames[user.ID] = entry.Text
		userNamesMu.Unlock()
		saveUserNames()

		d.Hide()
		ShowTodoLists(w, user.ID)
	})
}

func getUserName(userID int) string {
	userNamesMu.RLock()
	defer userNamesMu.RUnlock()
	if name, exists := userNames[userID]; exists {
		return name
	}
	return fmt.Sprintf("Пользователь %d", userID)
}

func ShowTodoLists(w fyne.Window, userID int) {
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

		check := widget.NewCheck("", func(checked bool) {
			fmt.Printf("Список %s отмечен как выполненный: %v\n", currentList.Title, checked)
		})

		listBtn := widget.NewButton(currentList.Title, func() {
			ShowTodoItems(w, currentList)
		})
		listBtn.Alignment = widget.ButtonAlignLeading

		deleteBtn := widget.NewButton("✕", func() {
			showDeleteConfirmDialog(w, "Удаление списка", "Удалить этот список и все его задачи?", func() {
				if err := db.DeleteTodoList(currentList.ID); err != nil {
					dialog.ShowError(err, w)
					return
				}
				ShowTodoLists(w, userID)
			})
		})

		listRow := container.NewHBox(
			check,
			listBtn,
			layout.NewSpacer(),
			deleteBtn,
		)

		listsContainer.Add(listRow)
		listsContainer.Add(container.NewPadded(container.NewVBox(layout.NewSpacer())))
	}

	scrollContainer := container.NewVScroll(listsContainer)
	scrollContainer.SetMinSize(fyne.NewSize(w.Canvas().Size().Width*0.9, 300))

	mainContainer.Add(scrollContainer)
	mainContainer.Add(layout.NewSpacer())

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

	backButton := widget.NewButton("← Назад к пользователям", func() {
		ShowUserSelection(w)
	})

	mainContainer.Add(backButton)

	w.SetContent(mainContainer)
}

func showDeleteConfirmDialog(w fyne.Window, title, message string, onConfirm func()) {
	content := widget.NewLabel(message)
	dialog.ShowCustomConfirm(
		title,
		"Да",
		"Нет",
		content,
		func(ok bool) {
			if ok {
				onConfirm()
			}
		},
		w,
	)
}

func showAddListDialog(w fyne.Window, userID int) {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Введите название списка")

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Введите описание")
	descEntry.MultiLine = true
	descEntry.Wrapping = fyne.TextWrapWord

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

	content := container.NewBorder(
		nil, buttons, nil, nil,
		formContent,
	)

	d := dialog.NewCustomWithoutButtons("",
		container.NewPadded(content),
		w,
	)

	createList := func() {
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
			ShowTodoLists(w, userID)
			d.Hide()
		}
	}

	// Добавляем обработчики Enter
	addEnterHandler(titleEntry, createList)
	addEnterHandler(descEntry, createList)

	buttons.Objects[1].(*widget.Button).OnTapped = d.Hide
	buttons.Objects[2].(*widget.Button).OnTapped = createList

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
		ShowTodoLists(w, list.UserID)
	})

	deleteListButton := widget.NewButton("Удалить список", func() {
		showDeleteConfirmDialog(w,
			"Удаление списка",
			"Вы уверены, что хотите удалить этот список и все его задачи?",
			func() {
				if err := db.DeleteTodoList(list.ID); err != nil {
					dialog.ShowError(err, w)
					return
				}
				ShowTodoLists(w, list.UserID)
			})
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

	// Функция обновления текста и цвета задачи
	updateTask := func() {
		text := task.Title
		if !task.DueDate.IsZero() {
			text += " (" + task.DueDate.Format(dateFormat) + ")"
		}
		taskBtn.SetText(text)

		// Устанавливаем цвет в зависимости от статуса и даты
		if task.IsDone {
			taskBtn.Importance = widget.LowImportance // Серый для выполненных
		} else if !task.DueDate.IsZero() && task.DueDate.Before(time.Now()) {
			taskBtn.Importance = widget.DangerImportance // Красный для просроченных
		} else {
			taskBtn.Importance = widget.MediumImportance // Обычный цвет
		}
	}
	updateTask()

	taskBtn.OnTapped = func() {
		showTaskDetails(w, task, updateTask)
	}

	check := widget.NewCheck("", func(done bool) {
		task.IsDone = done
		if err := db.UpdateTask(task); err != nil {
			dialog.ShowError(err, w)
			return
		}
		updateTask() // Обновляем цвет после изменения статуса
	})
	check.SetChecked(task.IsDone)

	deleteBtn := widget.NewButton("✕", func() {
		showDeleteConfirmDialog(w,
			"Удаление",
			"Удалить задачу?",
			func() {
				if err := db.DeleteTask(task.ID); err != nil {
					dialog.ShowError(err, w)
					return
				}
				ShowTodoItems(w, list)
			})
	})

	return container.NewHBox(
		check,
		taskBtn,
		layout.NewSpacer(),
		deleteBtn,
	)
}

func showAddTaskDialog(w fyne.Window, list models.TodoList) {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Название задачи")
	titleEntry.Validator = func(s string) error {
		if s == "" {
			return fmt.Errorf("введите название задачи")
		}
		return nil
	}

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Описание задачи")
	descEntry.MultiLine = true
	descEntry.Wrapping = fyne.TextWrapWord

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("дд.мм.гггг")

	// Автоформатирование даты
	dateEntry.OnChanged = func(s string) {
		if s == "" {
			return
		}

		cursorPos := dateEntry.CursorColumn
		cleanStr := strings.ReplaceAll(s, ".", "")

		if len(cleanStr) > 8 {
			cleanStr = cleanStr[:8]
			cursorPos = 10
		}

		var formatted strings.Builder
		for i, r := range cleanStr {
			if i == 2 || i == 4 {
				formatted.WriteRune('.')
				if cursorPos > i {
					cursorPos++
				}
			}
			formatted.WriteRune(r)
		}

		newText := formatted.String()
		if newText != s {
			dateEntry.SetText(newText)
			if cursorPos < len(newText) {
				dateEntry.CursorColumn = cursorPos
			} else {
				dateEntry.CursorColumn = len(newText)
			}
		}
	}

	// Создаем контейнер с формой
	form := widget.NewForm(
		widget.NewFormItem("Название:", titleEntry),
		widget.NewFormItem("Описание:", descEntry),
		widget.NewFormItem("Срок:", dateEntry),
	)

	// Создаем кнопки
	submitBtn := widget.NewButton("Добавить", nil)
	cancelBtn := widget.NewButton("Отмена", nil)

	buttons := container.NewHBox(
		layout.NewSpacer(),
		cancelBtn,
		submitBtn,
	)

	// Собираем основной контент
	content := container.NewVBox(
		form,
		buttons,
	)

	// Создаем диалог
	d := dialog.NewCustomWithoutButtons("Новая задача", content, w)

	// Назначаем действия кнопкам после создания диалога
	submitBtn.OnTapped = func() {
		if titleEntry.Validate() != nil {
			return
		}

		var dueDate time.Time
		if dateText := dateEntry.Text; dateText != "" {
			parsed, err := time.Parse(dateFormat, dateText)
			if err != nil {
				dialog.ShowError(fmt.Errorf("неверный формат даты. Используйте дд.мм.гггг"), w)
				return
			}
			dueDate = parsed
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

		d.Hide()
		ShowTodoItems(w, list)
	}

	cancelBtn.OnTapped = func() {
		d.Hide()
	}

	// Обработчики Enter
	addEnterHandler(titleEntry, func() {
		descEntry.FocusGained()
	})

	addEnterHandler(descEntry, func() {
		dateEntry.FocusGained()
	})

	addEnterHandler(dateEntry, func() {
		if titleEntry.Validate() == nil {
			submitBtn.OnTapped()
		}
	})

	d.Resize(fyne.NewSize(400, 300))
	d.Show()
}

func showTaskDetails(w fyne.Window, task *models.Task, onUpdate func()) {
	// Создаем элементы интерфейса
	titleLabel := widget.NewLabelWithStyle(task.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	descLabel := widget.NewLabel(task.Description)
	descLabel.Wrapping = fyne.TextWrapWord

	// Создаем лейбл для даты
	dateText := "Срок не установлен"
	if !task.DueDate.IsZero() {
		dateText = "Срок: " + task.DueDate.Format(dateFormat)
		if task.DueDate.Before(time.Now()) && !task.IsDone {
			dateText += " (ПРОСРОЧЕНО)"
		}
	}

	dateLabel := widget.NewLabel(dateText)
	if !task.DueDate.IsZero() && task.DueDate.Before(time.Now()) && !task.IsDone {
		dateLabel = widget.NewLabelWithStyle(dateText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		dateLabel.Importance = widget.DangerImportance // Устанавливаем красный цвет через Importance
	}

	// Кнопка редактирования
	editBtn := widget.NewButton("Редактировать", func() {
		editTaskDialog(w, task, func() {
			onUpdate()
			// Обновляем текст
			titleLabel.SetText(task.Title)
			descLabel.SetText(task.Description)

			// Обновляем дату
			newDateText := "Срок не установлен"
			if !task.DueDate.IsZero() {
				newDateText = "Срок: " + task.DueDate.Format(dateFormat)
				if task.DueDate.Before(time.Now()) && !task.IsDone {
					newDateText += " (ПРОСРОЧЕНО)"
				}
			}
			dateLabel.SetText(newDateText)

			// Обновляем стиль
			if !task.DueDate.IsZero() && task.DueDate.Before(time.Now()) && !task.IsDone {
				dateLabel.Importance = widget.DangerImportance
			} else {
				dateLabel.Importance = widget.MediumImportance
			}
		})
	})

	// Создаем контейнер с содержимым
	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		descLabel,
		dateLabel,
		layout.NewSpacer(),
		editBtn,
	)

	// Создаем и показываем диалог
	d := dialog.NewCustom(
		"Детали задачи",
		"Закрыть",
		content,
		w,
	)
	d.Show()
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

	// Ограничение длины ввода
	dateEntry.Validator = func(s string) error {
		cleanStr := strings.ReplaceAll(s, ".", "")
		if len(cleanStr) > 8 {
			return fmt.Errorf("максимум 8 цифр (ддммгггг)")
		}
		return nil
	}

	// Автоматическое форматирование даты
	dateEntry.OnChanged = func(s string) {
		if s == "" {
			return
		}

		// Удаляем все нецифры
		var cleanStr strings.Builder
		for _, r := range s {
			if r >= '0' && r <= '9' {
				cleanStr.WriteRune(r)
			}
		}

		// Форматируем с точками
		var formatted strings.Builder
		digits := cleanStr.String()
		for i, r := range digits {
			if i == 2 || i == 4 {
				formatted.WriteRune('.')
			}
			if i >= 8 { // Ограничиваем 8 цифрами
				break
			}
			formatted.WriteRune(r)
		}

		// Обновляем поле ввода
		newText := formatted.String()
		if newText != s {
			dateEntry.SetText(newText)
			dateEntry.CursorColumn = len(newText)
		}
	}

	d := dialog.NewForm(
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

			// Проверка даты перед сохранением
			if dateText := dateEntry.Text; dateText != "" {
				if _, err := time.Parse(dateFormat, dateText); err != nil {
					dialog.ShowError(fmt.Errorf("неверная дата. Формат: дд.мм.гггг"), w)
					return
				}
			}

			// Остальная логика сохранения...
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
	d.Show()
}
