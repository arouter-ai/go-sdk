package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
)

func main() {
	baseURL := envOr("AROUTER_BASE_URL", "http://localhost:18080")
	apiKey := envOr("AROUTER_API_KEY", "lr_live_362e5e6b315d4b9e0a63d524e199f78cf0e0a3a502c0f5b6")

	client := arouter.NewClient(baseURL, apiKey)

	fmt.Println("=== ARouter SDK Integration Test ===")
	fmt.Println()

	// Test 1: ARouter via ChatCompletion (OpenAI-compatible)
	fmt.Println("[Test 1] ARouter - ChatCompletion (google/gemini-2.0-flash-001)")
	testChatCompletion(client, "google/gemini-2.0-flash-001")

	// Test 2: ARouter streaming
	fmt.Println("[Test 2] ARouter - Streaming ChatCompletion")
	testStreamingCompletion(client, "google/gemini-2.0-flash-001")

	// Test 3: Direct proxy to ARouter
	fmt.Println("[Test 3] Direct proxy to arouter")
	testProxyRequest(client)

	fmt.Println()
	fmt.Println("=== All tests complete ===")
}

func testChatCompletion(client *arouter.Client, model string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	maxTok := 50
	resp, err := client.ChatCompletion(ctx, &arouter.ChatCompletionRequest{
		Model: model,
		Messages: []arouter.Message{
			{Role: "user", Content: "Say hello in exactly 5 words."},
		},
		MaxTokens: &maxTok,
	})
	if err != nil {
		log.Printf("  FAIL: %v\n\n", err)
		return
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		fmt.Printf("  OK: %s\n", resp.Choices[0].Message.Content)
		if resp.Usage != nil {
			fmt.Printf("  Usage: input=%d output=%d\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		}
	}
	fmt.Println()
}

func testStreamingCompletion(client *arouter.Client, model string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	maxTok := 100
	stream, err := client.ChatCompletionStream(ctx, &arouter.ChatCompletionRequest{
		Model: model,
		Messages: []arouter.Message{
			{Role: "user", Content: "Count from 1 to 5, one number per line."},
		},
		MaxTokens: &maxTok,
	})
	if err != nil {
		log.Printf("  FAIL: %v\n\n", err)
		return
	}
	defer stream.Close()

	fmt.Print("  Streaming: ")
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == arouter.ErrStreamDone || err == io.EOF {
				break
			}
			log.Printf("stream error: %v", err)
			break
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.Content != "" {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
	fmt.Println("  OK")
	fmt.Println()
}

func testProxyRequest(client *arouter.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payload, _ := json.Marshal(map[string]interface{}{
		"model":      "google/gemini-2.0-flash-001",
		"messages": []map[string]string{
			{"role": "user", "content": "What is 2+2? Reply with just the number."},
		},
		"max_tokens": 10,
	})

	resp, err := client.ProxyRequest(ctx, "arouter", "v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		log.Printf("  FAIL: %v\n\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(body, &result) == nil && len(result.Choices) > 0 {
		fmt.Printf("  OK: status=%d, answer=%s\n\n", resp.StatusCode, result.Choices[0].Message.Content)
	} else {
		fmt.Printf("  OK: status=%d, body=%s\n\n", resp.StatusCode, string(body[:min(len(body), 200)]))
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
