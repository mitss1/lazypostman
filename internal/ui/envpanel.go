package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mitss1/lazypostman/internal/environment"
)

// EnvPanelModel shows and allows editing environment variables
type EnvPanelModel struct {
	envManager *environment.Manager
	cursor     int
	visible    bool
	width      int
	height     int
}

func NewEnvPanelModel(envMgr *environment.Manager) EnvPanelModel {
	return EnvPanelModel{envManager: envMgr}
}

func (m *EnvPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *EnvPanelModel) Show() {
	m.visible = true
	m.cursor = 0
}

func (m *EnvPanelModel) Hide() {
	m.visible = false
}

func (m *EnvPanelModel) IsVisible() bool {
	return m.visible
}

func (m *EnvPanelModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *EnvPanelModel) MoveDown() {
	env := m.envManager.ActiveEnvironment()
	if env == nil {
		return
	}
	if m.cursor < len(env.Values)-1 {
		m.cursor++
	}
}

// SelectedVar returns the key and value of the selected variable
func (m *EnvPanelModel) SelectedVar() (key, value string, index int) {
	env := m.envManager.ActiveEnvironment()
	if env == nil || m.cursor >= len(env.Values) {
		return "", "", -1
	}
	v := env.Values[m.cursor]
	return v.Key, v.Value, m.cursor
}

// UpdateVar updates the value of a variable at the given index
func (m *EnvPanelModel) UpdateVar(index int, value string) {
	env := m.envManager.ActiveEnvironment()
	if env == nil || index >= len(env.Values) {
		return
	}
	env.Values[index].Value = value
}

// ToggleVar enables/disables the selected variable
func (m *EnvPanelModel) ToggleSelected() {
	env := m.envManager.ActiveEnvironment()
	if env == nil || m.cursor >= len(env.Values) {
		return
	}
	env.Values[m.cursor].Enabled = !env.Values[m.cursor].Enabled
}

func (m *EnvPanelModel) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	env := m.envManager.ActiveEnvironment()
	if env == nil {
		b.WriteString(dimText("  No active environment\n\n"))
		b.WriteString(dimText("  Press 'e' to cycle environments or 'E' to load from Postman Cloud"))
		return b.String()
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#3465a4")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render(fmt.Sprintf(" Environment: %s ", env.Name)) + "\n\n")

	if len(env.Values) == 0 {
		b.WriteString(dimText("  No variables defined"))
		return b.String()
	}

	// Find max key length for alignment
	maxKeyLen := 0
	for _, v := range env.Values {
		if len(v.Key) > maxKeyLen {
			maxKeyLen = len(v.Key)
		}
	}

	visibleHeight := m.height - 8
	if visibleHeight < 5 {
		visibleHeight = 5
	}
	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(env.Values) {
		end = len(env.Values)
	}

	for i := start; i < end; i++ {
		v := env.Values[i]
		selected := i == m.cursor

		status := "●"
		statusColor := "#73d216"
		if !v.Enabled {
			status = "○"
			statusColor = "#555555"
		}
		statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(status)

		key := v.Key
		// Pad key for alignment
		if len(key) < maxKeyLen {
			key += strings.Repeat(" ", maxKeyLen-len(key))
		}

		val := v.Value
		if v.Type == "secret" {
			if len(val) > 4 {
				val = strings.Repeat("•", len(val)-4) + val[len(val)-4:]
			} else {
				val = strings.Repeat("•", len(val))
			}
		}

		line := fmt.Sprintf("  %s %s = %s", statusStyled, keyText(key), valText(val))

		if selected {
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("#3465a4")).
				Foreground(lipgloss.Color("#ffffff")).
				Width(m.width - 6)
			line = selStyle.Render(line)
		}

		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimText("  Enter=edit value  Space=toggle  e=cycle env  Esc=close"))

	return b.String()
}
