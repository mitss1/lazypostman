package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mitss1/lazypostman/internal/postman"
)

// BrowserMode represents what we're browsing
type BrowserMode int

const (
	BrowseCollections BrowserMode = iota
	BrowseEnvironments
)

// BrowserModel handles browsing Postman Cloud resources
type BrowserModel struct {
	collections  []postman.CollectionInfo
	environments []postman.EnvironmentInfo
	mode         BrowserMode
	cursor       int
	width        int
	height       int
	loading      bool
	err          error
	visible      bool
}

func NewBrowserModel() BrowserModel {
	return BrowserModel{}
}

func (m *BrowserModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *BrowserModel) Show(mode BrowserMode) {
	m.mode = mode
	m.cursor = 0
	m.visible = true
	m.err = nil
}

func (m *BrowserModel) Hide() {
	m.visible = false
}

func (m *BrowserModel) IsVisible() bool {
	return m.visible
}

func (m *BrowserModel) SetCollections(cols []postman.CollectionInfo) {
	m.collections = cols
	m.loading = false
	m.cursor = 0
}

func (m *BrowserModel) SetEnvironments(envs []postman.EnvironmentInfo) {
	m.environments = envs
	m.loading = false
	m.cursor = 0
}

func (m *BrowserModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m *BrowserModel) SetError(err error) {
	m.err = err
	m.loading = false
}

func (m *BrowserModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *BrowserModel) MoveDown() {
	max := m.itemCount() - 1
	if m.cursor < max {
		m.cursor++
	}
}

func (m *BrowserModel) itemCount() int {
	switch m.mode {
	case BrowseCollections:
		return len(m.collections)
	case BrowseEnvironments:
		return len(m.environments)
	}
	return 0
}

// SelectedCollectionUID returns the UID of the selected collection
func (m *BrowserModel) SelectedCollectionUID() string {
	if m.mode == BrowseCollections && m.cursor < len(m.collections) {
		return m.collections[m.cursor].UID
	}
	return ""
}

// SelectedEnvironmentUID returns the UID of the selected environment
func (m *BrowserModel) SelectedEnvironmentUID() string {
	if m.mode == BrowseEnvironments && m.cursor < len(m.environments) {
		return m.environments[m.cursor].UID
	}
	return ""
}

func (m *BrowserModel) View() string {
	if !m.visible {
		return ""
	}

	var b strings.Builder

	// Title
	var title string
	switch m.mode {
	case BrowseCollections:
		title = " Postman Collections "
	case BrowseEnvironments:
		title = " Postman Environments "
	}

	titleRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#3465a4")).
		Padding(0, 1).
		Render(title)

	b.WriteString(titleRendered + "\n\n")

	if m.loading {
		spinner := lipgloss.NewStyle().Foreground(lipgloss.Color("#3465a4"))
		b.WriteString(spinner.Render("  Loading from Postman Cloud..."))
		return b.String()
	}

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc0000"))
		b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %s", m.err)))
		b.WriteString("\n\n")
		b.WriteString(dimText("  Press Esc to go back"))
		return b.String()
	}

	if m.itemCount() == 0 {
		b.WriteString(dimText("  No items found"))
		b.WriteString("\n\n")
		b.WriteString(dimText("  Press Esc to go back"))
		return b.String()
	}

	// List items
	visibleHeight := m.height - 6
	if visibleHeight < 5 {
		visibleHeight = 5
	}
	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	count := m.itemCount()
	if end > count {
		end = count
	}

	for i := start; i < end; i++ {
		var name string
		switch m.mode {
		case BrowseCollections:
			name = m.collections[i].Name
		case BrowseEnvironments:
			name = m.environments[i].Name
		}

		if i == m.cursor {
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("#3465a4")).
				Foreground(lipgloss.Color("#ffffff")).
				Width(m.width - 6).
				Padding(0, 1)
			b.WriteString("  " + selStyle.Render(fmt.Sprintf("▸ %s", name)))
		} else {
			b.WriteString(fmt.Sprintf("    %s", name))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimText("  Enter=select  Esc=cancel  j/k=navigate"))

	return b.String()
}
