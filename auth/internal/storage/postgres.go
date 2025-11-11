package storage

import (
	"database/sql"
	"errors"
	"fmt"

	appErrors "github.com/Novip1906/tasks-grpc/auth/internal/errors"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCode = 8

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(host, port, user, password, dbname string) (*PostgresStorage, error) {
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

	s := &PostgresStorage{db: db}

	if err := s.init(); err != nil {
		return nil, fmt.Errorf("canot initialize db schema: %w", err)
	}

	return s, nil
}

func (s *PostgresStorage) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := s.db.Exec(schema)
	return err
}

func (s *PostgresStorage) CheckUser(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCode)
	if err != nil {
		return appErrors.ErrDBInternal
	}

	var passwordFromDB string

	query := "SELECT password FROM WHERE username=?"
	err = s.db.QueryRow(query, username, hashedPassword).Scan(&passwordFromDB)
	if errors.Is(err, sql.ErrNoRows) {
		return appErrors.ErrUserNotFound
	}
	if err != nil {
		return appErrors.ErrDBInternal
	}

	if passwordFromDB != string(hashedPassword) {
		return appErrors.ErrWrongPassword
	}
	return nil
}

func (s *PostgresStorage) AddUser(username, password string) error {
	if username == "" || password == "" {
		return errors.New("invalid params")
	}

	if err := s.CheckUser(username, password); err == nil {
		return appErrors.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCode)
	if err != nil {
		return appErrors.ErrDBInternal
	}

	query := "INSERT INTO users (username, password) VALUES (?, ?)"
	_, err = s.db.Exec(query, username, hashedPassword)
	if err != nil {
		return appErrors.ErrDBInternal
	}

	return nil
}
