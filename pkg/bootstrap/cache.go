package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisEnv struct {
	Host     string `env:"HOST"`
	Port     uint   `env:"PORT"`
	Password string `env:"PASSWORD" envDefault:""`
}

func (env *RedisEnv) DSN() string {
	return fmt.Sprintf("%s:%d", env.Host, env.Port)
}

func NewCache(env *Env) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", env.Redis.Host, env.Redis.Port),
		Password: env.Redis.Password,
		DB:       0, // use default DB
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(pong + err.Error())
	}
	return rdb
}
