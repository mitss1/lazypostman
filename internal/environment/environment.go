package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Environment represents a Postman environment file
type Environment struct {
	Name   string     `json:"name"`
	Values []Variable `json:"values"`
}

type Variable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type,omitempty"`
}

// Manager handles environment variables and substitution
type Manager struct {
	environments []*Environment
	active       int // index of active environment, -1 for none
	globals      map[string]string
}

var varPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func NewManager() *Manager {
	return &Manager{
		active:  -1,
		globals: make(map[string]string),
	}
}

// LoadEnvironment reads and parses a Postman environment file
func (m *Manager) LoadEnvironment(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("reading environment: %w", err)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("parsing environment: %w", err)
	}

	m.environments = append(m.environments, &env)
	if m.active == -1 {
		m.active = 0
	}
	return nil
}

// AddEnvironment adds a pre-parsed environment to the manager
func (m *Manager) AddEnvironment(env *Environment) {
	m.environments = append(m.environments, env)
	if m.active == -1 {
		m.active = 0
	}
}

// SetActive sets the active environment by index
func (m *Manager) SetActive(index int) {
	if index >= -1 && index < len(m.environments) {
		m.active = index
	}
}

// ActiveEnvironment returns the current active environment
func (m *Manager) ActiveEnvironment() *Environment {
	if m.active >= 0 && m.active < len(m.environments) {
		return m.environments[m.active]
	}
	return nil
}

// Environments returns all loaded environments
func (m *Manager) Environments() []*Environment {
	return m.environments
}

// SetGlobal sets a global variable
func (m *Manager) SetGlobal(key, value string) {
	m.globals[key] = value
}

// Resolve replaces {{variable}} placeholders in the input string
func (m *Manager) Resolve(input string) string {
	if !strings.Contains(input, "{{") {
		return input
	}

	vars := m.buildVarMap()
	return varPattern.ReplaceAllStringFunc(input, func(match string) string {
		key := match[2 : len(match)-2]
		if val, ok := vars[key]; ok {
			return val
		}
		return match // leave unresolved variables as-is
	})
}

// buildVarMap creates a merged variable map (globals < active env)
func (m *Manager) buildVarMap() map[string]string {
	vars := make(map[string]string)

	// Start with globals
	for k, v := range m.globals {
		vars[k] = v
	}

	// Override with active environment
	if env := m.ActiveEnvironment(); env != nil {
		for _, v := range env.Values {
			if v.Enabled {
				vars[v.Key] = v.Value
			}
		}
	}

	return vars
}
