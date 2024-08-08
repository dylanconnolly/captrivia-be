package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	gameKey string = "game:%s"
)

var ctx = context.Background()

type Question struct {
}

func NewClient(addr string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		// Addr:     "redis:6379" // for docker container
		// Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return rdb
}
