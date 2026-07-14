
// Created by saifll (aka renagge39)
package headroom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	headroomAPIURLDefault = "http://localhost:8787/v1/compress" // Default, bisa di-override
)

// CompressRequest adalah struktur untuk payload kompresi Headroom
type CompressRequest struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model,omitempty"`
}

// Message adalah struktur pesan dalam request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompressResponse adalah struktur untuk respons kompresi Headroom
type CompressResponse struct {
	Messages     []Message `json:"messages"`
	TokensBefore int       `json:"tokens_before"`
	TokensAfter  int       `json:"tokens_after"`
}

// Client adalah klien untuk berinteraksi dengan Headroom API
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient membuat instance baru dari Headroom client
func NewClient(apiKey string) *Client {
	baseURL := os.Getenv("HEADROOM_API_URL")
	if baseURL == "" {
		baseURL = headroomAPIURLDefault
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Compress mengirimkan data untuk kompresi ke Headroom API
func (c *Client) Compress(requestBody CompressRequest) (*CompressResponse, error) {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim request ke headroom: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("headroom API mengembalikan status error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var compressResp CompressResponse
	if err := json.NewDecoder(resp.Body).Decode(&compressResp); err != nil {
		return nil, fmt.Errorf("gagal decode response body: %w", err)
	}

	return &compressResp, nil
}
