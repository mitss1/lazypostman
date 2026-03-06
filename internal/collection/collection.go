package collection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Collection represents a Postman Collection v2.1
type Collection struct {
	Info  Info   `json:"info"`
	Items []Item `json:"item"`
}

type Info struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
}

// Item can be either a folder (has Items) or a request (has Request)
type Item struct {
	Name    string   `json:"name"`
	Items   []Item   `json:"item,omitempty"`
	Request *Request `json:"request,omitempty"`
}

// IsFolder returns true if this item contains sub-items
func (i *Item) IsFolder() bool {
	return len(i.Items) > 0
}

type Request struct {
	Method string   `json:"method"`
	URL    URL      `json:"url"`
	Header []Header `json:"header,omitempty"`
	Body   *Body    `json:"body,omitempty"`
	Auth   *Auth    `json:"auth,omitempty"`
}

// URL can be a string or an object in Postman format
type URL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host,omitempty"`
	Path  []string `json:"path,omitempty"`
	Query []Query  `json:"query,omitempty"`
}

func (u *URL) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		u.Raw = s
		return nil
	}
	// Try object
	type urlAlias URL
	var obj urlAlias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*u = URL(obj)
	if u.Raw == "" {
		u.Raw = u.BuildRaw()
	}
	return nil
}

func (u *URL) BuildRaw() string {
	raw := strings.Join(u.Host, ".") + "/" + strings.Join(u.Path, "/")
	if len(u.Query) > 0 {
		params := make([]string, 0, len(u.Query))
		for _, q := range u.Query {
			if !q.Disabled {
				params = append(params, q.Key+"="+q.Value)
			}
		}
		if len(params) > 0 {
			raw += "?" + strings.Join(params, "&")
		}
	}
	return raw
}

type Query struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type Header struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type Body struct {
	Mode       string      `json:"mode"`
	Raw        string      `json:"raw,omitempty"`
	URLEncoded []KeyValue  `json:"urlencoded,omitempty"`
	FormData   []KeyValue  `json:"formdata,omitempty"`
	Options    BodyOptions `json:"options,omitempty"`
}

type BodyOptions struct {
	Raw RawOptions `json:"raw,omitempty"`
}

type RawOptions struct {
	Language string `json:"language,omitempty"`
}

type KeyValue struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type Auth struct {
	Type   string     `json:"type"`
	Bearer []KeyValue `json:"bearer,omitempty"`
	Basic  []KeyValue `json:"basic,omitempty"`
}

// FlatItem represents a flattened tree item for display
type FlatItem struct {
	Item   *Item
	Depth  int
	IsOpen bool
	Index  int // index in flat list
}

// Load reads and parses a Postman collection file
func Load(path string) (*Collection, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading collection: %w", err)
	}

	var col Collection
	if err := json.Unmarshal(data, &col); err != nil {
		return nil, fmt.Errorf("parsing collection: %w", err)
	}

	return &col, nil
}

// Flatten converts the tree structure into a flat list for display
func Flatten(items []Item, depth int, openFolders map[string]bool) []FlatItem {
	var result []FlatItem
	for i := range items {
		item := &items[i]
		path := itemPath(item, depth)
		isOpen := openFolders[path]

		result = append(result, FlatItem{
			Item:   item,
			Depth:  depth,
			IsOpen: isOpen,
		})

		if item.IsFolder() && isOpen {
			children := Flatten(item.Items, depth+1, openFolders)
			result = append(result, children...)
		}
	}
	// Update indices
	for i := range result {
		result[i].Index = i
	}
	return result
}

func itemPath(item *Item, depth int) string {
	return fmt.Sprintf("%d:%s", depth, item.Name)
}

// MethodColor returns a color hex for the HTTP method
func MethodColor(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "#73d216"
	case "POST":
		return "#3465a4"
	case "PUT":
		return "#f57900"
	case "PATCH":
		return "#c4a000"
	case "DELETE":
		return "#cc0000"
	case "HEAD":
		return "#75507b"
	case "OPTIONS":
		return "#888888"
	default:
		return "#ffffff"
	}
}

// MethodShort returns a short padded method string
func MethodShort(method string) string {
	m := strings.ToUpper(method)
	if len(m) < 6 {
		m += strings.Repeat(" ", 6-len(m))
	}
	return m
}
