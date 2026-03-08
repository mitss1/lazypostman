package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mitss1/lazypostman/internal/collection"
	"github.com/mitss1/lazypostman/internal/environment"
)

const maxResponseBodySize = 10 * 1024 * 1024 // 10MB

// Response holds the result of an HTTP request
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       string
	Duration   time.Duration
	Size       int64
	Truncated  bool
	Error      error
}

// Client executes HTTP requests
type Client struct {
	httpClient *http.Client
	envManager *environment.Manager
}

func New(envManager *environment.Manager) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		envManager: envManager,
	}
}

// Execute runs a Postman request and returns the response
func (c *Client) Execute(ctx context.Context, req *collection.Request) *Response {
	if req == nil {
		return &Response{Error: fmt.Errorf("nil request")}
	}

	resolvedURL := c.resolve(req.URL.Raw)
	method := strings.ToUpper(req.Method)

	var body io.Reader
	var contentType string

	if req.Body != nil {
		switch req.Body.Mode {
		case "raw":
			body = strings.NewReader(c.resolve(req.Body.Raw))
			if req.Body.Options.Raw.Language == "json" {
				contentType = "application/json"
			}
		case "urlencoded":
			form := url.Values{}
			for _, kv := range req.Body.URLEncoded {
				if !kv.Disabled {
					form.Set(c.resolve(kv.Key), c.resolve(kv.Value))
				}
			}
			body = strings.NewReader(form.Encode())
			contentType = "application/x-www-form-urlencoded"
		case "formdata":
			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			for _, kv := range req.Body.FormData {
				if !kv.Disabled {
					_ = w.WriteField(c.resolve(kv.Key), c.resolve(kv.Value))
				}
			}
			w.Close()
			body = &buf
			contentType = w.FormDataContentType()
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, resolvedURL, body)
	if err != nil {
		return &Response{Error: fmt.Errorf("creating request: %w", err)}
	}

	// Apply headers
	for _, h := range req.Header {
		if !h.Disabled {
			httpReq.Header.Set(c.resolve(h.Key), c.resolve(h.Value))
		}
	}

	if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// Apply auth
	if req.Auth != nil {
		switch req.Auth.Type {
		case "bearer":
			for _, kv := range req.Auth.Bearer {
				if kv.Key == "token" {
					httpReq.Header.Set("Authorization", "Bearer "+c.resolve(kv.Value))
				}
			}
		case "basic":
			var username, password string
			for _, kv := range req.Auth.Basic {
				switch kv.Key {
				case "username":
					username = c.resolve(kv.Value)
				case "password":
					password = c.resolve(kv.Value)
				}
			}
			httpReq.SetBasicAuth(username, password)
		}
	}

	start := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return &Response{Error: err, Duration: duration}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return &Response{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    resp.Header,
			Duration:   duration,
			Error:      fmt.Errorf("reading body: %w", err),
		}
	}

	truncated := int64(len(bodyBytes)) >= maxResponseBodySize

	return &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       string(bodyBytes),
		Duration:   duration,
		Size:       int64(len(bodyBytes)),
		Truncated:  truncated,
	}
}

func (c *Client) resolve(s string) string {
	if c.envManager != nil {
		return c.envManager.Resolve(s)
	}
	return s
}

// StatusColor returns a color for the status code
func StatusColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "#73d216"
	case code >= 300 && code < 400:
		return "#3465a4"
	case code >= 400 && code < 500:
		return "#f57900"
	case code >= 500:
		return "#cc0000"
	default:
		return "#ffffff"
	}
}
