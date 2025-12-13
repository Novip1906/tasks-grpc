package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Novip1906/tasks-grpc/tasks/internal/models"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	db  *sql.DB
	log *slog.Logger
}

func NewPostgresStorage(host, port, user, password, dbname string, log *slog.Logger) (*PostgresStorage, error) {
	psqlInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("cannot open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to db: %w", err)
	}

	s := &PostgresStorage{db: db, log: log}

	if err := s.init(); err != nil {
		return nil, fmt.Errorf("canot initialize db schema: %w", err)
	}

	return s, nil
}

func (s *PostgresStorage) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		text TEXT,
		author_id INT REFERENCES users (id),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := s.db.Exec(schema)
	return err
}

func (s *PostgresStorage) CreateTask(userId int64, text string) (id int64, err error) {
	query := "INSERT INTO tasks (text, author_id) VALUES ($1, $2) RETURNING id"
	err = s.db.QueryRow(query, text, userId).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *PostgresStorage) GetTaskById(userId, taskId int64) (*models.Task, error) {
	query := `
        SELECT t.id, t.text, u.username, t.created_at, u.id
        FROM tasks t 
        JOIN users u ON t.author_id = u.id 
        WHERE t.id=$1`

	var task models.Task

	err := s.db.QueryRow(query, taskId).Scan(&task.Id, &task.Text, &task.AuthorName, &task.CreatedAt, &task.AuthorId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}

	if userId != task.AuthorId {
		return nil, ErrNotTaskAuthor
	}

	return &task, nil
}

func (s *PostgresStorage) GetAllUserTasks(userId int64) ([]*models.Task, error) {
	query := `
	SELECT t.id, t.text, u.username, t.created_at, t.author_id
	FROM tasks t 
	JOIN users u ON t.author_id = u.id 
	WHERE t.author_id=$1 
	ORDER BY t.created_at DESC`

	rows, err := s.db.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var (
			task models.Task
		)
		err := rows.Scan(&task.Id, &task.Text, &task.AuthorName, &task.CreatedAt, &task.AuthorId)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, &task)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *PostgresStorage) UpdateTask(userId, taskId int64, newText string) (*models.Task, error) {
	task, err := s.GetTaskById(userId, taskId)
	if err != nil {
		return nil, err
	}

	if task.Text == newText {
		return task, nil
	}

	query := "UPDATE tasks SET text=$1 WHERE id=$2"
	_, err = s.db.Exec(query, newText, taskId)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *PostgresStorage) GetAllTasksForIndexing() ([]*models.Task, error) {
	query := `
	SELECT t.id, t.text, u.username, t.author_id, t.created_at
	FROM tasks t
	JOIN users u ON t.author_id = u.id`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.Id,
			&task.Text,
			&task.AuthorName,
			&task.AuthorId,
			&task.CreatedAt,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *PostgresStorage) DeleteTask(userId, taskId int64) (deletedTask *models.Task, err error) {
	task, err := s.GetTaskById(userId, taskId)
	if err != nil {
		return nil, err
	}

	query := "DELETE FROM tasks WHERE id=$1"
	_, err = s.db.Exec(query, taskId)
	if err != nil {
		return nil, err
	}
	return task, nil
}
