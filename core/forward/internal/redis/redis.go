package redis

import (
	"context"
	"encoding/json"
	"fmt"
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

// GetClient returns the Redis client (nil if not initialized).
func GetClient() *redis.Client {
	return rdb
}

// IsAvailable checks if Redis is connected and available.
func IsAvailable() bool {
	if rdb == nil {
		return false
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		return false
	}
	return true
}

// PublishSensorData publishes filtered sensor data to Redis pub/sub channel.
// Production service subscribes to this channel for real-time updates.
func PublishSensorData(data interface{}) {
	if rdb == nil {
		return
	}

	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("❌ Redis: failed to marshal sensor data: %v", err)
		return
	}

	if err := rdb.Publish(ctx, "sensor:data:live", string(payload)).Err(); err != nil {
		log.Printf("❌ Redis: publish failed: %v", err)
		return
	}
}

// PublishSensorStatus publishes sensor status changes (online/offline) to Redis.
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

// CacheKey helpers
func SensorDataKey(areaID, sensorNo int) string {
	return fmt.Sprintf("cache:sensor:data:%d:%d", areaID, sensorNo)
}

func AreaNameKey(areaID int) string {
	return fmt.Sprintf("cache:area:name:%d", areaID)
}

func DoorInfoKey(doorID int) string {
	return fmt.Sprintf("cache:door:info:%d", doorID)
}
