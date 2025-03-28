package store

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/peterbourgon/diskv"
)

// DiskStore is a storage engine that writes to the disk
type DiskStore struct{}

// NewDiskStore will instantiate the disk storage
func NewDiskStore() *DiskStore {
	return &DiskStore{}
}

// Ping will check if the connection works right
func (s DiskStore) Ping(ctx context.Context) error {
	// TODO not sure what can fail here
	return nil
}

// WriteUser will write a user object to disk
func (s DiskStore) WriteUser(user User) {
	log.Printf("DiskStore: Writing user: %+v", user) // Add logging
	s.writeField(user.ID, "username", user.Username)
	s.writeField(user.ID, "access", user.AccessToken)
	s.writeField(user.ID, "refresh", user.RefreshToken)
	s.writeField(user.ID, "expires_at", user.TokenExpiresAt.Format(time.RFC3339)) // Save TokenExpiresAt
}

// GetUser will load a user from disk
func (s DiskStore) GetUser(id string) *User {
	un, err := s.readField(id, "username")
	if err != nil {
		return nil
	}
	ac, err := s.readField(id, "access")
	if err != nil {
		return nil
	}
	re, err := s.readField(id, "refresh")
	if err != nil {
		return nil
	}
	expiresAtStr, err := s.readField(id, "expires_at")
	if err != nil {
		return nil
	}
	tokenExpiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)

	user := User{
		ID:             id,
		Username:       strings.ToLower(un),
		AccessToken:    ac,
		RefreshToken:   re,
		TokenExpiresAt: tokenExpiresAt,
		Store:          s, // Updated field name
	}

	return &user
}

func (s DiskStore) DeleteUser(id string) bool {
	s.eraseField(id, "username")
	s.eraseField(id, "access")
	s.eraseField(id, "refresh")
	s.eraseField(id, "expires_at") // Remove TokenExpiresAt field
	return true
}

func (s DiskStore) writeField(id, field, value string) {
	err := s.write(fmt.Sprintf("%s.%s", id, field), value)
	if err != nil {
		panic(err)
	}
}

func (s DiskStore) readField(id, field string) (string, error) {
	return s.read(fmt.Sprintf("%s.%s", id, field))
}

func (s DiskStore) eraseField(id, field string) error {
	d := diskv.New(diskv.Options{
		BasePath:     "keystore",
		Transform:    flatTransform,
		CacheSizeMax: 1024 * 1024,
	})
	return d.Erase(fmt.Sprintf("%s.%s", id, field))
}

func (s DiskStore) write(key, value string) error {
	d := diskv.New(diskv.Options{
		BasePath:     "keystore",
		Transform:    flatTransform,
		CacheSizeMax: 1024 * 1024,
	})
	return d.Write(key, []byte(value))
}

func (s DiskStore) read(key string) (string, error) {
	d := diskv.New(diskv.Options{
		BasePath:     "keystore",
		Transform:    flatTransform,
		CacheSizeMax: 1024 * 1024,
	})
	value, err := d.Read(key)
	return string(value), err
}
