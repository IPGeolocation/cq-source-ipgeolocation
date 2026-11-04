package ipgeolocation

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRequestLimitValidation(t *testing.T) {
	for _, options := range [][]Option{
		{WithTimeout(0)}, {WithTimeout(-time.Second)},
		{WithMaxResponseBytes(0)}, {WithMaxResponseBytes(-1)}, {WithMaxResponseBytes(math.MaxInt64)},
		{WithHTTPClient(nil)}, {WithHTTPClient(nil), WithTimeout(time.Second)},
	} {
		_, err := NewClient("test-key", options...)
		require.Error(t, err)
	}
	c, err := NewClient("test-key", WithTimeout(time.Nanosecond), WithMaxResponseBytes(math.MaxInt64-1))
	require.NoError(t, err)
	assert.Equal(t, time.Nanosecond, c.httpClient.Timeout)
	shared := &http.Client{Timeout: time.Second}
	c, err = NewClient("test-key", WithHTTPClient(shared), WithTimeout(2*time.Second))
	require.NoError(t, err)
	assert.Equal(t, time.Second, shared.Timeout)
	assert.Equal(t, 2*time.Second, c.httpClient.Timeout)
}

func TestResponseLimitBoundaries(t *testing.T) {
	body := `{"ip":"8.8.8.8","location":{"city":"القاهرة"}}`
	for _, encoding := range []string{"length", "chunked", "gzip"} {
		for _, delta := range []int64{-1, 0, 1} {
			t.Run(fmt.Sprintf("%s/limit_delta_%d", encoding, delta), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					switch encoding {
					case "chunked":
						w.(http.Flusher).Flush()
						_, _ = io.WriteString(w, body)
					case "gzip":
						w.Header().Set("Content-Encoding", "gzip")
						gz := gzip.NewWriter(w)
						_, _ = io.WriteString(gz, body)
						_ = gz.Close()
					default:
						w.Header().Set("Content-Length", fmt.Sprint(len(body)))
						_, _ = io.WriteString(w, body)
					}
				}))
				t.Cleanup(server.Close)
				c, err := NewClient("test-key", WithEndpoint(server.URL), WithMaxResponseBytes(int64(len(body))+delta), WithRetries(1))
				require.NoError(t, err)
				result, err := c.GetIPGeolocation(context.Background(), "8.8.8.8")
				if delta < 0 {
					require.ErrorIs(t, err, errResponseTooLarge)
					assert.Nil(t, result)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "القاهرة", result.Location.City)
				}
				assert.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

type countingBody struct {
	read   int64
	closed bool
}

func (b *countingBody) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	b.read += int64(len(p))
	return len(p), nil
}
func (b *countingBody) Close() error { b.closed = true; return nil }

func TestResponseLimitStopsReading(t *testing.T) {
	c, err := NewClient("test-key", WithMaxResponseBytes(64))
	require.NoError(t, err)
	for _, length := range []int64{-1, 1 << 40} {
		body := &countingBody{}
		data, err := c.readResponse(&http.Response{Body: body, ContentLength: length})
		require.ErrorIs(t, err, errResponseTooLarge)
		assert.Nil(t, data)
		assert.True(t, body.closed)
		if length < 0 {
			assert.EqualValues(t, 65, body.read)
		} else {
			assert.Zero(t, body.read)
		}
	}
}

func TestHTTPErrorStatusSurvivesBodyFailures(t *testing.T) {
	for _, kind := range []string{"html", "invalid_message", "truncated", "oversized", "status_in_body"} {
		for _, status := range []int{http.StatusUnauthorized, http.StatusServiceUnavailable} {
			t.Run(fmt.Sprintf("%s/%d", kind, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if kind == "truncated" {
						w.Header().Set("Content-Length", "100")
					}
					w.WriteHeader(status)
					switch kind {
					case "html":
						_, _ = io.WriteString(w, "<html>unavailable</html>")
					case "invalid_message":
						_, _ = io.WriteString(w, `{"message":3}`)
					case "truncated":
						_, _ = io.WriteString(w, `{"message":`)
					case "oversized":
						_, _ = io.WriteString(w, strings.Repeat("x", 1000))
					case "status_in_body":
						_, _ = io.WriteString(w, `{"StatusCode":200,"message":"request failed"}`)
					}
				}))
				t.Cleanup(server.Close)
				c, err := NewClient("test-key", WithEndpoint(server.URL), WithMaxResponseBytes(200), WithRetries(0))
				require.NoError(t, err)
				result, err := c.GetIPGeolocation(context.Background(), "8.8.8.8")
				assert.Nil(t, result)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, status, apiErr.StatusCode)
				assert.Equal(t, status == http.StatusServiceUnavailable, apiErr.IsRetryable())
			})
		}
	}
}

func TestHTTPErrorRetriesAfterUnreadableBody(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, strings.Repeat("x", 100))
			return
		}
		_, _ = io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	t.Cleanup(server.Close)
	c, err := NewClient("test-key", WithEndpoint(server.URL), WithMaxResponseBytes(50), WithRetries(1))
	require.NoError(t, err)
	result, err := c.GetIPGeolocation(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	assert.Equal(t, "8.8.8.8", result.IP)
	assert.EqualValues(t, 2, calls.Load())
}

func TestRequestTimeoutAndCancellation(t *testing.T) {
	for _, stage := range []string{"headers", "body"} {
		t.Run(stage, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if stage == "body" {
					_, _ = io.WriteString(w, `{"ip":"`)
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			t.Cleanup(server.Close)
			c, err := NewClient("test-key", WithEndpoint(server.URL), WithTimeout(100*time.Millisecond), WithRetries(0))
			require.NoError(t, err)
			result, err := c.GetIPGeolocation(context.Background(), "8.8.8.8")
			assert.Nil(t, result)
			assertKeyHidden(t, err, "test-key")
			require.ErrorIs(t, err, context.DeadlineExceeded)
			var networkError net.Error
			require.ErrorAs(t, err, &networkError)
			assert.True(t, networkError.Timeout())
		})
	}
	c, err := NewClient("test-key", WithRetries(0))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.GetIPGeolocation(ctx, "8.8.8.8")
	require.ErrorIs(t, err, context.Canceled)
}

func TestLargeASNResponse(t *testing.T) {
	routes := make([]string, 50000)
	for i := range routes {
		routes[i] = fmt.Sprintf("2001:db8:%x::/48", i)
	}
	whois := strings.Repeat("network record\n", 1<<20)
	payload, err := json.Marshal(map[string]any{"asn": map[string]any{"as_number": "AS15169", "routes": routes, "whois_response": whois}})
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "15169", r.URL.Query().Get("asn"))
		assert.Equal(t, "routes,whois_response", r.URL.Query().Get("include"))
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)
	c, err := NewClient("test-key", WithEndpoint(server.URL), WithRetries(0))
	require.NoError(t, err)
	result, err := c.GetASNByNumber(context.Background(), "15169", "routes", "whois_response")
	require.NoError(t, err)
	assert.Equal(t, routes, result.ASN.Routes)
	assert.Equal(t, whois, result.ASN.WhoisResponse)
}
