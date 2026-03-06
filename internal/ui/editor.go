package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// EditorMode represents what field we're editing
type EditorMode int

const (
	EditNone EditorMode = iota
	EditURL
	EditHeaderKey
	EditHeaderValue
	EditBody
	EditParamKey
	EditParamValue
	EditEnvKey
	EditEnvValue
)

// EditorModel handles inline text editing
type EditorModel struct {
	mode      EditorMode
	label     string
	input     string
	cursor    int
	visible   bool
	multiline bool
	width     int
	height    int
	// For key-value editing
	index int // which header/param/env var we're editing
}

func NewEditorModel() EditorModel {
	return EditorModel{}
}

func (m *EditorModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *EditorModel) Open(mode EditorMode, label, value string, index int) {
	m.mode = mode
	m.label = label
	m.input = value
	m.cursor = len(value)
	m.index = index
	m.visible = true
	m.multiline = (mode == EditBody)
}

func (m *EditorModel) Close() {
	m.visible = false
	m.mode = EditNone
}

func (m *EditorModel) IsVisible() bool {
	return m.visible
}

func (m *EditorModel) Mode() EditorMode {
	return m.mode
}

func (m *EditorModel) Index() int {
	return m.index
}

func (m *EditorModel) Value() string {
	return m.input
}

func (m *EditorModel) TypeChar(ch rune) {
	m.input = m.input[:m.cursor] + string(ch) + m.input[m.cursor:]
	m.cursor++
}

func (m *EditorModel) Paste(text string) {
	m.input = m.input[:m.cursor] + text + m.input[m.cursor:]
	m.cursor += len(text)
}

func (m *EditorModel) Backspace() {
	if m.cursor > 0 {
		m.input = m.input[:m.cursor-1] + m.input[m.cursor:]
		m.cursor--
	}
}

func (m *EditorModel) Delete() {
	if m.cursor < len(m.input) {
		m.input = m.input[:m.cursor] + m.input[m.cursor+1:]
	}
}

func (m *EditorModel) MoveLeft() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *EditorModel) MoveRight() {
	if m.cursor < len(m.input) {
		m.cursor++
	}
}

func (m *EditorModel) Home() {
	if m.multiline {
		// Move to start of current line
		lineStart := strings.LastIndex(m.input[:m.cursor], "\n")
		if lineStart == -1 {
			m.cursor = 0
		} else {
			m.cursor = lineStart + 1
		}
	} else {
		m.cursor = 0
	}
}

func (m *EditorModel) End() {
	if m.multiline {
		// Move to end of current line
		lineEnd := strings.Index(m.input[m.cursor:], "\n")
		if lineEnd == -1 {
			m.cursor = len(m.input)
		} else {
			m.cursor += lineEnd
		}
	} else {
		m.cursor = len(m.input)
	}
}

func (m *EditorModel) NewLine() {
	if m.multiline {
		m.input = m.input[:m.cursor] + "\n" + m.input[m.cursor:]
		m.cursor++
	}
}

func (m *EditorModel) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#f57900")).
		Padding(0, 1)

	b.WriteString(titleStyle.Render(fmt.Sprintf(" Edit: %s ", m.label)) + "\n\n")

	if m.multiline {
		b.WriteString(m.renderMultiline())
	} else {
		b.WriteString(m.renderSingleLine())
	}

	b.WriteString("\n\n")
	if m.multiline {
		b.WriteString(dimText("  Enter=newline  Ctrl+S=save  Esc=cancel"))
	} else {
		b.WriteString(dimText("  Enter=save  Esc=cancel"))
	}

	return b.String()
}

func (m *EditorModel) renderSingleLine() string {
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#f57900")).
		Width(m.width - 8).
		Padding(0, 1)

	display := m.input
	if m.cursor >= len(display) {
		display += "█"
	} else {
		display = display[:m.cursor] + "█" + display[m.cursor+1:]
	}

	return "  " + inputStyle.Render(display)
}

func (m *EditorModel) renderMultiline() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#f57900")).
		Width(m.width - 8).
		Height(m.height - 10).
		Padding(0, 1)

	// Insert cursor marker
	display := m.input
	if m.cursor >= len(display) {
		display += "█"
	} else {
		display = display[:m.cursor] + "█" + display[m.cursor+1:]
	}

	lines := strings.Split(display, "\n")
	numbered := make([]string, len(lines))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	for i, line := range lines {
		numbered[i] = numStyle.Render(fmt.Sprintf("%3d ", i+1)) + line
	}

	return "  " + borderStyle.Render(strings.Join(numbered, "\n"))
}
