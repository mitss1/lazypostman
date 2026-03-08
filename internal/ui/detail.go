package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mitss1/lazypostman/internal/collection"
	"github.com/mitss1/lazypostman/internal/httpclient"
)

// DetailModel shows request details and response
type DetailModel struct {
	request    *collection.Request
	response   *httpclient.Response
	width      int
	height     int
	tab        int // 0=params, 1=headers, 2=body
	respTab    int // 0=body, 1=headers
	reqScroll  int
	respScroll int
	reqCursor  int // selected param/header index
}

func NewDetailModel() DetailModel {
	return DetailModel{}
}

func (m *DetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *DetailModel) SetRequest(req *collection.Request) {
	m.request = req
	m.tab = 0
	m.reqScroll = 0
	m.reqCursor = 0
}

// ReqCursor returns the currently selected param/header index
func (m *DetailModel) ReqCursor() int {
	return m.reqCursor
}

// MoveReqCursorUp moves the cursor up in the params/headers list
func (m *DetailModel) MoveReqCursorUp() {
	if m.reqCursor > 0 {
		m.reqCursor--
	}
}

// MoveReqCursorDown moves the cursor down in the params/headers list
func (m *DetailModel) MoveReqCursorDown(max int) {
	if m.reqCursor < max-1 {
		m.reqCursor++
	}
}

func (m *DetailModel) SetResponse(resp *httpclient.Response) {
	m.response = resp
	m.respTab = 0
	m.respScroll = 0
}

func (m *DetailModel) ScrollReqUp() {
	if m.reqScroll > 0 {
		m.reqScroll--
	}
}

func (m *DetailModel) ScrollReqDown(contentLines int) {
	maxScroll := contentLines - m.height + 5 // header takes ~5 lines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.reqScroll < maxScroll {
		m.reqScroll++
	}
}

func (m *DetailModel) ScrollRespUp() {
	if m.respScroll > 0 {
		m.respScroll--
	}
}

func (m *DetailModel) ScrollRespDown(contentLines int) {
	maxScroll := contentLines - m.height + 5
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.respScroll < maxScroll {
		m.respScroll++
	}
}

func (m *DetailModel) ResetReqScroll() {
	m.reqScroll = 0
}

func (m *DetailModel) ResetRespScroll() {
	m.respScroll = 0
}

func (m *DetailModel) ScrollReqToBottom() {
	maxScroll := m.ReqContentLines() - m.height + 5
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.reqScroll = maxScroll
}

func (m *DetailModel) ScrollRespToBottom() {
	maxScroll := m.RespContentLines() - m.height + 5
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.respScroll = maxScroll
}

func (m *DetailModel) NextTab() {
	m.tab = (m.tab + 1) % 3
	m.reqCursor = 0
}

func (m *DetailModel) NextRespTab() {
	m.respTab = (m.respTab + 1) % 2
}

func (m *DetailModel) RequestView() string {
	return m.requestContent(true)
}

// ReqContentLines returns the total line count of request content (for scroll bounds)
func (m *DetailModel) ReqContentLines() int {
	return strings.Count(m.RequestViewFull(), "\n") + 1
}

// RequestViewFull returns unclipped request content (used for line counting)
func (m *DetailModel) RequestViewFull() string {
	return m.requestContent(false)
}

func (m *DetailModel) requestContent(clip bool) string {
	if m.request == nil {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		return dim.Render("  Select a request from the tree")
	}

	var b strings.Builder

	// Method and URL
	method := m.request.Method
	color := collection.MethodColor(method)
	methodStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

	b.WriteString(fmt.Sprintf("  %s %s\n\n", methodStyle.Render(method), urlStyle.Render(m.request.URL.Raw)))

	// Tabs
	tabs := []string{"Params", "Headers", "Body"}
	tabLine := "  "
	for i, t := range tabs {
		if i == m.tab {
			tabStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#73d216")).Underline(true)
			tabLine += tabStyle.Render(t)
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			tabLine += dimStyle.Render(t)
		}
		if i < len(tabs)-1 {
			tabLine += "  │  "
		}
	}
	b.WriteString(tabLine + "\n")
	b.WriteString("  " + strings.Repeat("─", m.width-4) + "\n")

	switch m.tab {
	case 0: // Params
		if len(m.request.URL.Query) == 0 {
			b.WriteString(dimText("  No query parameters"))
		} else {
			for i, q := range m.request.URL.Query {
				cursor := "  "
				if i == m.reqCursor {
					cursor = "> "
				}
				if q.Disabled {
					b.WriteString(fmt.Sprintf("%s%s = %s (disabled)\n", cursor, dimText(q.Key), dimText(q.Value)))
				} else {
					b.WriteString(fmt.Sprintf("%s%s = %s\n", cursor, keyText(q.Key), valText(q.Value)))
				}
			}
		}
	case 1: // Headers
		if len(m.request.Header) == 0 {
			b.WriteString(dimText("  No headers"))
		} else {
			for i, h := range m.request.Header {
				cursor := "  "
				if i == m.reqCursor {
					cursor = "> "
				}
				if h.Disabled {
					b.WriteString(fmt.Sprintf("%s%s: %s (disabled)\n", cursor, dimText(h.Key), dimText(h.Value)))
				} else {
					b.WriteString(fmt.Sprintf("%s%s: %s\n", cursor, keyText(h.Key), valText(h.Value)))
				}
			}
		}
	case 2: // Body
		if m.request.Body == nil {
			b.WriteString(dimText("  No body"))
		} else {
			b.WriteString(fmt.Sprintf("  Mode: %s\n\n", valText(m.request.Body.Mode)))
			switch m.request.Body.Mode {
			case "raw":
				b.WriteString(formatBody(m.request.Body.Raw, m.width))
			case "urlencoded":
				for _, kv := range m.request.Body.URLEncoded {
					b.WriteString(fmt.Sprintf("  %s = %s\n", keyText(kv.Key), valText(kv.Value)))
				}
			case "formdata":
				for _, kv := range m.request.Body.FormData {
					b.WriteString(fmt.Sprintf("  %s = %s\n", keyText(kv.Key), valText(kv.Value)))
				}
			}
		}
	}

	if clip {
		return clipLines(b.String(), m.reqScroll, m.height-5)
	}
	return b.String()
}

func (m *DetailModel) ResponseView() string {
	if m.response == nil {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		return dim.Render("  Press Enter to send request")
	}

	var b strings.Builder

	if m.response.Error != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc0000")).Bold(true)
		b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %s", m.response.Error)))
		return b.String()
	}

	// Status line
	statusColor := httpclient.StatusColor(m.response.StatusCode)
	statusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(statusColor))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	b.WriteString(fmt.Sprintf("  %s  %s  %s\n\n",
		statusStyle.Render(m.response.Status),
		timeStyle.Render(fmt.Sprintf("%dms", m.response.Duration.Milliseconds())),
		sizeStyle.Render(formatSize(m.response.Size)),
	))

	// Response tabs
	tabs := []string{"Body", "Headers"}
	tabLine := "  "
	for i, t := range tabs {
		if i == m.respTab {
			tabStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#73d216")).Underline(true)
			tabLine += tabStyle.Render(t)
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			tabLine += dimStyle.Render(t)
		}
		if i < len(tabs)-1 {
			tabLine += "  │  "
		}
	}
	b.WriteString(tabLine + "\n")
	b.WriteString("  " + strings.Repeat("─", m.width-4) + "\n")

	switch m.respTab {
	case 0: // Body
		b.WriteString(formatBody(m.response.Body, m.width))
	case 1: // Headers
		keys := make([]string, 0, len(m.response.Headers))
		for k := range m.response.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			for _, v := range m.response.Headers[k] {
				b.WriteString(fmt.Sprintf("  %s: %s\n", keyText(k), valText(v)))
			}
		}
	}

	return clipLines(b.String(), m.respScroll, m.height-5)
}

// RespContentLines returns total line count of response content
func (m *DetailModel) RespContentLines() int {
	if m.response == nil {
		return 0
	}
	if m.response.Error != nil {
		return 1
	}
	switch m.respTab {
	case 0:
		return strings.Count(m.response.Body, "\n") + 10
	case 1:
		count := 0
		for _, vals := range m.response.Headers {
			count += len(vals)
		}
		return count + 5
	}
	return 0
}

func formatBody(body string, width int) string {
	if body == "" {
		return dimText("  (empty)")
	}

	// Try to pretty-print JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(body), &parsed); err == nil {
		pretty, err := json.MarshalIndent(parsed, "  ", "  ")
		if err == nil {
			return "  " + string(pretty)
		}
	}

	// Return raw body with indentation
	lines := strings.Split(body, "\n")
	var result strings.Builder
	for _, line := range lines {
		result.WriteString("  " + line + "\n")
	}
	return result.String()
}

func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}

func keyText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#c4a000")).Render(s)
}

func valText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Render(s)
}

func dimText(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(s)
}

// clipLines takes rendered content, applies scroll offset, and clips to maxLines
func clipLines(content string, scroll, maxLines int) string {
	if maxLines <= 0 {
		maxLines = 10
	}
	lines := strings.Split(content, "\n")
	if scroll > len(lines) {
		scroll = len(lines)
	}
	if scroll > 0 {
		lines = lines[scroll:]
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	result := strings.Join(lines, "\n")

	// Add scroll indicator
	totalLines := scroll + len(lines)
	if scroll > 0 || totalLines < scroll+maxLines+1 {
		// Show position indicator if scrolled
		if scroll > 0 {
			indicator := dimText(fmt.Sprintf("  ↑ %d more above", scroll))
			result = indicator + "\n" + result
			// Remove one line from bottom to keep height
			clipped := strings.Split(result, "\n")
			if len(clipped) > maxLines {
				clipped = clipped[:maxLines]
			}
			result = strings.Join(clipped, "\n")
		}
	}

	return result
}
