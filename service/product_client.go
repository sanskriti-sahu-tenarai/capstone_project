package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

type ProductPriceResponse struct {
	ProductID int64   `json:"productId"`
	Price     float64 `json:"price"`
}

type ProductClient interface {
	GetProductPrice(ctx context.Context, productID int64) (float64, error)
}

type productClient struct {
	baseURL    string
	httpClient *http.Client
	cb         *gobreaker.CircuitBreaker
}

func NewProductClient(baseURL string) ProductClient {
	cbSettings := gobreaker.Settings{
		Name:        "ProductMicroserviceCircuitBreaker",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	}

	return &productClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		cb: gobreaker.NewCircuitBreaker(cbSettings),
	}
}

func (c *productClient) GetProductPrice(ctx context.Context, productID int64) (float64, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.executeWithRetry(ctx, productID, 3)
	})
	if err != nil {
		return 0.0, fmt.Errorf("circuit breaker open / call failed: %w", err)
	}
	return result.(float64), nil
}

func (c *productClient) executeWithRetry(ctx context.Context, productID int64, maxRetries int) (float64, error) {
	var lastErr error
	url := fmt.Sprintf("%s/api/v1/products/%d/price", c.baseURL, productID)

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return 0, err
		}

		resp, err := c.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var data ProductPriceResponse
			if err := json.Unmarshal(body, &data); err == nil {
				return data.Price, nil
			}
		}

		if resp != nil {
			resp.Body.Close()
		}

		lastErr = err
		backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return 0, fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}
