package template

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"
)

type Variable struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	Sensitive bool   `json:"sensitive"`
}
type Version struct {
	Number    int        `json:"number"`
	Locale    string     `json:"locale"`
	Subject   string     `json:"subject"`
	TextBody  string     `json:"text_body"`
	HTMLBody  string     `json:"html_body"`
	Variables []Variable `json:"variables"`
	Published bool       `json:"published"`
}
type Definition struct {
	ID       string    `json:"id"`
	TenantID string    `json:"tenant_id"`
	Name     string    `json:"name"`
	Versions []Version `json:"versions"`
}
type Service struct {
	mu    sync.RWMutex
	defs  map[string]Definition
	cache map[string]*template.Template
}

func NewService() *Service {
	return &Service{defs: map[string]Definition{}, cache: map[string]*template.Template{}}
}
func (s *Service) Create(d Definition) error {
	if d.ID == "" || d.TenantID == "" {
		return errors.New("template id and tenant required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.defs[d.ID]; ok {
		return errors.New("template already exists")
	}
	s.defs[d.ID] = d
	return nil
}
func (s *Service) AddVersion(id string, v Version) error {
	if err := validateVersion(v); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.defs[id]
	if !ok {
		return errors.New("template not found")
	}
	v.Number = len(d.Versions) + 1
	d.Versions = append(d.Versions, v)
	s.defs[id] = d
	return nil
}
func (s *Service) Publish(id string, version int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.defs[id]
	if !ok {
		return errors.New("template not found")
	}
	found := false
	for i := range d.Versions {
		d.Versions[i].Published = d.Versions[i].Number == version
		found = found || d.Versions[i].Published
	}
	if !found {
		return errors.New("version not found")
	}
	s.defs[id] = d
	return nil
}
func (s *Service) Get(id string, version int) (Definition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.defs[id]
	if !ok {
		return Definition{}, errors.New("template not found")
	}
	if version == 0 {
		for i := len(d.Versions) - 1; i >= 0; i-- {
			if d.Versions[i].Published {
				version = d.Versions[i].Number
				break
			}
		}
	}
	for _, v := range d.Versions {
		if v.Number == version {
			return Definition{ID: d.ID, TenantID: d.TenantID, Name: d.Name, Versions: []Version{v}}, nil
		}
	}
	return Definition{}, errors.New("version not found")
}
func (s *Service) Render(id string, version int, vars map[string]any) (Version, map[string]string, error) {
	d, err := s.Get(id, version)
	if err != nil {
		return Version{}, nil, err
	}
	v := d.Versions[0]
	if err := validateVars(v.Variables, vars); err != nil {
		return Version{}, nil, err
	}
	funcs := template.FuncMap{"safe": func(x any) string { return html.EscapeString(toString(x)) }}
	render := func(name, src string) (string, error) {
		t, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(src)
		if err != nil {
			return "", err
		}
		var b bytes.Buffer
		if err = t.Execute(&b, vars); err != nil {
			return "", err
		}
		return b.String(), nil
	}
	sub, err := render("subject", v.Subject)
	if err != nil {
		return Version{}, nil, err
	}
	txt, err := render("text", v.TextBody)
	if err != nil {
		return Version{}, nil, err
	}
	htm, err := render("html", v.HTMLBody)
	if err != nil {
		return Version{}, nil, err
	}
	v.Subject = sub
	v.TextBody = txt
	v.HTMLBody = htm
	return v, s.redact(vars, v.Variables), nil
}
func (s *Service) redact(vars map[string]any, spec []Variable) map[string]string {
	out := map[string]string{}
	for _, v := range spec {
		if val, ok := vars[v.Name]; ok {
			if v.Sensitive {
				out[v.Name] = redacted
			} else {
				out[v.Name] = toString(val)
			}
		}
	}
	return out
}
const redacted = "[REDACTED]"
func validateVersion(v Version) error {
	if strings.TrimSpace(v.TextBody) == "" && strings.TrimSpace(v.HTMLBody) == "" {
		return errors.New("template body required")
	}
	if len(v.TextBody)+len(v.HTMLBody) > 1<<20 {
		return errors.New("template too large")
	}
	seen := map[string]struct{}{}
	for _, x := range v.Variables {
		if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`).MatchString(x.Name) {
			return errors.New("invalid variable name")
		}
		if _, dup := seen[x.Name]; dup {
			return errors.New("duplicate variable: " + x.Name)
		}
		seen[x.Name] = struct{}{}
	}
	return nil
}
func validateVars(spec []Variable, vars map[string]any) error {
	for _, v := range spec {
		if v.Required {
			if _, ok := vars[v.Name]; !ok {
				return errors.New("missing variable: " + v.Name)
			}
		}
		if val, ok := vars[v.Name]; ok && v.Type != "" {
			if !typeMatches(v.Type, val) {
				return errors.New("variable type mismatch: " + v.Name)
			}
		}
	}
	return nil
}
func typeMatches(t string, val any) bool {
	switch t {
	case "string":
		_, ok := val.(string)
		return ok
	case "int":
		switch x := val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		case float64:
			return x == float64(int64(x))
		}
		return false
	case "bool":
		_, ok := val.(bool)
		return ok
	}
	return true
}
func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(fmtAny(x)), "\n", " "))
	}
}
func fmtAny(v any) string { return fmt.Sprint(v) }
func SortedVariables(v []Variable) []Variable {
	out := append([]Variable(nil), v...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
