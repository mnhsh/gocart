package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return respBody, nil
}

// Auth

func (c *Client) Register(email, password string) error {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	_, err := c.doRequest("POST", "/api/users", body)
	return err
}

func (c *Client) Login(email, password string) (*LoginResponse, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	respBody, err := c.doRequest("POST", "/api/login", body)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(respBody, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to parse login response: %w", err)
	}

	c.Token = loginResp.Token
	return &loginResp, nil
}

// Products

func (c *Client) GetProducts() ([]Product, error) {
	respBody, err := c.doRequest("GET", "/api/products", nil)
	if err != nil {
		return nil, err
	}

	var products []Product
	if err := json.Unmarshal(respBody, &products); err != nil {
		return nil, fmt.Errorf("failed to parse products: %w", err)
	}

	return products, nil
}

// Cart

func (c *Client) GetCart() (*Cart, error) {
	respBody, err := c.doRequest("GET", "/api/cart", nil)
	if err != nil {
		return nil, err
	}

	var cart Cart
	if err := json.Unmarshal(respBody, &cart); err != nil {
		return nil, fmt.Errorf("failed to parse cart: %w", err)
	}

	return &cart, nil
}

func (c *Client) AddToCart(productID string, quantity int) error {
	body := map[string]interface{}{
		"product_id": productID,
		"quantity":   quantity,
	}
	_, err := c.doRequest("POST", "/api/cart/items", body)
	return err
}

func (c *Client) ClearCart() error {
	_, err := c.doRequest("DELETE", "/api/cart", nil)
	return err
}

// Orders

func (c *Client) CreateOrder() (*Order, error) {
	respBody, err := c.doRequest("POST", "/api/orders", nil)
	if err != nil {
		return nil, err
	}

	var order Order
	if err := json.Unmarshal(respBody, &order); err != nil {
		return nil, fmt.Errorf("failed to parse order: %w", err)
	}

	return &order, nil
}

func (c *Client) GetOrders() ([]Order, error) {
	respBody, err := c.doRequest("GET", "/api/orders", nil)
	if err != nil {
		return nil, err
	}

	var orders []Order
	if err := json.Unmarshal(respBody, &orders); err != nil {
		return nil, fmt.Errorf("failed to parse orders: %w", err)
	}

	return orders, nil
}

func (c *Client) CancelOrder(orderID string) error {
	_, err := c.doRequest("DELETE", "/api/orders/"+orderID, nil)
	return err
}

// Admin - Products

func (c *Client) CreateProduct(name, description string, priceCents, stock int) (*Product, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": description,
		"price_cents": priceCents,
		"stock":       stock,
	}
	respBody, err := c.doRequest("POST", "/admin/products", body)
	if err != nil {
		return nil, err
	}

	var product Product
	if err := json.Unmarshal(respBody, &product); err != nil {
		return nil, fmt.Errorf("failed to parse product: %w", err)
	}

	return &product, nil
}

func (c *Client) DeleteProduct(productID string) error {
	_, err := c.doRequest("DELETE", "/admin/products/"+productID, nil)
	return err
}
