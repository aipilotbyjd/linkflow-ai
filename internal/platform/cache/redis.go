package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrCacheMiss is returned when a key is not found in cache
	ErrCacheMiss = errors.New("cache miss")
)

// RedisCache implements caching with Redis
type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	keyPrefix  string
}

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	KeyPrefix string
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(cfg Config) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client:     client,
		defaultTTL: 5 * time.Minute,
		keyPrefix:  cfg.KeyPrefix,
	}, nil
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.buildKey(key)
	
	val, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return nil
}

// Set stores a value in cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.buildKey(key)
	
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.buildKey(key)
	
	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.buildKey(key)
	
	result, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// Expire sets expiration for a key
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := c.buildKey(key)
	
	if err := c.client.Expire(ctx, fullKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

// Increment increments a numeric value
func (c *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := c.buildKey(key)
	
	val, err := c.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return val, nil
}

// IncrementBy increments a numeric value by a specific amount
func (c *RedisCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	fullKey := c.buildKey(key)
	
	val, err := c.client.IncrBy(ctx, fullKey, value).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment by: %w", err)
	}

	return val, nil
}

// SetNX sets a value only if it doesn't exist
func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	fullKey := c.buildKey(key)
	
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	result, err := c.client.SetNX(ctx, fullKey, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx: %w", err)
	}

	return result, nil
}

// InvalidatePattern deletes all keys matching a pattern
func (c *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	fullPattern := c.buildKey(pattern)
	
	// Use SCAN to find keys matching pattern
	iter := c.client.Scan(ctx, 0, fullPattern, 0).Iterator()
	
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}
	return nil
}

// Health checks the health of the cache
func (c *RedisCache) Health(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}
	return nil
}

// buildKey builds the full cache key with prefix
func (c *RedisCache) buildKey(key string) string {
	if c.keyPrefix != "" {
		return fmt.Sprintf("%s:%s", c.keyPrefix, key)
	}
	return key
}

// CacheAside implements the cache-aside pattern
func CacheAside[T any](
	ctx context.Context,
	cache *RedisCache,
	key string,
	ttl time.Duration,
	loader func() (T, error),
) (T, error) {
	var result T
	
	// Try cache first
	err := cache.Get(ctx, key, &result)
	if err == nil {
		return result, nil
	}
	
	// If not in cache, load from source
	result, err = loader()
	if err != nil {
		return result, err
	}
	
	// Store in cache (async to not block)
	go func() {
		_ = cache.Set(context.Background(), key, result, ttl)
	}()
	
	return result, nil
}

// Lock implements distributed locking using Redis
type Lock struct {
	cache *RedisCache
	key   string
	value int64
	ttl   time.Duration
}

// NewLock creates a new distributed lock
func (c *RedisCache) NewLock(key string, ttl time.Duration) *Lock {
	return &Lock{
		cache: c,
		key:   fmt.Sprintf("lock:%s", key),
		value: time.Now().UnixNano(), // Simple unique value
		ttl:   ttl,
	}
}

// Acquire tries to acquire the lock
func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	return l.cache.SetNX(ctx, l.key, l.value, l.ttl)
}

// Release releases the lock
func (l *Lock) Release(ctx context.Context) error {
	// Use Lua script to ensure we only delete our own lock
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	
	fullKey := l.cache.buildKey(l.key)
	_, err := l.cache.client.Eval(ctx, script, []string{fullKey}, l.value).Result()
	return err
}
