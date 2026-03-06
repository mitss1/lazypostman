package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LoginModel handles API key input
type LoginModel struct {
	input   string
	cursor  int
	visible bool
	err     string
	width   int
	height  int
}

func NewLoginModel() LoginModel {
	return LoginModel{}
}

func (m *LoginModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *LoginModel) Show() {
	m.visible = true
	m.input = ""
	m.cursor = 0
	m.err = ""
}

func (m *LoginModel) Hide() {
	m.visible = false
}

func (m *LoginModel) IsVisible() bool {
	return m.visible
}

func (m *LoginModel) SetError(err string) {
	m.err = err
}

func (m *LoginModel) TypeChar(ch rune) {
	m.input = m.input[:m.cursor] + string(ch) + m.input[m.cursor:]
	m.cursor++
}

func (m *LoginModel) Paste(text string) {
	m.input = m.input[:m.cursor] + text + m.input[m.cursor:]
	m.cursor += len(text)
}

func (m *LoginModel) Backspace() {
	if m.cursor > 0 {
		m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
		m.cursor--
	}
}

func (m *LoginModel) Value() string {
	return m.input
}

func (m *LoginModel) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#3465a4")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render(" Postman Login ") + "\n\n")
	b.WriteString("  Enter your Postman API key to connect.\n")
	b.WriteString(dimText("  Get one at: https://postman.co/settings/me/api-keys") + "\n\n")

	// Input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3465a4")).
		Width(60).
		Padding(0, 1)

	var display string
	if len(m.input) == 0 {
		display = "█"
	} else {
		// Show PMAK-prefix if present, mask the rest, show last 4
		display = strings.Repeat("•", len(m.input))
		if len(m.input) > 4 {
			display = strings.Repeat("•", len(m.input)-4) + m.input[len(m.input)-4:]
		}
		display += "█"
	}

	// Show character count
	countText := dimText(fmt.Sprintf("  %d chars", len(m.input)))

	b.WriteString("  " + inputStyle.Render(display) + "\n")
	b.WriteString(countText + "\n")

	if m.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc0000"))
		b.WriteString("\n  " + errStyle.Render(m.err) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimText("  Enter=submit  Esc=cancel"))

	return b.String()
}
