package nonce

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type redisNoncer struct {
	client *redis.Client
}

func NewRedisNoncer(addr string) *redisNoncer {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})
	pingResult := client.Ping()
	err := pingResult.Err()
	if err != nil {
		panic(fmt.Sprintf("Could not connect to Redis Client at Address %s", addr))
	}
	return &redisNoncer{
		client,
	}
}

func (noncer *redisNoncer) Next() Nonce {
	nonce := generateRandomNonce()
	noncer.client.Set(string(nonce), time.Now().UTC().Format(time.RFC3339), time.Hour)
	return nonce
}

func (noncer *redisNoncer) Valid(nonce Nonce) bool {
	_, err := noncer.client.Get(string(nonce)).Result()
	if err == redis.Nil {
		return false
	}
	noncer.client.Del(string(nonce))
	return true
}

func (noncer *redisNoncer) Healthy() error {
	if err := noncer.client.Ping().Err(); err != nil {
		return err
	}
	return nil
}
