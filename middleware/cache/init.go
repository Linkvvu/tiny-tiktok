package cache

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var Ctx context.Context = context.TODO()

func init() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	err := Rdb.Ping(Ctx).Err()
	if err != nil {
		log.Panicln("failed to connect target database from Redis, detail:", err)
	}
}
