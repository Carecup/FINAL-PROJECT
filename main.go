package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"strconv"
	"time"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("postgres", "postgres://postgres:postgres@192.168.0.102:5432/todo?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Task struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Content   string `json:"content"`
	Completed bool   `json:"completed"`
}

type Project struct {
	ID          int       `json:"id"`          // Уникальный идентификатор проекта
	Name        string    `json:"name"`        // Название проекта
	Description string    `json:"description"` // Описание проекта
	CreatedAt   time.Time `json:"created_at"`  // Время создания проекта
}

type Comment struct {
	ID        int       `json:"id"`         // Уникальный идентификатор комментария
	TaskID    int       `json:"task_id"`    // Идентификатор задачи, к которой относится комментарий
	UserID    int       `json:"user_id"`    // Идентификатор пользователя, оставившего комментарий
	Content   string    `json:"content"`    // Содержимое комментария
	CreatedAt time.Time `json:"created_at"` // Время создания комментария
}

type Tag struct {
	ID        int       `json:"id"`         // Уникальный идентификатор тега
	Name      string    `json:"name"`       // Название тега
	CreatedAt time.Time `json:"created_at"` // Время создания тега
}

type TaskTag struct {
	TaskID int `json:"task_id"` // Идентификатор задачи
	TagID  int `json:"tag_id"`  // Идентификатор тега
}

var (
	users      []User
	tasks      []Task
	nextUserID int = 1
	nextTaskID int = 1
)

func createUser(w http.ResponseWriter, r *http.Request) {
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.QueryRow("INSERT INTO users(name) VALUES($1) RETURNING id", newUser.Name).Scan(&newUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(newUser)
}

func createProject(w http.ResponseWriter, r *http.Request) {
	var project Project
	err := json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.QueryRow("INSERT INTO projects(name, description) VALUES($1, $2) RETURNING id", project.Name, project.Description).Scan(&project.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(project)
}

func getProjects(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, description FROM projects")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	json.NewEncoder(w).Encode(projects)
}

func addCommentToTask(w http.ResponseWriter, r *http.Request) {
	var comment Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.QueryRow("INSERT INTO comments (task_id, user_id, content) VALUES ($1, $2, $3) RETURNING id", comment.TaskID, comment.UserID, comment.Content).Scan(&comment.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(comment)
}

func getCommentsByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	rows, err := db.Query("SELECT id, task_id, user_id, content FROM comments WHERE task_id = $1", taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		comments = append(comments, c)
	}

	json.NewEncoder(w).Encode(comments)
}

func createTag(w http.ResponseWriter, r *http.Request) {
	var tag Tag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.QueryRow("INSERT INTO tags (name) VALUES ($1) RETURNING id", tag.Name).Scan(&tag.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(tag)
}

func getTags(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM tags")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tags = append(tags, t)
	}

	json.NewEncoder(w).Encode(tags)
}

func assignTagToTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID, err := strconv.Atoi(vars["task_id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	tagID, err := strconv.Atoi(vars["tag_id"])
	if err != nil {
		http.Error(w, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	if _, err := db.Exec("INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)", taskID, tagID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func removeTagFromTask(w http.ResponseWriter, r *http.Request) {
	taskIDStr := r.URL.Query().Get("task_id")
	tagIDStr := r.URL.Query().Get("tag_id")

	taskID, err := strconv.Atoi(taskIDStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil {
		http.Error(w, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	// Выполнение запроса на удаление
	if _, err := db.Exec("DELETE FROM task_tags WHERE task_id = $1 AND tag_id = $2", taskID, tagID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	rows, err := db.Query("SELECT id, user_id, content, completed FROM tasks WHERE user_id = $1", userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Content, &task.Completed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(tasks)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	var newTask Task
	err := json.NewDecoder(r.Body).Decode(&newTask)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.QueryRow("INSERT INTO tasks(user_id, content, completed) VALUES($1, $2, $3) RETURNING id", newTask.UserID, newTask.Content, newTask.Completed).Scan(&newTask.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(newTask)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	taskID, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM tasks WHERE id = $1", taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Task with ID %d deleted", taskID)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	taskID, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var updatedTask Task
	err = json.NewDecoder(r.Body).Decode(&updatedTask)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE tasks SET user_id = $1, content = $2, completed = $3 WHERE id = $4", updatedTask.UserID, updatedTask.Content, updatedTask.Completed, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedTask.ID = taskID
	json.NewEncoder(w).Encode(updatedTask)
}

func main() {
	initDB() // Инициализация базы данных

	router := mux.NewRouter()

	router.HandleFunc("/user", createUser).Methods("POST")
	router.HandleFunc("/todo", getTasks).Methods("GET")
	router.HandleFunc("/todo", createTask).Methods("POST")
	router.HandleFunc("/todo/{id}", deleteTask).Methods("DELETE")
	router.HandleFunc("/todo/{id}", updateTask).Methods("PUT")

	// Обработчики для проектов
	router.HandleFunc("/project", createProject).Methods("POST")
	router.HandleFunc("/projects", getProjects).Methods("GET")

	// Обработчики для комментариев
	router.HandleFunc("/comment", addCommentToTask).Methods("POST")
	router.HandleFunc("/comments", getCommentsByTask).Methods("GET") // Может потребоваться параметр для указания задачи

	// Обработчики для тегов
	router.HandleFunc("/tag", createTag).Methods("POST")
	router.HandleFunc("/tags", getTags).Methods("GET")
	router.HandleFunc("/task/{task_id}/tag/{tag_id}", assignTagToTask).Methods("POST")
	router.HandleFunc("/task/{task_id}/tag/{tag_id}", removeTagFromTask).Methods("DELETE")

	// Запуск сервера
	log.Fatal(http.ListenAndServe(":8080", router))
}
