package store

import (
	"context"
	"fmt"
	"log" // Add log package
	"strings"
	"time"

	"database/sql"

	// Postgres db library loading
	_ "github.com/lib/pq"
)

// PostgresqlStore is a storage engine that writes to postgres
type PostgresqlStore struct {
	db *sql.DB
}

// NewPostgresqlClient creates a new db client object
func NewPostgresqlClient(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	rows, err := db.Query(`
		CREATE TABLE IF NOT EXISTS users (
			id varchar(255) NOT NULL,
			username varchar(255) NOT NULL,
			access varchar(255) NOT NULL,
			refresh varchar(255) NOT NULL,
			expires_at timestamp with time zone NOT NULL,
			PRIMARY KEY(id)
		)
	`)
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	return db
}

// NewPostgresqlStore creates new store
func NewPostgresqlStore(db *sql.DB) PostgresqlStore {
	return PostgresqlStore{
		db: db,
	}
}

// Ping will check if the connection works right
func (s PostgresqlStore) Ping(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return conn.PingContext(ctx)
}

// WriteUser will write a user object to postgres
func (s PostgresqlStore) WriteUser(user User) {
	log.Printf("PostgresqlStore: Writing user: %+v", user) // Add logging
	_, err := s.db.Exec(
		`
			INSERT INTO users
				(id, username, access, refresh, expires_at)
				VALUES($1, $2, $3, $4, $5)
			ON CONFLICT(id)
			DO UPDATE set username=EXCLUDED.username, access=EXCLUDED.access, refresh=EXCLUDED.refresh, expires_at=EXCLUDED.expires_at
		`,
		user.ID,
		user.Username,
		user.AccessToken,
		user.RefreshToken,
		user.TokenExpiresAt,
	)
	if err != nil {
		log.Printf("PostgresqlStore: Error writing user: %v", err) // Add error logging
		panic(err)
	}
}

// GetUser will load a user from postgres
func (s PostgresqlStore) GetUser(id string) *User {
	var username, access, refresh string
	var tokenExpiresAt time.Time

	err := s.db.QueryRow(
		"SELECT username, access, refresh, expires_at FROM users WHERE id=$1",
		id,
	).Scan(
		&username,
		&access,
		&refresh,
		&tokenExpiresAt,
	)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("PostgresqlStore: No user with id %s", id)
		return nil
	case err != nil:
		log.Printf("PostgresqlStore: Query error: %v", err)
		panic(fmt.Errorf("query error: %v", err))
	}

	user := User{
		ID:             id,
		Username:       strings.ToLower(username),
		AccessToken:    access,
		RefreshToken:   refresh,
		TokenExpiresAt: tokenExpiresAt,
		Store:          s, // Updated field name
	}

	return &user
}

func (s PostgresqlStore) DeleteUser(id string) bool {
	_, err := s.db.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		log.Printf("PostgresqlStore: Error deleting user %s: %v", id, err)
		return false
	}
	return true
}
