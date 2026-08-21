package provider

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/notification-platform/internal/notification/domain"
)

type CredentialReference struct {
	SecretName string `json:"secret_name"`
	SecretKey  string `json:"secret_key"`
	Version    string `json:"version,omitempty"`
}

func (r CredentialReference) Validate() error {
	if strings.TrimSpace(r.SecretName) == "" {
		return errors.New("secret name is required")
	}
	if strings.TrimSpace(r.SecretKey) == "" {
		return errors.New("secret key is required")
	}
	if strings.Contains(r.SecretName, "..") || strings.Contains(r.SecretKey, "..") {
		return errors.New("secret reference contains invalid path component")
	}
	return nil
}

type Configuration struct {
	ID          string              `json:"id"`
	TenantID    string              `json:"tenant_id"`
	Name        string              `json:"name"`
	Channel     domain.Channel      `json:"channel"`
	Region      string              `json:"region"`
	Priority    int                 `json:"priority"`
	Enabled     bool                `json:"enabled"`
	Credentials CredentialReference `json:"credentials"`
	RatePerSec  int                 `json:"rate_per_second"`
	Concurrency int                 `json:"concurrency"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Version     int64               `json:"version"`
}

func (c *Configuration) Validate() error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.TenantID) == "" {
		return errors.New("provider id and tenant are required")
	}
	switch c.Channel {
	case domain.ChannelEmail, domain.ChannelSMS, domain.ChannelWebhook, domain.ChannelPush:
	default:
		return errors.New("unsupported provider channel")
	}
	if strings.TrimSpace(c.Region) == "" {
		return errors.New("provider region is required")
	}
	if c.Priority < 0 || c.Priority > 1000 {
		return errors.New("provider priority must be between 0 and 1000")
	}
	if c.RatePerSec < 1 || c.Concurrency < 1 {
		return errors.New("provider capacity must be positive")
	}
	return c.Credentials.Validate()
}

type Registry struct {
	mu      sync.RWMutex
	configs map[string]Configuration
}

func NewRegistry() *Registry {
	return &Registry{configs: make(map[string]Configuration)}
}

func (r *Registry) Register(c Configuration) error {
	if err := c.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.configs[c.ID]; exists {
		return errors.New("provider configuration already exists")
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	c.Version = 1
	r.configs[c.ID] = c
	return nil
}

func (r *Registry) Update(c Configuration, expectedVersion int64) error {
	if err := c.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	previous, exists := r.configs[c.ID]
	if !exists {
		return errors.New("provider configuration not found")
	}
	if previous.Version != expectedVersion {
		return errors.New("provider configuration version conflict")
	}
	c.CreatedAt = previous.CreatedAt
	c.UpdatedAt = time.Now().UTC()
	c.Version = previous.Version + 1
	r.configs[c.ID] = c
	return nil
}

func (r *Registry) SetEnabled(id string, enabled bool, expectedVersion int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, exists := r.configs[id]
	if !exists {
		return errors.New("provider configuration not found")
	}
	if c.Version != expectedVersion {
		return errors.New("provider configuration version conflict")
	}
	c.Enabled = enabled
	c.Version++
	c.UpdatedAt = time.Now().UTC()
	r.configs[id] = c
	return nil
}

func (r *Registry) Get(id string) (Configuration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, exists := r.configs[id]
	if !exists {
		return Configuration{}, nil
	}
	return c, nil
}

func (r *Registry) List(tenant string, channel domain.Channel) []Configuration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Configuration, 0)
	for _, c := range r.configs {
		if tenant != "" && c.TenantID != tenant {
			continue
		}
		if channel != "" && c.Channel != channel {
			continue
		}
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].ID < result[j].ID
		}
		return result[i].Priority < result[j].Priority
	})
	return result
}
