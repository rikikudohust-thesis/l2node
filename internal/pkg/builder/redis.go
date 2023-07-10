package builder

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"

	"github.com/go-redis/redis/v8"
)

type redisConfig struct {
	Host     string `default:"localhost"`
	Port     int    `default:"6379"`
	DBNumber int    `default:"0"`
	Password string `default:"password"`
	Prefix   string `default:""`
}

type service struct {
	config *redisConfig
	rdb    *redis.Client
}

func New(cfg *redisConfig) (model.IService, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DBNumber,
	})

	if _, err := rdb.Ping(gocontext.Background()).Result(); err != nil {
		return nil, err
	}

	return &service{config: cfg, rdb: rdb}, nil
}

func (s *service) Set(ctx gocontext.Context, key string, value interface{}, expiration time.Duration) error {
	cacheEntry, err := json.Marshal(value)
	if err != nil {
		return err
	}

	k := fmt.Sprintf("%s:%s", s.config.Prefix, key)
	return s.rdb.Set(ctx, k, cacheEntry, expiration).Err()
}

func (s *service) Get(ctx gocontext.Context, key string, src interface{}) error {
	k := fmt.Sprintf("%s:%s", s.config.Prefix, key)
	val, err := s.rdb.Get(ctx, k).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), &src)
}
