package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mitss1/lazypostman/internal/collection"
)

// TreeModel handles the collection tree navigation
type TreeModel struct {
	items       []collection.FlatItem
	collection  *collection.Collection
	cursor      int
	openFolders map[string]bool
	width       int
	height      int
}

func NewTreeModel(col *collection.Collection) TreeModel {
	open := make(map[string]bool)
	items := collection.Flatten(col.Items, 0, open)
	return TreeModel{
		items:       items,
		collection:  col,
		openFolders: open,
	}
}

func (m *TreeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *TreeModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *TreeModel) MoveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

func (m *TreeModel) Toggle() {
	if m.cursor >= len(m.items) {
		return
	}
	item := m.items[m.cursor]
	if item.Item.IsFolder() {
		m.openFolders[item.Path] = !m.openFolders[item.Path]
		m.rebuild()
	}
}

// Expand opens the folder at cursor (no-op if already open or not a folder)
func (m *TreeModel) Expand() {
	if m.cursor >= len(m.items) {
		return
	}
	item := m.items[m.cursor]
	if item.Item.IsFolder() && !m.openFolders[item.Path] {
		m.openFolders[item.Path] = true
		m.rebuild()
	}
}

// Collapse closes the folder at cursor, or moves to parent folder
func (m *TreeModel) Collapse() {
	if m.cursor >= len(m.items) {
		return
	}
	item := m.items[m.cursor]
	if item.Item.IsFolder() && m.openFolders[item.Path] {
		m.openFolders[item.Path] = false
		m.rebuild()
		return
	}
	// Move to parent folder
	if item.Depth > 0 {
		for i := m.cursor - 1; i >= 0; i-- {
			if m.items[i].Depth < item.Depth && m.items[i].Item.IsFolder() {
				m.cursor = i
				return
			}
		}
	}
}

// MoveToTop moves the cursor to the first item
func (m *TreeModel) MoveToTop() {
	m.cursor = 0
}

// MoveToBottom moves the cursor to the last item
func (m *TreeModel) MoveToBottom() {
	if len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
}

func (m *TreeModel) SelectedItem() *collection.Item {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor].Item
	}
	return nil
}

func (m *TreeModel) SelectedRequest() *collection.Request {
	item := m.SelectedItem()
	if item != nil {
		return item.Request
	}
	return nil
}

func (m *TreeModel) rebuild() {
	m.items = collection.Flatten(m.collection.Items, 0, m.openFolders)
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m *TreeModel) View() string {
	if len(m.items) == 0 {
		return "  No items in collection"
	}

	var b strings.Builder

	// Calculate scroll offset
	visibleHeight := m.height - 2
	if visibleHeight < 1 {
		visibleHeight = 20
	}
	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := start; i < end; i++ {
		fi := m.items[i]
		indent := strings.Repeat("  ", fi.Depth)
		selected := i == m.cursor

		var line string
		if fi.Item.IsFolder() {
			arrow := "▸"
			if fi.IsOpen {
				arrow = "▾"
			}
			folderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c4a000"))
			line = fmt.Sprintf("%s%s %s", indent, arrow, folderStyle.Render(fi.Item.Name))
		} else if fi.Item.Request != nil {
			method := collection.MethodShort(fi.Item.Request.Method)
			color := collection.MethodColor(fi.Item.Request.Method)
			methodStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(color)).
				Bold(true)
			line = fmt.Sprintf("%s  %s %s", indent, methodStyle.Render(method), fi.Item.Name)
		} else {
			line = fmt.Sprintf("%s  %s", indent, fi.Item.Name)
		}

		if selected {
			selStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("#3465a4")).
				Foreground(lipgloss.Color("#ffffff")).
				Width(m.width - 2)
			line = selStyle.Render(line)
		}

		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
