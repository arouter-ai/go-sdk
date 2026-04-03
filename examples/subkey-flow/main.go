package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	arouter "github.com/arouter-ai/arouter-go"
)

var (
	gatewayURL = envOr("AROUTER_BASE_URL", "http://localhost:18080")
	mgmtKey    = envOr("AROUTER_MGMT_KEY", "lr_mgmt_61d1517a55a7c664a8717300ec6cddb0460aee2a502b17f5")
)

var models = []string{
	"anthropic/claude-sonnet-4.6",
	"google/gemini-2.5-flash",
	"openai/gpt-5.4",
	"deepseek/deepseek-v3.2",
}

func main() {
	ctx := context.Background()

	fmt.Println("============================================================")
	fmt.Println("ARouter Go SDK - Key Management Flow")
	fmt.Printf("Gateway:  %s\n", gatewayURL)
	fmt.Printf("Mgmt Key: %s...\n", mgmtKey[:24])
	fmt.Println("============================================================")

	mgmtClient := arouter.NewClient(gatewayURL, mgmtKey)

	// Step 1: Create a regular API key via management key
	fmt.Println("\n[Step 1] Create regular API key via management key")
	limit := 150.0
	createResp, err := mgmtClient.CreateKey(ctx, &arouter.CreateKeyRequest{
		Name:             fmt.Sprintf("GoSDK-%d", time.Now().Unix()),
		Limit:            &limit,
		LimitReset:       "monthly",
		AllowedProviders: []string{"anthropic", "google", "openai"},
	})
	if err != nil {
		fmt.Printf("  FAIL: %v\n", err)
		os.Exit(1)
	}
	regularKey := createResp.Key
	fmt.Printf("  Hash:  %s\n", createResp.Data.Hash)
	fmt.Printf("  Name:  %s\n", createResp.Data.Name)
	fmt.Printf("  Key:   %s...\n", regularKey[:24])
	fmt.Printf("  Type:  %s\n", createResp.Data.KeyType)

	// Step 2: List keys to verify
	fmt.Println("\n[Step 2] List Keys")
	listResp, err := mgmtClient.ListKeys(ctx, nil)
	if err != nil {
		fmt.Printf("  FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Found %d key(s):\n", len(listResp.Data))
	for _, k := range listResp.Data {
		fmt.Printf("    - %s  %s  %q  [%s]\n", k.Hash[:16], k.KeyType, k.Name, boolStr(k.Disabled, "disabled", "active"))
	}

	// Step 3: Use the regular key to call LLM models
	fmt.Println("\n[Step 3] Test LLM calls with the new regular API key")
	fmt.Println("============================================================")

	llmClient := arouter.NewClient(gatewayURL, regularKey)
	passCount, totalCount := 0, 0

	for _, model := range models {
		totalCount++
		if testChat(ctx, llmClient, model) {
			passCount++
		}
	}

	totalCount++
	if testStream(ctx, llmClient, models[0]) {
		passCount++
	}

	// Summary
	fmt.Println("\n============================================================")
	fmt.Println("Summary")
	fmt.Println("============================================================")
	fmt.Printf("  Key:    %s (%s)\n", createResp.Data.Name, createResp.Data.KeyType)
	fmt.Printf("  Tests:  %d/%d passed\n", passCount, totalCount)
	if passCount == totalCount {
		fmt.Println("\n  All done. Management key creates regular keys, regular keys call models.")
	} else {
		fmt.Println("\n  Some tests failed.")
		os.Exit(1)
	}
}

func boolStr(b bool, ifTrue, ifFalse string) string {
	if b {
		return ifTrue
	}
	return ifFalse
}

func testChat(ctx context.Context, client *arouter.Client, model string) bool {
	fmt.Printf("\n  [Chat] %s\n", model)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	maxTok := 50
	start := time.Now()
	resp, err := client.ChatCompletion(ctx, &arouter.ChatCompletionRequest{
		Model:     model,
		Messages:  []arouter.Message{{Role: "user", Content: "Say hello in exactly 5 words."}},
		MaxTokens: &maxTok,
	})
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf("    FAIL (%.2fs): %v\n", elapsed.Seconds(), err)
		return false
	}
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		fmt.Printf("    Response: %s\n", resp.Choices[0].Message.Content)
	}
	if resp.Usage != nil {
		fmt.Printf("    Tokens:   in=%d out=%d\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	}
	fmt.Printf("    Latency:  %.2fs  OK\n", elapsed.Seconds())
	return true
}

func testStream(ctx context.Context, client *arouter.Client, model string) bool {
	fmt.Printf("\n  [Stream] %s\n", model)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	maxTok := 100
	start := time.Now()
	stream, err := client.ChatCompletionStream(ctx, &arouter.ChatCompletionRequest{
		Model:     model,
		Messages:  []arouter.Message{{Role: "user", Content: "Count from 1 to 5, one number per line."}},
		MaxTokens: &maxTok,
	})
	if err != nil {
		fmt.Printf("    FAIL (%.2fs): %v\n", time.Since(start).Seconds(), err)
		return false
	}
	defer stream.Close()

	fmt.Print("    Stream: ")
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == arouter.ErrStreamDone || err == io.EOF {
				break
			}
			fmt.Printf("\n    Stream error: %v\n", err)
			return false
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.Content != "" {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Printf("\n    Latency:  %.2fs  OK\n", time.Since(start).Seconds())
	return true
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
