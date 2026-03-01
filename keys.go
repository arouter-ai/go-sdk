package llmrouter

import (
	"context"
	"fmt"
	"net/http"
)

// --- Admin: Key Management ---
//
// These methods manage subkeys under the current API key.
// Only main keys (lr_live_) can create/list/revoke subkeys.
// Subkeys (lr_sub_) cannot create further subkeys.

// CreateSubKey creates a sub-key under the current API key.
//
//	sub, err := client.CreateSubKey(ctx, &llmrouter.CreateSubKeyRequest{
//	    Name: "my-service",
//	    AllowedProviders: []string{"arouter"},
//	})
//	fmt.Println(sub.RawKey) // lr_sub_xxx — use this to make LLM calls
func (c *Client) CreateSubKey(ctx context.Context, req *CreateSubKeyRequest) (*CreateSubKeyResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/v1/keys/subkeys", req)
	if err != nil {
		return nil, err
	}

	var resp CreateSubKeyResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListSubKeys lists sub-keys belonging to the current API key.
func (c *Client) ListSubKeys(ctx context.Context, opts *ListKeysOptions) (*ListKeysResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/v1/keys/subkeys", nil)
	if err != nil {
		return nil, err
	}

	if opts != nil {
		q := httpReq.URL.Query()
		if opts.PageSize > 0 {
			q.Set("page_size", fmt.Sprintf("%d", opts.PageSize))
		}
		if opts.PageToken != "" {
			q.Set("page_token", opts.PageToken)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	var resp ListKeysResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeSubKey revokes a sub-key by its ID. Only the parent key can revoke.
func (c *Client) RevokeSubKey(ctx context.Context, subKeyID string) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/v1/keys/subkeys/"+subKeyID, nil)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}
