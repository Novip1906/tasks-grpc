package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

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

func (s *PostgresStorage) CheckUser(username, password string) (userId int64, err error) {
	var passwordFromDB string

	query := "SELECT id, password FROM users WHERE username=$1"
	err = s.db.QueryRow(query, username).Scan(&userId, &passwordFromDB)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	if err != nil {
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordFromDB), []byte(password))
	if err != nil {
		return 0, ErrWrongPassword
	}

	return userId, nil
}

func (s *PostgresStorage) AddUser(username, password string) error {
	if username == "" || password == "" {
		return errors.New("invalid params")
	}

	if _, err := s.CheckUser(username, password); err == nil {
		return ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCode)
	if err != nil {
		return err
	}

	query := "INSERT INTO users (username, password) VALUES ($1, $2)"
	_, err = s.db.Exec(query, username, hashedPassword)
	if err != nil {
		return err
	}

	return nil
}
