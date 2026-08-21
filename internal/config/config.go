package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DataFile          string
	WebhookSecret     string
	WorkerConcurrency int
	QueueCapacity     int
	ProviderTimeout   time.Duration
	MaxAttempts       int
	ShutdownTimeout   time.Duration
}

func Load(path string) (Config, error) {
	c := Config{HTTPAddr: ":8080", DataFile: "data/store.json", WorkerConcurrency: 4, QueueCapacity: 10000, ProviderTimeout: 3 * time.Second, MaxAttempts: 5, ShutdownTimeout: 10 * time.Second}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return c, err
		}
		for _, line := range strings.Split(string(b), "\n") {
			p := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(p) == 2 {
				if e := apply(&c, strings.TrimSpace(p[0]), strings.Trim(strings.TrimSpace(p[1]), "\"'")); e != nil {
					return c, e
				}
			}
		}
	}
	for _, x := range []struct{ env, key string }{{"NOTIFY_HTTP_ADDR", "http_addr"}, {"NOTIFY_DATA_FILE", "data_file"}, {"NOTIFY_WEBHOOK_SECRET", "webhook_secret"}, {"NOTIFY_WORKER_CONCURRENCY", "worker_concurrency"}, {"NOTIFY_QUEUE_CAPACITY", "queue_capacity"}, {"NOTIFY_PROVIDER_TIMEOUT", "provider_timeout"}, {"NOTIFY_MAX_ATTEMPTS", "max_attempts"}} {
		if v, ok := os.LookupEnv(x.env); ok {
			if e := apply(&c, x.key, v); e != nil {
				return c, e
			}
		}
	}
	if c.WorkerConcurrency < 1 || c.QueueCapacity < 1 || c.MaxAttempts < 1 {
		return c, errors.New("numeric configuration must be positive")
	}
	if c.ProviderTimeout <= 0 {
		return c, errors.New("provider_timeout must be a positive duration")
	}
	if c.ShutdownTimeout <= 0 {
		return c, errors.New("shutdown_timeout must be a positive duration")
	}
	return c, nil
}
func apply(c *Config, k, v string) error {
	switch k {
	case "http_addr":
		c.HTTPAddr = v
	case "data_file":
		c.DataFile = v
	case "webhook_secret":
		c.WebhookSecret = v
	case "worker_concurrency":
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("worker_concurrency: %w", err)
		}
		c.WorkerConcurrency = n
	case "queue_capacity":
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("queue_capacity: %w", err)
		}
		c.QueueCapacity = n
	case "max_attempts":
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("max_attempts: %w", err)
		}
		c.MaxAttempts = n
	case "provider_timeout":
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("provider_timeout: %w", err)
		}
		c.ProviderTimeout = d
	case "shutdown_timeout":
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("shutdown_timeout: %w", err)
		}
		c.ShutdownTimeout = d
	}
	return nil
}
