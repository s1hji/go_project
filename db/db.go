package db

import (
	"database/sql"
	"fmt"
	"hash/crc32"
	"log"
	"todolist/models"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "todo"
)

var DB *sql.DB

func Init() error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения: %v", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ошибка проверки подключения: %v", err)
	}

	log.Println("Подключение к БД успешно")
	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func GetOrCreateDefaultUser() (int, error) {

	tgID := int64(crc32.ChecksumIEEE([]byte("student")))

	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE tg_id = $1", tgID).Scan(&userID)
	if err == sql.ErrNoRows {
		err = DB.QueryRow(
			"INSERT INTO users (tg_id) VALUES ($1) RETURNING id",
			tgID,
		).Scan(&userID)
	}
	return userID, err
}

func CreateTodoList(list *models.TodoList) error {
	return DB.QueryRow(
		"INSERT INTO todo_lists (user_id, title, description, created_at) VALUES ($1, $2, $3, $4) RETURNING id",
		list.UserID, list.Title, list.Description, list.CreatedAt,
	).Scan(&list.ID)
}

func GetTodoLists(userID int) ([]models.TodoList, error) {
	rows, err := DB.Query(
		"SELECT id, user_id, title, description, created_at FROM todo_lists WHERE user_id = $1 ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []models.TodoList
	for rows.Next() {
		var list models.TodoList
		if err := rows.Scan(&list.ID, &list.UserID, &list.Title, &list.Description, &list.CreatedAt); err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, rows.Err()
}

func DeleteTodoList(listID int) error {
	_, err := DB.Exec("DELETE FROM todo_lists WHERE id = $1", listID)
	return err
}

func CreateTask(task *models.Task) error {
	return DB.QueryRow(
		"INSERT INTO tasks (list_id, title, description, due_date, is_done, created_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		task.ListID, task.Title, task.Description, task.DueDate, task.IsDone, task.CreatedAt,
	).Scan(&task.ID)
}

func GetTasksByList(listID int) ([]models.Task, error) {
	rows, err := DB.Query(
		"SELECT id, list_id, title, description, due_date, is_done, created_at FROM tasks WHERE list_id = $1 ORDER BY due_date NULLS LAST, created_at DESC",
		listID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(&task.ID, &task.ListID, &task.Title, &task.Description, &task.DueDate, &task.IsDone, &task.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func UpdateTask(task *models.Task) error {
	_, err := DB.Exec(
		"UPDATE tasks SET title = $1, description = $2, due_date = $3, is_done = $4 WHERE id = $5",
		task.Title, task.Description, task.DueDate, task.IsDone, task.ID,
	)
	return err
}

func DeleteTask(taskID int) error {
	_, err := DB.Exec("DELETE FROM tasks WHERE id = $1", taskID)
	return err
}
