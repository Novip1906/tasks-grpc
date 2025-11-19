package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 8
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

	storage := &PostgresStorage{db: db, log: log}

	if err := storage.init(); err != nil {
		return nil, fmt.Errorf("cannot initialize db schema: %w", err)
	}

	return storage, nil
}

func (s *PostgresStorage) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.db.Exec(schema)
	return err
}

func (s *PostgresStorage) CheckUsernamePassword(username, password string) (userID int64, email string, err error) {
	var passwordFromDB string

	query := "SELECT id, email, password FROM users WHERE username = $1"
	err = s.db.QueryRow(query, username).Scan(&userID, &email, &passwordFromDB)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", ErrUserNotFound
	}
	if err != nil {
		return 0, "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordFromDB), []byte(password))
	if err != nil {
		return 0, "", ErrWrongPassword
	}

	return userID, email, nil
}

func (s *PostgresStorage) CheckUsernameExists(username string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)"
	err := s.db.QueryRow(query, username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PostgresStorage) CheckEmailExists(email string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)"
	err := s.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PostgresStorage) AddUser(username, password, email string) (int64, error) {
	usernameExists, err := s.CheckUsernameExists(username)
	if err != nil {
		return 0, err
	}
	if usernameExists {
		return 0, ErrUserAlreadyExists
	}

	if email != "" {
		emailExists, err := s.CheckEmailExists(email)
		if err != nil {
			return 0, err
		}
		if emailExists {
			return 0, ErrEmailAlreadyExists
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return 0, err
	}

	var userID int64
	query := "INSERT INTO users (username, password, email) VALUES ($1, $2, $3) RETURNING id"

	err = s.db.QueryRow(query, username, hashedPassword, email).Scan(&userID)
	if err != nil {
		return 0, err
	}

	s.log.Debug("user added successfully", "user_id", userID, "username", username)
	return userID, nil
}

func (s *PostgresStorage) SetEmail(userID int64, email string) error {
	emailExists, err := s.CheckEmailExists(email)
	if err != nil {
		return err
	}
	if emailExists {
		return ErrEmailAlreadyExists
	}

	query := "UPDATE users SET email = $1 WHERE id = $2"

	s.log.Debug("setting email", "user_id", userID, "email", email)
	_, err = s.db.Exec(query, email, userID)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
