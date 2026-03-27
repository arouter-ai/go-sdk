package arouter

import (
	"context"
	"fmt"
	"net/http"
)

// ---- Webhook Destination API ----
//
// These methods manage webhook delivery destinations through the Dashboard Gateway.
// Authentication uses a session-scoped API key (dashboard management key).
//
// Base path: /api/webhooks

// CreateDestination creates a new webhook destination.
// The response contains the endpoint details and the signing secret.
// The secret is returned ONCE; store it securely.
func (c *Client) CreateDestination(ctx context.Context, req *CreateDestinationRequest) (*CreateDestinationResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/webhooks/endpoints", req)
	if err != nil {
		return nil, err
	}
	var resp CreateDestinationResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("arouter: CreateDestination: %w", err)
	}
	return &resp, nil
}

// ListDestinations returns all webhook destinations for the authenticated tenant.
func (c *Client) ListDestinations(ctx context.Context) (*ListDestinationsResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/api/webhooks/endpoints", nil)
	if err != nil {
		return nil, err
	}
	var resp ListDestinationsResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("arouter: ListDestinations: %w", err)
	}
	return &resp, nil
}

// GetDestination returns a single webhook destination by its ID.
func (c *Client) GetDestination(ctx context.Context, id string) (*Destination, error) {
	httpReq, err := c.newRequest(ctx, http.MethodGet, "/api/webhooks/endpoints/"+id, nil)
	if err != nil {
		return nil, err
	}
	var dest Destination
	if err := c.do(httpReq, &dest); err != nil {
		return nil, fmt.Errorf("arouter: GetDestination: %w", err)
	}
	return &dest, nil
}

// UpdateDestination updates an existing destination. Only provided fields are changed.
func (c *Client) UpdateDestination(ctx context.Context, id string, req *UpdateDestinationRequest) (*Destination, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPatch, "/api/webhooks/endpoints/"+id, req)
	if err != nil {
		return nil, err
	}
	var dest Destination
	if err := c.do(httpReq, &dest); err != nil {
		return nil, fmt.Errorf("arouter: UpdateDestination: %w", err)
	}
	return &dest, nil
}

// DeleteDestination permanently removes a webhook destination.
func (c *Client) DeleteDestination(ctx context.Context, id string) error {
	httpReq, err := c.newRequest(ctx, http.MethodDelete, "/api/webhooks/endpoints/"+id, nil)
	if err != nil {
		return err
	}
	if err := c.do(httpReq, nil); err != nil {
		return fmt.Errorf("arouter: DeleteDestination: %w", err)
	}
	return nil
}

// RotateDestinationSecret rotates the signing secret for a destination and returns the new secret.
// The previous secret is immediately invalidated.
func (c *Client) RotateDestinationSecret(ctx context.Context, id string) (string, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/webhooks/endpoints/"+id+"/rotate-secret", nil)
	if err != nil {
		return "", err
	}
	var resp struct {
		Secret string `json:"secret"`
	}
	if err := c.do(httpReq, &resp); err != nil {
		return "", fmt.Errorf("arouter: RotateDestinationSecret: %w", err)
	}
	return resp.Secret, nil
}

// TestConnection probes the given URL to verify it is reachable before saving a destination.
// This does NOT require an existing destination — it is used during configuration.
// Note: Svix has no native test-connection API; this is implemented in arouter directly.
func (c *Client) TestConnection(ctx context.Context, req *TestConnectionRequest) (*TestConnectionResponse, error) {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/webhooks/test-connection", req)
	if err != nil {
		return nil, err
	}
	var resp TestConnectionResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, fmt.Errorf("arouter: TestConnection: %w", err)
	}
	return &resp, nil
}

// SendTestEvent dispatches an example event to an existing destination via Svix.
// The destination must already exist (use CreateDestination first).
func (c *Client) SendTestEvent(ctx context.Context, req *SendTestEventRequest) error {
	httpReq, err := c.newRequest(ctx, http.MethodPost, "/api/webhooks/test-event", req)
	if err != nil {
		return err
	}
	if err := c.do(httpReq, nil); err != nil {
		return fmt.Errorf("arouter: SendTestEvent: %w", err)
	}
	return nil
}
