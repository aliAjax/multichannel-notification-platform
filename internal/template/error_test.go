package template

import "testing"

func TestRenderRedactsSensitiveVariables(t *testing.T) {
	s := NewService()
	_ = s.Create(Definition{ID: "d", TenantID: "t"})
	_ = s.AddVersion("d", Version{TextBody: "hi {{.phone}}", Variables: []Variable{{Name: "phone", Required: true, Sensitive: true}}})
	_, diag, e := s.Render("d", 1, map[string]any{"phone": "555-1212"})
	if e != nil {
		t.Fatal(e)
	}
	if diag["phone"] == "555-1212" {
		t.Fatal("sensitive value leaked")
	}
}
func TestDuplicateVariablesRejected(t *testing.T) {
	e := validateVersion(Version{TextBody: "x", Variables: []Variable{{Name: "name"}, {Name: "name"}}})
	if e == nil {
		t.Fatal("duplicate variable accepted")
	}
}
func TestVariableTypeMismatchRejected(t *testing.T) {
	e := validateVars([]Variable{{Name: "count", Type: "int", Required: true}}, map[string]any{"count": "many"})
	if e == nil {
		t.Fatal("type mismatch accepted")
	}
}
