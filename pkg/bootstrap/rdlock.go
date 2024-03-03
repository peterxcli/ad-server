package bootstrap

import (
	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
)

func NewRdLock(redis *redis.Client) *redislock.Client {
	return redislock.New(redis)
}
