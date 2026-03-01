package llmrouter

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// --- LLM: Chat Completions ---

// ChatCompletion sends a non-streaming chat completion request.
//
//	resp, err := client.ChatCompletion(ctx, &llmrouter.ChatCompletionRequest{
//	    Model:    "anthropic/claude-sonnet-4",
//	    Messages: []llmrouter.Message{{Role: "user", Content: "Hello"}},
//	})
func (c *Client) ChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	req.Stream = false

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/chat/completions", req)
	if err != nil {
		return nil, err
	}

	var resp ChatCompletionResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChatCompletionStream sends a streaming chat completion request and returns a
// stream reader. The caller must call Close on the returned stream when done.
//
//	stream, err := client.ChatCompletionStream(ctx, &llmrouter.ChatCompletionRequest{
//	    Model:    "anthropic/claude-sonnet-4",
//	    Messages: []llmrouter.Message{{Role: "user", Content: "Hello"}},
//	})
//	defer stream.Close()
//	for { chunk, err := stream.Recv(); ... }
func (c *Client) ChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionStream, error) {
	req.Stream = true

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/chat/completions", req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llmrouter: request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, parseErrorResponse(resp)
	}

	return newChatCompletionStream(resp), nil
}

// ProxyRequest sends a raw request through the LLM provider proxy.
// The path is appended to /{provider}/, e.g. ProxyRequest(ctx, "openai", "v1/embeddings", body)
// hits /openai/v1/embeddings. The caller is responsible for closing the
// returned response body.
func (c *Client) ProxyRequest(ctx context.Context, provider, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, provider, path)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llmrouter: request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, parseErrorResponse(resp)
	}

	return resp, nil
}
