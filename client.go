package llmrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client is the LLMRouter SDK client.
//
// Initialize with just a base URL and API key — no tenant ID, JWT, or
// login credentials needed. The server identifies your tenant from the key.
//
//	client := llmrouter.NewClient("https://api.llmrouter.io", "lr_live_xxx")
//
// The client provides two groups of methods:
//
//	Admin:  CreateSubKey, ListSubKeys, RevokeSubKey
//	LLM:    ChatCompletion, ChatCompletionStream, ProxyRequest
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new LLMRouter client.
//
//	baseURL is the root URL of the LLMRouter gateway (e.g. "https://api.llmrouter.io").
//	apiKey  is your API key (lr_live_xxx or lr_sub_xxx).
func NewClient(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// newRequest builds an authenticated *http.Request.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	url := c.baseURL + path

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("llmrouter: marshal request: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// do executes a request and decodes the JSON response into dst.
func (c *Client) do(req *http.Request, dst any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("llmrouter: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseErrorResponse(resp)
	}

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			return fmt.Errorf("llmrouter: decode response: %w", err)
		}
	}
	return nil
}

// parseErrorResponse reads an error response body and returns an *APIError.
func parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	apiErr := &APIError{StatusCode: resp.StatusCode}

	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error.Message != "" {
		apiErr.Code = envelope.Error.Code
		apiErr.Message = envelope.Error.Message
		return apiErr
	}

	// Fallback: try flat structure.
	_ = json.Unmarshal(body, apiErr)
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}
