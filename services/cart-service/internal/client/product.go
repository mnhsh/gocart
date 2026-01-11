package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ProductClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

type Product struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	PriceCents int32     `json:"price_cents"`
	Stock      int32     `json:"stock"`
}

func NewProductClient(baseURL string, timeout time.Duration) *ProductClient {
	return &ProductClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *ProductClient) GetProduct(ctx context.Context, productID uuid.UUID) (*Product, bool, error) {
	if productID == uuid.Nil {
		return nil, false, fmt.Errorf("invalid UUID: nil")
	}
	url := fmt.Sprintf("%s/api/products/%s", c.BaseURL, productID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("error getting the response: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("calling product service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var product Product
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, false, fmt.Errorf("decoding response: %w", err)
	}
	return &product, true, nil
}
