package hocoosmiddleware

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v7"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

type redisClient struct {
	redis    *redis.Client
	replicas int
	logger   *zap.SugaredLogger
}

const (
	connectedSlavesReplicas  = "connected_slaves:"
	infoReplicationDelimiter = "\r\n"
)

func newRedisClient() *redisClient {
	return &redisClient{}
}
func (r *redisClient) init(ctx context.Context, url string, logger *zap.SugaredLogger) error {
	r.logger = logger
	r.logger.Debugf("Initializing redis client with url: %s", url)
	r.logger = logger
	r.logger.Debugf("Parsing redis url: %s", url)
	redisInfo, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("error parsing redis url %s: %w", url, err)
	}
	r.redis = redis.NewClient(redisInfo)
	r.logger.Debugf("Connecting to redis at %s", url)
	_, err = r.redis.WithContext(ctx).Ping().Result()
	if err != nil {
		_ = r.redis.Close()
		return fmt.Errorf("error connecting to redis at %s: %w", redisInfo.Addr, err)
	}
	r.logger.Debugf("Checking redis salves connections")
	r.replicas, err = r.getConnectedSlaves(ctx)
	if err != nil {
		_ = r.redis.Close()
		return fmt.Errorf("error getting connected slaves: %w", err)
	}
	r.logger.Debugf("Connected slaves: %d", r.replicas)
	r.logger.Debugf("Redis client initialized")
	return nil
}

func (r *redisClient) getConnectedSlaves(ctx context.Context) (int, error) {
	res, err := r.redis.DoContext(ctx, "INFO", "replication").Result()
	if err != nil {
		return 0, err
	}

	s, _ := strconv.Unquote(fmt.Sprintf("%q", res))
	if len(s) == 0 {
		return 0, nil
	}

	return r.parseConnectedSlaves(s), nil
}

func (r *redisClient) parseConnectedSlaves(res string) int {
	infos := strings.Split(res, infoReplicationDelimiter)
	for _, info := range infos {
		if strings.Contains(info, connectedSlavesReplicas) {
			parsedReplicas, _ := strconv.ParseUint(info[len(connectedSlavesReplicas):], 10, 32)
			return int(parsedReplicas)
		}
	}

	return 0
}
func (r *redisClient) get(ctx context.Context, key string) (string, error) {
	r.logger.Debugf("Getting key %s from redis", key)
	res, err := r.redis.DoContext(ctx, "HGETALL", key).Result() // Prefer values with ETags
	if err != nil {
		return r.directGet(ctx, key) //Falls back to original get
	}
	if res == nil {
		return "", fmt.Errorf("no data found for this key")
	}
	vals := res.([]interface{})
	if len(vals) == 0 {
		return "", fmt.Errorf("no data found for this key")
	}

	data, _, err := r.getKeyVersion(vals)
	if err != nil {
		return "", fmt.Errorf("error found for get this key, %w", err)
	}
	r.logger.Debugf("Key %s found in redis, Value: %s", key, data)
	return data, nil
}

func (r *redisClient) directGet(ctx context.Context, key string) (string, error) {
	r.logger.Debugf("Getting key %s from redis in direct mode", key)
	res, err := r.redis.DoContext(ctx, "GET", key).Result()
	if err != nil {
		return "", err
	}
	data, err := strconv.Unquote(fmt.Sprintf("%q", res))
	if err != nil {
		return "", err
	}
	r.logger.Debugf("Key %s found in redis, Value: %s", key, data)
	return data, nil
}
func (r *redisClient) getKeyVersion(vals []interface{}) (data string, version string, err error) {
	seenData := false
	seenVersion := false
	for i := 0; i < len(vals); i += 2 {
		field, _ := strconv.Unquote(fmt.Sprintf("%q", vals[i]))
		switch field {
		case "data":
			data, _ = strconv.Unquote(fmt.Sprintf("%q", vals[i+1]))
			seenData = true
		case "version":
			version, _ = strconv.Unquote(fmt.Sprintf("%q", vals[i+1]))
			seenVersion = true
		}
	}
	if !seenData || !seenVersion {
		return "", "", fmt.Errorf("required hash field 'data' or 'version' was not found")
	}
	return data, version, nil
}
