package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temp collection file
	col := `{
		"info": {"name": "Test Collection", "description": "desc"},
		"item": [
			{"name": "Get Users", "request": {"method": "GET", "url": "https://api.example.com/users"}},
			{"name": "Folder", "item": [
				{"name": "Nested", "request": {"method": "POST", "url": "https://api.example.com/nested"}}
			]}
		]
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "test.postman_collection.json")
	if err := os.WriteFile(path, []byte(col), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if c.Info.Name != "Test Collection" {
		t.Errorf("expected name 'Test Collection', got %q", c.Info.Name)
	}
	if len(c.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(c.Items))
	}
	if !c.Items[1].IsFolder() {
		t.Error("expected second item to be a folder")
	}
	if c.Items[0].IsFolder() {
		t.Error("expected first item to not be a folder")
	}
}

func TestLoadInvalidPath(t *testing.T) {
	_, err := Load("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFlatten(t *testing.T) {
	items := []Item{
		{Name: "Request1", Request: &Request{Method: "GET", URL: URL{Raw: "http://a"}}},
		{Name: "Folder1", Items: []Item{
			{Name: "Nested1", Request: &Request{Method: "POST", URL: URL{Raw: "http://b"}}},
			{Name: "Nested2", Request: &Request{Method: "PUT", URL: URL{Raw: "http://c"}}},
		}},
		{Name: "Request2", Request: &Request{Method: "DELETE", URL: URL{Raw: "http://d"}}},
	}

	// Closed folders
	open := map[string]bool{}
	flat := Flatten(items, 0, open)
	if len(flat) != 3 {
		t.Errorf("closed: expected 3 items, got %d", len(flat))
	}

	// Open the folder
	open["Folder1"] = true
	flat = Flatten(items, 0, open)
	if len(flat) != 5 {
		t.Errorf("open: expected 5 items, got %d", len(flat))
	}

	// Verify depths
	if flat[0].Depth != 0 {
		t.Errorf("item 0 depth: expected 0, got %d", flat[0].Depth)
	}
	if flat[2].Depth != 1 {
		t.Errorf("item 2 depth: expected 1, got %d", flat[2].Depth)
	}

	// Verify indices
	for i, fi := range flat {
		if fi.Index != i {
			t.Errorf("item %d: expected Index=%d, got %d", i, i, fi.Index)
		}
	}
}

func TestItemPathUniqueness(t *testing.T) {
	// Two folders with the same name at the same depth but different parents
	items := []Item{
		{Name: "Parent1", Items: []Item{
			{Name: "Child", Items: []Item{
				{Name: "Req", Request: &Request{Method: "GET", URL: URL{Raw: "http://a"}}},
			}},
		}},
		{Name: "Parent2", Items: []Item{
			{Name: "Child", Items: []Item{
				{Name: "Req", Request: &Request{Method: "GET", URL: URL{Raw: "http://b"}}},
			}},
		}},
	}

	// Open all folders
	open := map[string]bool{
		"Parent1":              true,
		"Parent1/Child":        true,
		"Parent2":              true,
		"Parent2/Child":        true,
	}
	flat := Flatten(items, 0, open)

	// Collect all paths
	paths := map[string]bool{}
	for _, fi := range flat {
		if fi.Path != "" {
			if paths[fi.Path] {
				t.Errorf("duplicate path: %s", fi.Path)
			}
			paths[fi.Path] = true
		}
	}

	// The two "Child" folders should have different paths
	if !paths["Parent1/Child"] {
		t.Error("expected path Parent1/Child")
	}
	if !paths["Parent2/Child"] {
		t.Error("expected path Parent2/Child")
	}
}

func TestURLUnmarshalJSON_String(t *testing.T) {
	data := `"https://api.example.com/users"`
	var u URL
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if u.Raw != "https://api.example.com/users" {
		t.Errorf("expected raw URL, got %q", u.Raw)
	}
}

func TestURLUnmarshalJSON_Object(t *testing.T) {
	data := `{
		"raw": "https://api.example.com/users?page=1",
		"host": ["api", "example", "com"],
		"path": ["users"],
		"query": [{"key": "page", "value": "1"}]
	}`
	var u URL
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if u.Raw != "https://api.example.com/users?page=1" {
		t.Errorf("expected raw URL, got %q", u.Raw)
	}
	if len(u.Query) != 1 {
		t.Fatalf("expected 1 query param, got %d", len(u.Query))
	}
	if u.Query[0].Key != "page" || u.Query[0].Value != "1" {
		t.Errorf("unexpected query param: %+v", u.Query[0])
	}
}

func TestURLUnmarshalJSON_ObjectNoRaw(t *testing.T) {
	data := `{
		"host": ["api", "example", "com"],
		"path": ["users"],
		"query": [{"key": "page", "value": "1"}]
	}`
	var u URL
	if err := json.Unmarshal([]byte(data), &u); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if u.Raw == "" {
		t.Error("expected BuildRaw to populate Raw")
	}
}

func TestBuildRaw(t *testing.T) {
	u := URL{
		Host:  []string{"api", "example", "com"},
		Path:  []string{"v1", "users"},
		Query: []Query{{Key: "page", Value: "1"}, {Key: "limit", Value: "10"}},
	}
	raw := u.BuildRaw()
	expected := "api.example.com/v1/users?page=1&limit=10"
	if raw != expected {
		t.Errorf("expected %q, got %q", expected, raw)
	}
}

func TestBuildRawDisabledQuery(t *testing.T) {
	u := URL{
		Host:  []string{"api", "example", "com"},
		Path:  []string{"users"},
		Query: []Query{{Key: "page", Value: "1", Disabled: true}},
	}
	raw := u.BuildRaw()
	expected := "api.example.com/users"
	if raw != expected {
		t.Errorf("expected %q, got %q", expected, raw)
	}
}

func TestMethodColor(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"GET", "#73d216"},
		{"POST", "#3465a4"},
		{"PUT", "#f57900"},
		{"DELETE", "#cc0000"},
		{"UNKNOWN", "#ffffff"},
	}
	for _, tt := range tests {
		got := MethodColor(tt.method)
		if got != tt.want {
			t.Errorf("MethodColor(%q) = %q, want %q", tt.method, got, tt.want)
		}
	}
}

func TestMethodShort(t *testing.T) {
	if got := MethodShort("GET"); len(got) != 6 {
		t.Errorf("expected padded to 6 chars, got %d: %q", len(got), got)
	}
	if got := MethodShort("DELETE"); got != "DELETE" {
		t.Errorf("expected DELETE, got %q", got)
	}
}
