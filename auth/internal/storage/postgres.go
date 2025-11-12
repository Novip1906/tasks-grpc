package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	appErrors "github.com/Novip1906/tasks-grpc/auth/internal/errors"
	"github.com/Novip1906/tasks-grpc/auth/internal/logging"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCode = 8

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
	log := s.log.With(
		slog.String("op", "db.CheckUser"),
		slog.String("username", username),
	)

	var passwordFromDB string

	query := "SELECT password FROM users WHERE username=$1"
	err := s.db.QueryRow(query, username).Scan(&passwordFromDB)
	if errors.Is(err, sql.ErrNoRows) {
		return appErrors.ErrUserNotFound
	}
	if err != nil {
		log.Error("query error", logging.Err(err))
		return appErrors.ErrDBInternal
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordFromDB), []byte(password))
	if err != nil {
		return appErrors.ErrWrongPassword
	}

	return nil
}

func (s *PostgresStorage) AddUser(username, password string) error {
	log := s.log.With(
		slog.String("op", "db.AddUser"),
		slog.String("username", username),
	)

	if username == "" || password == "" {
		return errors.New("invalid params")
	}

	if err := s.CheckUser(username, password); err == nil {
		return appErrors.ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCode)
	if err != nil {
		log.Error("bcrypt error", logging.Err(err))
		return appErrors.ErrDBInternal
	}

	query := "INSERT INTO users (username, password) VALUES ($1, $2)"
	_, err = s.db.Exec(query, username, hashedPassword)
	if err != nil {
		log.Error("query error", logging.Err(err))
		return appErrors.ErrDBInternal
	}

	return nil
}
