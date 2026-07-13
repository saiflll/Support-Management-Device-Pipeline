package redis

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb  *redis.Client
	once sync.Once
	ctx  = context.Background()
)

// InitRedis initializes the Redis client from REDIS_URL env var.
// Returns nil if REDIS_URL is not set (Redis is optional).
func InitRedis() *redis.Client {
	once.Do(func() {
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			log.Println("ℹ️ REDIS_URL not set — Redis integration disabled")
			return
		}

		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Printf("⚠️ Invalid REDIS_URL '%s': %v — Redis disabled", redisURL, err)
			return
		}

		opts.DialTimeout = 3 * time.Second
		opts.ReadTimeout = 3 * time.Second
		opts.WriteTimeout = 3 * time.Second

		client := redis.NewClient(opts)

		// Test connection
		if err := client.Ping(ctx).Err(); err != nil {
			log.Printf("⚠️ Redis connection failed: %v — Redis disabled", err)
			return
		}

		rdb = client
		log.Println("✅ Redis connected!")
	})
	return rdb
}

// IsAvailable checks if Redis is connected and available.
func IsAvailable() bool {
	if rdb == nil {
		return false
	}
	return rdb.Ping(ctx).Err() == nil
}

// SetCache stores a key-value pair with TTL in Redis cache.
func SetCache(key string, value interface{}, ttl time.Duration) {
	if rdb == nil {
		return
	}

	var data string
	switch v := value.(type) {
	case string:
		data = v
	default:
		b, err := json.Marshal(value)
		if err != nil {
			log.Printf("❌ Redis SetCache: marshal error: %v", err)
			return
		}
		data = string(b)
	}

	if err := rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("❌ Redis SetCache: %v", err)
	}
}

// GetCache retrieves a value from Redis cache.
func GetCache(key string) (string, bool) {
	if rdb == nil {
		return "", false
	}

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

// DelCache removes one or more keys from Redis cache.
func DelCache(keys ...string) {
	if rdb == nil {
		return
	}
	if err := rdb.Del(ctx, keys...).Err(); err != nil {
		log.Printf("❌ Redis DelCache: %v", err)
	}
}

// SubscribeSensorStatus subscribes to sensor status changes via Redis pub/sub.
// The handler is called asynchronously for each status change event.
func SubscribeSensorStatus(handler func(msg map[string]interface{})) {
	if rdb == nil {
		log.Println("⚠️ Redis not available — sensor:status:change subscription disabled")
		return
	}

	pubsub := rdb.Subscribe(ctx, "sensor:status:change")

	go func() {
		ch := pubsub.Channel()
		log.Println("✅ Subscribed to Redis channel: sensor:status:change")
		for msg := range ch {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
				log.Printf("⚠️ Redis: failed to parse status change: %v", err)
				continue
			}
			handler(data)
		}
	}()
}

// PublishSensorStatus publishes a sensor status change (Production-side).
func PublishSensorStatus(sensorKey string, status string, areaID, sensorNo int, value float64) {
	if rdb == nil {
		return
	}

	msg := map[string]interface{}{
		"sensor_key": sensorKey,
		"status":     status,
		"area_id":    areaID,
		"sensor_no":  sensorNo,
		"value":      value,
		"timestamp":  time.Now().Unix(),
	}

	payload, _ := json.Marshal(msg)
	rdb.Publish(ctx, "sensor:status:change", string(payload))
}

// Cache key helpers
const (
	MasterProductKey = "cache:master:produk"
	CacheTTL         = 30 * time.Minute
)
