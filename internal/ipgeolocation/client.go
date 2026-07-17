package ipgeolocation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const defaultEndpoint = "https://api.ipgeolocation.io"

// Client is a reusable, concurrency-safe HTTP client for the IPGeolocation.io v3 API.
type Client struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
	limiter    *rate.Limiter
	retries    int
	userAgent  string
}

// Option configures a Client.
type Option func(*Client)

// WithEndpoint overrides the default API base URL.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) { c.endpoint = strings.TrimRight(endpoint, "/") }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithHTTPClient replaces the default HTTP client entirely (useful for tests).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// WithRetries sets the maximum number of retry attempts for transient errors.
func WithRetries(n int) Option {
	return func(c *Client) { c.retries = n }
}

// WithUserAgent sets the User-Agent header on every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// WithRateLimit sets the maximum requests per second.
func WithRateLimit(rps int) Option {
	return func(c *Client) {
		c.limiter = rate.NewLimiter(rate.Limit(rps), rps)
	}
}

// NewClient creates a new IPGeolocation API client.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("ipgeolocation: api key is required")
	}

	c := &Client{
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter:   rate.NewLimiter(15, 15),
		retries:   3,
		userAgent: "cq-source-ipgeolocation/1.0",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// doRequest builds, sends, and decodes an API request with retries and rate limiting.
func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values, body io.Reader, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("apiKey", c.apiKey)

	reqURL := fmt.Sprintf("%s%s?%s", c.endpoint, path, params.Encode())

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return fmt.Errorf("request cancelled: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("executing request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := &APIError{StatusCode: resp.StatusCode}
			_ = json.Unmarshal(respBody, apiErr)
			if apiErr.Message == "" {
				apiErr.Message = http.StatusText(resp.StatusCode)
			}
			if apiErr.IsRetryable() {
				lastErr = apiErr
				continue
			}
			return apiErr
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("all %d retries exhausted: %w", c.retries+1, lastErr)
}

// GetIPGeolocation looks up geolocation data for a single IP address.
// If ip is empty, the API returns data for the caller's own public IP.
func (c *Client) GetIPGeolocation(ctx context.Context, ip string, includes ...string) (*IPGeolocation, error) {
	params := url.Values{}
	if ip != "" {
		params.Set("ip", ip)
	}
	if len(includes) > 0 {
		params.Set("include", strings.Join(includes, ","))
	}

	var result IPGeolocation
	if err := c.doRequest(ctx, http.MethodGet, "/v3/ipgeo", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get ip geolocation for %q: %w", ip, err)
	}
	return &result, nil
}

// GetIPGeolocationBulk looks up geolocation data for multiple IP addresses in a single request.
func (c *Client) GetIPGeolocationBulk(ctx context.Context, ips []string, includes ...string) ([]IPGeolocation, error) {
	params := url.Values{}
	if len(includes) > 0 {
		params.Set("include", strings.Join(includes, ","))
	}

	payload := struct {
		IPs []string `json:"ips"`
	}{IPs: ips}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling bulk request: %w", err)
	}

	var result []IPGeolocation
	if err := c.doRequest(ctx, http.MethodPost, "/v3/ipgeo-bulk", params, strings.NewReader(string(bodyBytes)), &result); err != nil {
		return nil, fmt.Errorf("get ip geolocation bulk: %w", err)
	}
	return result, nil
}

// GetTimezone looks up timezone data by IP address.
func (c *Client) GetTimezone(ctx context.Context, ip string) (*TimezoneResponse, error) {
	params := url.Values{}
	if ip != "" {
		params.Set("ip", ip)
	}

	var result TimezoneResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/timezone", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get timezone for %q: %w", ip, err)
	}
	return &result, nil
}

// GetTimezoneByLocation looks up timezone data by location string or coordinates.
func (c *Client) GetTimezoneByLocation(ctx context.Context, location string) (*TimezoneResponse, error) {
	params := url.Values{}
	params.Set("location", location)

	var result TimezoneResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/timezone", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get timezone for location %q: %w", location, err)
	}
	return &result, nil
}

// GetTimezoneByName looks up timezone data by IANA timezone name.
func (c *Client) GetTimezoneByName(ctx context.Context, tz string) (*TimezoneResponse, error) {
	params := url.Values{}
	params.Set("tz", tz)

	var result TimezoneResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/timezone", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get timezone for tz %q: %w", tz, err)
	}
	return &result, nil
}

// GetAstronomy looks up astronomical data for an IP, location, or coordinates.
func (c *Client) GetAstronomy(ctx context.Context, ip string) (*AstronomyResponse, error) {
	params := url.Values{}
	if ip != "" {
		params.Set("ip", ip)
	}

	var result AstronomyResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/astronomy", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get astronomy for %q: %w", ip, err)
	}
	return &result, nil
}

// GetAstronomyByLocation looks up astronomical data for a location string.
func (c *Client) GetAstronomyByLocation(ctx context.Context, location string) (*AstronomyResponse, error) {
	params := url.Values{}
	params.Set("location", location)

	var result AstronomyResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/astronomy", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get astronomy for location %q: %w", location, err)
	}
	return &result, nil
}

// GetUserAgent parses a user-agent string.
func (c *Client) GetUserAgent(ctx context.Context, uaString string) (*UserAgentResponse, error) {
	payload := map[string]string{
		"uaString": uaString,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	var result UserAgentResponse
	if err := c.doRequest(
		ctx,
		http.MethodPost,
		"/v3/user-agent",
		nil, // only apiKey will be added by doRequest
		bytes.NewReader(body),
		&result,
	); err != nil {
		return nil, fmt.Errorf("get user-agent for %q: %w", uaString, err)
	}

	return &result, nil
}

// SecurityResponse wraps the /v3/security endpoint response.
type SecurityResponse struct {
	IP       string    `json:"ip"`
	Security *Security `json:"security"`
}

// GetSecurity looks up security/threat data for an IP via the dedicated endpoint.
func (c *Client) GetSecurity(ctx context.Context, ip string) (*SecurityResponse, error) {
	params := url.Values{}
	if ip != "" {
		params.Set("ip", ip)
	}

	var result SecurityResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/security", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get security for %q: %w", ip, err)
	}
	return &result, nil
}

// AbuseContactResponse wraps the /v3/abuse endpoint response.
type AbuseContactResponse struct {
	IP    string        `json:"ip"`
	Abuse *AbuseContact `json:"abuse"`
}

// GetAbuseContact looks up abuse contact information for an IP.
func (c *Client) GetAbuseContact(ctx context.Context, ip string) (*AbuseContactResponse, error) {
	params := url.Values{}
	if ip != "" {
		params.Set("ip", ip)
	}

	var result AbuseContactResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/abuse", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get abuse contact for %q: %w", ip, err)
	}
	return &result, nil
}

// GetASNByIP looks up detailed ASN information for an IP via the dedicated endpoint.
// Includes routes, peers, upstreams, and downstreams.
func (c *Client) GetASNByIP(ctx context.Context, ip string) (*ASNResponse, error) {
	params := url.Values{}
	if ip != "" {
		params.Set("ip", ip)
	}
	params.Set("include", "peers,upstreams,downstreams,routes")

	var result ASNResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/asn", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get asn for ip %q: %w", ip, err)
	}
	return &result, nil
}

// GetASNByNumber looks up detailed ASN information by AS number.
func (c *Client) GetASNByNumber(ctx context.Context, asn string) (*ASNResponse, error) {
	params := url.Values{}
	params.Set("asn", asn)
	params.Set("include", "peers,upstreams,downstreams,routes")

	var result ASNResponse
	if err := c.doRequest(ctx, http.MethodGet, "/v3/asn", params, nil, &result); err != nil {
		return nil, fmt.Errorf("get asn for %q: %w", asn, err)
	}
	return &result, nil
}
