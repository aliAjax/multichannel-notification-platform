package routing

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"example.com/notification-platform/internal/notification/domain"
)

type Match struct {
	Subject  string `json:"subject"`
	Pattern  string `json:"pattern"`
	compiled *regexp.Regexp
}

func (m *Match) Compile() error {
	if m.Subject != "tenant" && m.Subject != "business_id" && m.Subject != "target" {
		return errors.New("unsupported match subject")
	}
	if len(m.Pattern) > 256 {
		return errors.New("match pattern too long")
	}
	r, err := regexp.Compile(m.Pattern)
	if err != nil {
		return err
	}
	m.compiled = r
	return nil
}
func (m Match) Matches(value string) bool { return m.compiled != nil && m.compiled.MatchString(value) }

type Rule struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	Channel       domain.Channel `json:"channel"`
	Priority      int            `json:"priority"`
	Matches       []Match        `json:"matches"`
	ProviderNames []string       `json:"provider_names"`
	Weight        int            `json:"weight"`
	Region        string         `json:"region,omitempty"`
	Enabled       bool           `json:"enabled"`
	Version       int64          `json:"version"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (r *Rule) Validate() error {
	if r.ID == "" || r.TenantID == "" {
		return errors.New("rule id and tenant required")
	}
	if len(r.ProviderNames) == 0 {
		return errors.New("at least one provider required")
	}
	if r.Weight < 1 || r.Weight > 1000 {
		return errors.New("weight must be positive")
	}
	for i := range r.Matches {
		if err := r.Matches[i].Compile(); err != nil {
			return err
		}
	}
	return nil
}

type PolicySet struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

func NewPolicySet() *PolicySet { return &PolicySet{rules: map[string]Rule{}} }
func (p *PolicySet) Put(r Rule) error {
	if err := r.Validate(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if old, ok := p.rules[r.ID]; ok {
		if old.Version != r.Version {
			return errors.New("policy version conflict")
		}
		r.Version++
	} else {
		r.Version = 1
	}
	r.UpdatedAt = time.Now().UTC()
	p.rules[r.ID] = r
	return nil
}
func (p *PolicySet) Delete(id string, version int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.rules[id]
	if !ok {
		return errors.New("policy not found")
	}
	if r.Version != version {
		return errors.New("policy version conflict")
	}
	delete(p.rules, id)
	return nil
}
func (p *PolicySet) List(tenant string) []Rule {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := []Rule{}
	for _, r := range p.rules {
		if tenant == "" || r.TenantID == tenant {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].ID < out[j].ID
		}
		return out[i].Priority < out[j].Priority
	})
	return out
}
func (p *PolicySet) Select(tenant string, ch domain.Channel, attributes map[string]string) []Rule {
	rules := p.List(tenant)
	out := []Rule{}
	for _, r := range rules {
		if !r.Enabled || r.Channel != ch {
			continue
		}
		matched := true
		for _, m := range r.Matches {
			if !m.Matches(strings.TrimSpace(attributes[m.Subject])) {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, r)
		}
	}
	return out
}

type Decision struct {
	PolicyID      string    `json:"policy_id"`
	PolicyVersion int64     `json:"policy_version"`
	Provider      string    `json:"provider"`
	Reason        string    `json:"reason"`
	EvaluatedAt   time.Time `json:"evaluated_at"`
}

func Explain(rules []Rule, provider string) Decision {
	d := Decision{Provider: provider, Reason: "no matching policy", EvaluatedAt: time.Now().UTC()}
	if len(rules) > 0 {
		d.PolicyID = rules[0].ID
		d.PolicyVersion = rules[0].Version
		d.Reason = "highest priority matching policy"
	}
	return d
}
