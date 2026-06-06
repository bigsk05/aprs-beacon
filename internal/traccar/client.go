package traccar

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/goccy/go-json"
)

// Position is the subset of a Traccar /api/positions record we use.
//
// Speed is reported by Traccar in knots, course in degrees, altitude in metres.
type Position struct {
	ID         int       `json:"id"`
	DeviceID   int       `json:"deviceId"`
	ServerTime time.Time `json:"serverTime"`
	DeviceTime time.Time `json:"deviceTime"`
	FixTime    time.Time `json:"fixTime"`
	Valid      bool      `json:"valid"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Altitude   float64   `json:"altitude"`
	Speed      float64   `json:"speed"`
	Course     float64   `json:"course"`
}

// Client queries a Traccar server's REST API for the latest device position.
type Client struct {
	http *http.Client
}

// NewClient builds a Traccar HTTP client with the given per-request timeout.
func NewClient(timeout time.Duration) *Client {
	return &Client{http: &http.Client{Timeout: timeout}}
}

// LatestPosition fetches the most recent position for a device. account and
// password supply HTTP Basic auth. It returns an error if the request fails or
// no position is available.
func (c *Client) LatestPosition(ctx context.Context, baseURL, account, password, deviceID string) (Position, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return Position{}, fmt.Errorf("traccar: bad url %q: %w", baseURL, err)
	}
	endpoint.Path = "/api/positions"
	q := endpoint.Query()
	q.Set("deviceId", deviceID)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Position{}, fmt.Errorf("traccar: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(account + ":" + password))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.http.Do(req)
	if err != nil {
		return Position{}, fmt.Errorf("traccar: request: %w", err)
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Position{}, fmt.Errorf("traccar: status %d: %s", resp.StatusCode, body)
	}

	var positions []Position
	if err := json.NewDecoder(resp.Body).Decode(&positions); err != nil {
		return Position{}, fmt.Errorf("traccar: decode: %w", err)
	}
	if len(positions) == 0 {
		return Position{}, fmt.Errorf("traccar: no position for device %s", deviceID)
	}
	return positions[0], nil
}
