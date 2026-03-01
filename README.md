# LLMRouter Go SDK

Official Go client for the [LLMRouter](https://github.com/llmrouter-ai) API gateway — one API key, every LLM provider.

## Installation

```bash
go get github.com/llmrouter/llmrouter-go
```

> Requires Go 1.21+. Zero external dependencies.

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	llmrouter "github.com/llmrouter/llmrouter-go"
)

func main() {
	client := llmrouter.NewClient("https://api.llmrouter.io", "lr_live_xxx")

	resp, err := client.ChatCompletion(context.Background(), &llmrouter.ChatCompletionRequest{
		Model: "openrouter/anthropic/claude-sonnet-4",
		Messages: []llmrouter.Message{
			{Role: "user", Content: "Hello!"},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Choices[0].Message.Content)
}
```

## API Overview

| Category | Methods |
|----------|---------|
| **LLM** | `ChatCompletion`, `ChatCompletionStream`, `ProxyRequest` |
| **Admin** | `CreateSubKey`, `ListSubKeys`, `RevokeSubKey` |

## Streaming

```go
stream, err := client.ChatCompletionStream(ctx, &llmrouter.ChatCompletionRequest{
	Model:    "openrouter/anthropic/claude-sonnet-4",
	Messages: []llmrouter.Message{{Role: "user", Content: "Tell me a story"}},
})
if err != nil {
	log.Fatal(err)
}
defer stream.Close()

for {
	chunk, err := stream.Recv()
	if err == llmrouter.ErrStreamDone {
		break
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(chunk.Choices[0].Delta.Content)
}
```

## Sub-Key Management

Main keys (`lr_live_`) can create scoped sub-keys with provider/model restrictions, rate limits, and quotas — no dashboard login required.

```go
sub, err := client.CreateSubKey(ctx, &llmrouter.CreateSubKeyRequest{
	Name:             "worker-1",
	AllowedProviders: []string{"openrouter"},
	AllowedModels:    []string{"openrouter/anthropic/claude-sonnet-4"},
	RateLimit: &llmrouter.RateLimitConfig{
		RequestsPerMinute: 60,
	},
})
if err != nil {
	log.Fatal(err)
}
fmt.Println("Sub-key:", sub.RawKey) // lr_sub_xxx

// List all sub-keys
keys, _ := client.ListSubKeys(ctx, nil)
for _, k := range keys.Keys {
	fmt.Println(k.ID, k.Name)
}

// Revoke
_ = client.RevokeSubKey(ctx, sub.Key.ID)
```

## Provider Proxy

Forward raw requests to any provider endpoint (embeddings, images, audio, etc.):

```go
body := strings.NewReader(`{"input": "hello", "model": "text-embedding-3-small"}`)
resp, err := client.ProxyRequest(ctx, "openai", "v1/embeddings", body)
if err != nil {
	log.Fatal(err)
}
defer resp.Body.Close()
// read resp.Body ...
```

## Client Options

```go
client := llmrouter.NewClient(baseURL, apiKey,
	llmrouter.WithTimeout(60 * time.Second),
	llmrouter.WithHTTPClient(customHTTPClient),
)
```

## Error Handling

All API errors are returned as `*llmrouter.APIError` and support `errors.Is` matching:

```go
_, err := client.ChatCompletion(ctx, req)
if errors.Is(err, llmrouter.ErrRateLimited) {
	// back off and retry
}
if errors.Is(err, llmrouter.ErrQuotaExceeded) {
	// quota exhausted
}

var apiErr *llmrouter.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Code, apiErr.Message)
}
```

Sentinel errors: `ErrUnauthorized` · `ErrForbidden` · `ErrNotFound` · `ErrRateLimited` · `ErrQuotaExceeded` · `ErrBadRequest` · `ErrServerError`

## License

MIT
