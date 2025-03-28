package store

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

// RedisStore is a storage engine that writes to redis
type RedisStore struct {
	client redis.Client
}

// NewRedisClient creates a new redis client object
func NewRedisClient(addr string, password string) redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	_, err := client.Ping().Result()
	// FIXME
	if err != nil {
		panic(err)
	}
	return *client
}

// NewRedisStore creates new store
func NewRedisStore(client redis.Client) RedisStore {
	return RedisStore{
		client: client,
	}
}

// Ping will check if the connection works right
func (s RedisStore) Ping(ctx context.Context) error {
	_, err := s.client.WithContext(ctx).Ping().Result()
	return err
}

// WriteUser will write a user object to redis
func (s RedisStore) WriteUser(user User) {
	log.Printf("RedisStore: Writing user: %+v", user) // Add logging
	data := map[string]interface{}{
		"username":   user.Username,
		"access":     user.AccessToken,
		"refresh":    user.RefreshToken,
		"expires_at": user.TokenExpiresAt.Format(time.RFC3339), // Save TokenExpiresAt
	}
	err := s.client.HMSet("goplaxt:user:"+user.ID, data).Err()
	if err != nil {
		log.Printf("RedisStore: Error writing user: %v", err) // Add error logging
		panic(err)
	}
}

// GetUser will load a user from redis
func (s RedisStore) GetUser(id string) *User {
	data, err := s.client.HGetAll("goplaxt:user:" + id).Result()
	if err != nil || len(data) == 0 {
		log.Printf("RedisStore: Error or no data for user %s: %v", id, err)
		return nil
	}

	tokenExpiresAt, _ := time.Parse(time.RFC3339, data["expires_at"])

	user := User{
		ID:             id,
		Username:       strings.ToLower(data["username"]),
		AccessToken:    data["access"],
		RefreshToken:   data["refresh"],
		TokenExpiresAt: tokenExpiresAt,
		Store:          s,
	}

	return &user
}

// DeleteUser will remove all fields associated with the user in Redis
func (s RedisStore) DeleteUser(id string) bool {
	err := s.client.Del("goplaxt:user:" + id).Err()
	if err != nil {
		log.Printf("RedisStore: Error deleting user %s: %v", id, err)
		return false
	}
	return true
}
