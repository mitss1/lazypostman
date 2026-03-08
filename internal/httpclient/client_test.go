package httpclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mitss1/lazypostman/internal/collection"
	"github.com/mitss1/lazypostman/internal/environment"
)

func TestExecuteGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "GET",
		URL:    collection.URL{Raw: server.URL + "/test"},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Body != `{"ok":true}` {
		t.Errorf("unexpected body: %q", resp.Body)
	}
}

func TestExecuteNilRequest(t *testing.T) {
	client := New(environment.NewManager())
	resp := client.Execute(context.Background(), nil)
	if resp.Error == nil {
		t.Error("expected error for nil request")
	}
}

func TestExecuteFormData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data content type, got %q", ct)
		}
		if !strings.Contains(ct, "boundary=") {
			t.Errorf("expected boundary in content type, got %q", ct)
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("ParseMultipartForm error: %v", err)
		}

		if got := r.FormValue("name"); got != "test" {
			t.Errorf("expected form value name=test, got %q", got)
		}
		if got := r.FormValue("value"); got != "hello" {
			t.Errorf("expected form value value=hello, got %q", got)
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "POST",
		URL:    collection.URL{Raw: server.URL},
		Body: &collection.Body{
			Mode: "formdata",
			FormData: []collection.KeyValue{
				{Key: "name", Value: "test"},
				{Key: "value", Value: "hello"},
				{Key: "disabled", Value: "skip", Disabled: true},
			},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExecuteURLEncoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected url-encoded content type, got %q", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "key=value") {
			t.Errorf("expected key=value in body, got %q", string(body))
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "POST",
		URL:    collection.URL{Raw: server.URL},
		Body: &collection.Body{
			Mode: "urlencoded",
			URLEncoded: []collection.KeyValue{
				{Key: "key", Value: "value"},
			},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestExecuteRawJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected application/json, got %q", ct)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"test":true}` {
			t.Errorf("unexpected body: %q", string(body))
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "POST",
		URL:    collection.URL{Raw: server.URL},
		Body: &collection.Body{
			Mode: "raw",
			Raw:  `{"test":true}`,
			Options: collection.BodyOptions{
				Raw: collection.RawOptions{Language: "json"},
			},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestExecuteBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-token" {
			t.Errorf("expected 'Bearer my-token', got %q", auth)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "GET",
		URL:    collection.URL{Raw: server.URL},
		Auth: &collection.Auth{
			Type: "bearer",
			Bearer: []collection.KeyValue{
				{Key: "token", Value: "my-token"},
			},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestExecuteBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "admin" || pass != "secret" {
			t.Errorf("expected admin:secret, got %s:%s", user, pass)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "GET",
		URL:    collection.URL{Raw: server.URL},
		Auth: &collection.Auth{
			Type: "basic",
			Basic: []collection.KeyValue{
				{Key: "username", Value: "admin"},
				{Key: "password", Value: "secret"},
			},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestExecuteCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom"); got != "hello" {
			t.Errorf("expected X-Custom=hello, got %q", got)
		}
		// Disabled header should not be present
		if got := r.Header.Get("X-Disabled"); got != "" {
			t.Errorf("expected no X-Disabled header, got %q", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := New(environment.NewManager())
	req := &collection.Request{
		Method: "GET",
		URL:    collection.URL{Raw: server.URL},
		Header: []collection.Header{
			{Key: "X-Custom", Value: "hello"},
			{Key: "X-Disabled", Value: "nope", Disabled: true},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestExecuteWithEnvResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer resolved-token" {
			t.Errorf("expected resolved token, got %q", got)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	mgr := environment.NewManager()
	mgr.AddEnvironment(&environment.Environment{
		Name: "test",
		Values: []environment.Variable{
			{Key: "token", Value: "resolved-token", Enabled: true},
		},
	})

	client := New(mgr)
	req := &collection.Request{
		Method: "GET",
		URL:    collection.URL{Raw: server.URL},
		Auth: &collection.Auth{
			Type: "bearer",
			Bearer: []collection.KeyValue{
				{Key: "token", Value: "{{token}}"},
			},
		},
	}

	resp := client.Execute(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "#73d216"},
		{301, "#3465a4"},
		{404, "#f57900"},
		{500, "#cc0000"},
		{0, "#ffffff"},
	}
	for _, tt := range tests {
		got := StatusColor(tt.code)
		if got != tt.want {
			t.Errorf("StatusColor(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
