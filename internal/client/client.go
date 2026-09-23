// Copyright (c) OpenMetadata Contributors
// SPDX-License-Identifier: Apache-2.0

// Package client provides an HTTP client for the OpenMetadata REST API.
//
// It handles authentication (JWT token), request/response serialization, and
// retries. All resource CRUD operations go through this client.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client wraps the OpenMetadata REST API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new OpenMetadata API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError represents an error response from the OpenMetadata API.
type APIError struct {
	Code             int    `json:"code"`
	Message          string `json:"message"`
	ResponseHTTPCode int    `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("OpenMetadata API error (HTTP %d): code=%d, message=%s", e.ResponseHTTPCode, e.Code, e.Message)
}

// doRequest executes an HTTP request against the OpenMetadata API.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	u := fmt.Sprintf("%s/api/v1/%s", c.BaseURL, strings.TrimLeft(path, "/"))

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		tflog.Trace(ctx, "API request body", map[string]interface{}{"body": string(jsonBody)})
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}

	tflog.Debug(ctx, "API request", map[string]interface{}{"method": method, "url": u})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	tflog.Trace(ctx, "API response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(respBody),
	})

	if resp.StatusCode >= 400 {
		apiErr := &APIError{ResponseHTTPCode: resp.StatusCode}
		if json.Unmarshal(respBody, apiErr) != nil {
			apiErr.Message = string(respBody)
		}
		return nil, resp.StatusCode, apiErr
	}

	return respBody, resp.StatusCode, nil
}

// CreateOrUpdate performs an idempotent PUT (OM's create_or_update pattern).
func (c *Client) CreateOrUpdate(ctx context.Context, collection string, body interface{}) (json.RawMessage, error) {
	respBody, _, err := c.doRequest(ctx, http.MethodPut, collection, body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(respBody), nil
}

// GetByName retrieves an entity by its fully qualified name.
func (c *Client) GetByName(ctx context.Context, collection, fqn string, fields []string) (json.RawMessage, error) {
	path := fmt.Sprintf("%s/name/%s", collection, url.PathEscape(fqn))
	if len(fields) > 0 {
		path += "?fields=" + strings.Join(fields, ",")
	}
	respBody, statusCode, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		if statusCode == http.StatusNotFound {
			return nil, nil // not found → nil, nil (lets Terraform know to recreate)
		}
		return nil, err
	}
	return json.RawMessage(respBody), nil
}

// GetByID retrieves an entity by its UUID.
func (c *Client) GetByID(ctx context.Context, collection, id string, fields []string) (json.RawMessage, error) {
	path := fmt.Sprintf("%s/%s", collection, id)
	if len(fields) > 0 {
		path += "?fields=" + strings.Join(fields, ",")
	}
	respBody, statusCode, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		if statusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return json.RawMessage(respBody), nil
}

// Delete removes an entity by its UUID. hardDelete=true permanently removes it.
func (c *Client) Delete(ctx context.Context, collection, id string, hardDelete bool) error {
	path := fmt.Sprintf("%s/%s", collection, id)
	if hardDelete {
		path += "?hardDelete=true&recursive=true"
	} else {
		path += "?recursive=true"
	}
	_, _, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// Ping checks API connectivity by hitting /api/v1/system/version.
func (c *Client) Ping(ctx context.Context) error {
	_, _, err := c.doRequest(ctx, http.MethodGet, "system/version", nil)
	return err
}
