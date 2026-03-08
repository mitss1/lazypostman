package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	m := NewManager()
	m.AddEnvironment(&Environment{
		Name: "test",
		Values: []Variable{
			{Key: "host", Value: "api.example.com", Enabled: true},
			{Key: "token", Value: "abc123", Enabled: true},
		},
	})

	tests := []struct {
		input string
		want  string
	}{
		{"{{host}}/users", "api.example.com/users"},
		{"Bearer {{token}}", "Bearer abc123"},
		{"no vars here", "no vars here"},
		{"{{unknown}}", "{{unknown}}"},
		{"{{host}}/{{token}}", "api.example.com/abc123"},
	}

	for _, tt := range tests {
		got := m.Resolve(tt.input)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveDisabledVar(t *testing.T) {
	m := NewManager()
	m.AddEnvironment(&Environment{
		Name: "test",
		Values: []Variable{
			{Key: "host", Value: "api.example.com", Enabled: false},
		},
	})

	got := m.Resolve("{{host}}/users")
	if got != "{{host}}/users" {
		t.Errorf("expected unresolved, got %q", got)
	}
}

func TestBuildVarMapPrecedence(t *testing.T) {
	m := NewManager()
	m.SetGlobal("host", "global.example.com")
	m.SetGlobal("only_global", "global_value")

	m.AddEnvironment(&Environment{
		Name: "test",
		Values: []Variable{
			{Key: "host", Value: "env.example.com", Enabled: true},
		},
	})

	// Environment should override global
	got := m.Resolve("{{host}}")
	if got != "env.example.com" {
		t.Errorf("expected env to override global, got %q", got)
	}

	// Global-only var should still resolve
	got = m.Resolve("{{only_global}}")
	if got != "global_value" {
		t.Errorf("expected global var to resolve, got %q", got)
	}
}

func TestLoadEnvironment(t *testing.T) {
	envJSON := `{
		"name": "Test Env",
		"values": [
			{"key": "host", "value": "test.com", "enabled": true},
			{"key": "port", "value": "8080", "enabled": true}
		]
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "test.postman_environment.json")
	if err := os.WriteFile(path, []byte(envJSON), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager()
	if err := m.LoadEnvironment(path); err != nil {
		t.Fatalf("LoadEnvironment() error: %v", err)
	}

	if len(m.Environments()) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(m.Environments()))
	}

	env := m.ActiveEnvironment()
	if env == nil {
		t.Fatal("expected active environment")
	}
	if env.Name != "Test Env" {
		t.Errorf("expected name 'Test Env', got %q", env.Name)
	}

	got := m.Resolve("{{host}}:{{port}}")
	if got != "test.com:8080" {
		t.Errorf("expected 'test.com:8080', got %q", got)
	}
}

func TestLoadEnvironmentInvalidPath(t *testing.T) {
	m := NewManager()
	err := m.LoadEnvironment("/nonexistent/env.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadEnvironmentInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	if err := m.LoadEnvironment(path); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSetActive(t *testing.T) {
	m := NewManager()
	m.AddEnvironment(&Environment{Name: "env1", Values: []Variable{{Key: "k", Value: "v1", Enabled: true}}})
	m.AddEnvironment(&Environment{Name: "env2", Values: []Variable{{Key: "k", Value: "v2", Enabled: true}}})

	// First env is auto-active
	if got := m.Resolve("{{k}}"); got != "v1" {
		t.Errorf("expected v1, got %q", got)
	}

	m.SetActive(1)
	if got := m.Resolve("{{k}}"); got != "v2" {
		t.Errorf("expected v2, got %q", got)
	}

	// Invalid index should be ignored
	m.SetActive(99)
	if got := m.Resolve("{{k}}"); got != "v2" {
		t.Errorf("expected v2 after invalid SetActive, got %q", got)
	}
}

func TestNoActiveEnvironment(t *testing.T) {
	m := NewManager()
	if env := m.ActiveEnvironment(); env != nil {
		t.Error("expected nil active environment")
	}
	// Should return input unchanged
	got := m.Resolve("{{host}}")
	if got != "{{host}}" {
		t.Errorf("expected unresolved, got %q", got)
	}
}
