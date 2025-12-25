package core

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient is the minimal queue interface used by API/worker.
// It supports visibility timeout and explicit ack to avoid job loss.
type RedisClient interface {
	Enqueue(ctx context.Context, pendingKey string, value string) error
	Reserve(ctx context.Context, pendingKey, processingKey string, visibility time.Duration) (string, error)
	Ack(ctx context.Context, processingKey string, value string) error
	RequeueExpired(ctx context.Context, processingKey, pendingKey string, now time.Time) ([]string, error)
}

// RedisClientRaw exposes a minimal subset used for metrics and heartbeat.
type RedisClientRaw interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd
	ZCount(ctx context.Context, key, min, max string) *redis.IntCmd
}

// RedisQueue implements RedisClient using go-redis.
type RedisQueue struct {
	client *redis.Client
}

// NewRedisClient returns a configured go-redis client from URL (e.g., redis://localhost:6379/0).
func NewRedisClient(redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, errors.New("empty redis url")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// NewRedisQueue wraps a redis.Client with queue helpers.
func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

// Enqueue pushes a value to the head of the pending list (LPUSH).
func (q *RedisQueue) Enqueue(ctx context.Context, pendingKey string, value string) error {
	return q.client.LPush(ctx, pendingKey, value).Err()
}

// Reserve moves an item atomically from pending -> processing with a visibility deadline score.
// It uses RPOP + ZADD so the job is not lost if a worker dies before ack.
func (q *RedisQueue) Reserve(ctx context.Context, pendingKey, processingKey string, visibility time.Duration) (string, error) {
	// Lua script:
	// local v=redis.call('RPOP', KEYS[1]); if v then redis.call('ZADD', KEYS[2], ARGV[1], v) end; return v
	script := redis.NewScript(`
local v = redis.call('RPOP', KEYS[1])
if v then
  redis.call('ZADD', KEYS[2], ARGV[1], v)
end
return v
`)
	expireScore := float64(time.Now().Add(visibility).UnixMilli())
	res, err := script.Run(ctx, q.client, []string{pendingKey, processingKey}, expireScore).Result()
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", redis.Nil
	}
	if s, ok := res.(string); ok {
		return s, nil
	}
	return "", errors.New("unexpected reserve response type")
}

// Ack removes a processing item after successful handling.
func (q *RedisQueue) Ack(ctx context.Context, processingKey string, value string) error {
	return q.client.ZRem(ctx, processingKey, value).Err()
}

// RequeueExpired moves expired processing items back to pending and returns the moved jobs.
func (q *RedisQueue) RequeueExpired(ctx context.Context, processingKey, pendingKey string, now time.Time) ([]string, error) {
	// Lua script:
	// local vals = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
	// if #vals > 0 then redis.call('ZREM', KEYS[1], unpack(vals)); redis.call('LPUSH', KEYS[2], unpack(vals)) end
	// return vals
	script := redis.NewScript(`
local vals = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
local count = table.getn(vals)
if count > 0 then
  redis.call('ZREM', KEYS[1], unpack(vals))
  redis.call('LPUSH', KEYS[2], unpack(vals))
end
return vals
`)
	score := float64(now.UnixMilli())
	res, err := script.Run(ctx, q.client, []string{processingKey, pendingKey}, score).Result()
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	rawVals, ok := res.([]interface{})
	if !ok {
		return nil, errors.New("unexpected requeue response type")
	}
	out := make([]string, 0, len(rawVals))
	for _, v := range rawVals {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}
