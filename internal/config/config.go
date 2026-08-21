package config

import (
	"errors"
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
				apply(&c, strings.TrimSpace(p[0]), strings.Trim(strings.TrimSpace(p[1]), "\"'"))
			}
		}
	}
	for _, x := range []struct{ env, key string }{{"NOTIFY_HTTP_ADDR", "http_addr"}, {"NOTIFY_DATA_FILE", "data_file"}, {"NOTIFY_WEBHOOK_SECRET", "webhook_secret"}, {"NOTIFY_WORKER_CONCURRENCY", "worker_concurrency"}, {"NOTIFY_QUEUE_CAPACITY", "queue_capacity"}, {"NOTIFY_PROVIDER_TIMEOUT", "provider_timeout"}, {"NOTIFY_MAX_ATTEMPTS", "max_attempts"}} {
		if v, ok := os.LookupEnv(x.env); ok {
			apply(&c, x.key, v)
		}
	}
	if c.WorkerConcurrency < 1 || c.QueueCapacity < 1 || c.MaxAttempts < 1 {
		return c, errors.New("numeric configuration must be positive")
	}
	return c, nil
}
func apply(c *Config, k, v string) {
	switch k {
	case "http_addr":
		c.HTTPAddr = v
	case "data_file":
		c.DataFile = v
	case "webhook_secret":
		c.WebhookSecret = v
	case "worker_concurrency":
		c.WorkerConcurrency, _ = strconv.Atoi(v)
	case "queue_capacity":
		c.QueueCapacity, _ = strconv.Atoi(v)
	case "max_attempts":
		c.MaxAttempts, _ = strconv.Atoi(v)
	case "provider_timeout":
		c.ProviderTimeout, _ = time.ParseDuration(v)
	case "shutdown_timeout":
		c.ShutdownTimeout, _ = time.ParseDuration(v)
	}
}
