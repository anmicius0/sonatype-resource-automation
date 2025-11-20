package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/anmicius0/sonatype-resource-automation/internal/utils"
	"go.uber.org/zap"
	"resty.dev/v3"
)

// HTTPClient is a base HTTP client using resty for API requests.
type HTTPClient struct {
	client *resty.Client
}

// HTTPError represents an HTTP error response from the remote API.
// It exposes the status code so callers can detect specific cases (e.g., 404)
// without parsing text messages.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// NewHTTPClient creates a new HTTPClient with basic auth and JSON headers.
func NewHTTPClient(baseURL, username, password string) *HTTPClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &HTTPClient{
		client: resty.New().
			SetBaseURL(baseURL).
			SetHeader("Accept", "application/json").
			SetHeader("Content-Type", "application/json").
			SetBasicAuth(username, password).
			SetTimeout(30 * time.Second),
	}
}

// DoReq performs an HTTP request with the given method, endpoint, body, and query params.
// Logs errors for 4xx/5xx responses and truncates long bodies.
func (c *HTTPClient) DoReq(method, endpoint string, body any, params map[string]string) (*resty.Response, error) {
	request := c.client.R().
		SetBody(body).
		SetQueryParams(params)

	utils.Logger.Debug("HTTP request start",
		zap.String("method", method),
		zap.String("endpoint", endpoint))

	start := time.Now()
	response, err := request.Execute(method, endpoint)
	duration := time.Since(start)
	if err != nil {
		utils.Logger.Error("HTTP request failed",
			zap.String("method", method),
			zap.String("endpoint", endpoint),
			zap.Error(err))
		return nil, err
	}

	// When status >= 400, log differently for 404 (common existence check) vs other errors
	if response.StatusCode() >= 400 {
		responseBody := strings.TrimSpace(response.String())
		if len(responseBody) > 1000 {
			responseBody = responseBody[:1000] + "…"
		}
		if response.StatusCode() == 404 {
			// 404 is often used to detect non-existence; quieter debug-level log to reduce noise
			utils.Logger.Debug("API returned 404 (resource not found)",
				zap.String("method", method),
				zap.String("url", response.Request.URL),
				zap.Int("status_code", response.StatusCode()),
				zap.String("body", responseBody),
				zap.Duration("duration", duration))
		} else if response.StatusCode() >= 500 {
			// Server errors are noteworthy
			utils.Logger.Error("API error response (server)",
				zap.String("method", method),
				zap.String("url", response.Request.URL),
				zap.Int("status_code", response.StatusCode()),
				zap.String("body", responseBody),
				zap.Duration("duration", duration))
		} else {
			// Client errors (other than 404) are warnings
			utils.Logger.Warn("API error response (client)",
				zap.String("method", method),
				zap.String("url", response.Request.URL),
				zap.Int("status_code", response.StatusCode()),
				zap.String("body", responseBody),
				zap.Duration("duration", duration))
		}
		return nil, &HTTPError{StatusCode: response.StatusCode(), Body: responseBody}
	}

	utils.Logger.Debug("HTTP request completed",
		zap.String("method", method),
		zap.String("url", response.Request.URL),
		zap.Int("status_code", response.StatusCode()),
		zap.Duration("duration", duration))

	return response, nil
}
