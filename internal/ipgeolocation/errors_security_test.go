package ipgeolocation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRoundTripper func(*http.Request) (*http.Response, error)

func (f testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func assertKeyHidden(t *testing.T, err error, key string) {
	t.Helper()
	require.Error(t, err)
	for current := err; current != nil; current = errors.Unwrap(current) {
		for _, formatted := range []string{current.Error(), fmt.Sprintf("%+v", current), fmt.Sprintf("%#v", current)} {
			assert.NotContains(t, formatted, key)
			assert.NotContains(t, formatted, url.QueryEscape(key))
		}
	}
}

func TestRequestErrorsRedactAPIKey(t *testing.T) {
	const key = "test-only key/+?&"
	t.Run("network error and nested details", func(t *testing.T) {
		c, err := NewClient(key, WithRetries(0), WithHTTPClient(&http.Client{
			Timeout: time.Second,
			Transport: testRoundTripper(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, key, r.URL.Query().Get("apiKey"))
				return nil, fmt.Errorf("connection failed for %s", key)
			}),
		}))
		require.NoError(t, err)
		_, err = c.GetIPGeolocation(context.Background(), "8.8.8.8")
		assertKeyHidden(t, err, key)
	})
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("api error %d", status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, key, r.URL.Query().Get("apiKey"))
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "rejected " + key + " " + url.QueryEscape(key)})
			}))
			t.Cleanup(server.Close)
			c, err := NewClient(key, WithEndpoint(server.URL), WithRetries(0))
			require.NoError(t, err)
			_, err = c.GetIPGeolocation(context.Background(), "8.8.8.8")
			assertKeyHidden(t, err, key)
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, status, apiErr.StatusCode)
		})
	}
}

func TestSuccessfulResponsePreservesText(t *testing.T) {
	const key = "test-only-placeholder"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, key, r.URL.Query().Get("apiKey"))
		_ = json.NewEncoder(w).Encode(map[string]any{"ip": "8.8.8.8", "location": map[string]string{"city": key}})
	}))
	t.Cleanup(server.Close)
	c, err := NewClient(key, WithEndpoint(server.URL), WithRetries(0))
	require.NoError(t, err)
	result, err := c.GetIPGeolocation(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	assert.Equal(t, key, result.Location.City)
}

func TestRedirectsDoNotSendAPIKeyInHeaders(t *testing.T) {
	const key = "test-only-redirect-key"
	var hits atomic.Int32
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		assert.Empty(t, r.Header.Get("Referer"))
		assert.NotContains(t, fmt.Sprint(r.Header), key)
		assert.Empty(t, r.URL.Query().Get("apiKey"))
		_, _ = w.Write([]byte(`{"ip":"8.8.8.8"}`))
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, key, r.URL.Query().Get("apiKey"))
		http.Redirect(w, r, target.URL+"/result", http.StatusFound)
	}))
	t.Cleanup(redirect.Close)
	for _, custom := range []bool{false, true} {
		hc := redirect.Client()
		var callbackCalls atomic.Int32
		if custom {
			hc.CheckRedirect = func(r *http.Request, via []*http.Request) error {
				callbackCalls.Add(1)
				assert.Empty(t, r.Header.Get("Referer"))
				r.Header.Set("Referer", via[0].URL.String())
				return nil
			}
		}
		c, err := NewClient(key, WithEndpoint(redirect.URL), WithHTTPClient(hc), WithTimeout(time.Second), WithRetries(0))
		require.NoError(t, err)
		result, err := c.GetIPGeolocation(context.Background(), "8.8.8.8")
		require.NoError(t, err)
		assert.Equal(t, "8.8.8.8", result.IP)
		if custom {
			assert.EqualValues(t, 1, callbackCalls.Load())
		}
	}
	assert.EqualValues(t, 2, hits.Load())
}

func TestRedirectLimitsAndCallbackErrors(t *testing.T) {
	const key = "test-only-redirect-key"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Redirect(w, r, "/next", http.StatusFound)
	}))
	t.Cleanup(server.Close)
	c, err := NewClient(key, WithEndpoint(server.URL), WithRetries(0))
	require.NoError(t, err)
	_, err = c.GetIPGeolocation(context.Background(), "8.8.8.8")
	require.ErrorContains(t, err, "stopped after 10 redirects")
	assert.EqualValues(t, 10, calls.Load())
	assertKeyHidden(t, err, key)

	hc := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return fmt.Errorf("redirect denied for %s", key)
	}}
	c, err = NewClient(key, WithEndpoint(server.URL), WithHTTPClient(hc), WithRetries(0))
	require.NoError(t, err)
	_, err = c.GetIPGeolocation(context.Background(), "8.8.8.8")
	require.ErrorContains(t, err, "redirect denied")
	assertKeyHidden(t, err, key)
}

func TestRequestCreationAndDecodeErrorsHideKey(t *testing.T) {
	const key = "test-only-secret"
	c, err := NewClient(key, WithEndpoint("http://invalid%"+key), WithRetries(0))
	require.NoError(t, err)
	_, err = c.GetIPGeolocation(context.Background(), key)
	assertKeyHidden(t, err, key)

	numericKey := strings.Repeat("1", 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"location":{"is_eu":%s}}`, numericKey)
	}))
	t.Cleanup(server.Close)
	c, err = NewClient(numericKey, WithEndpoint(server.URL), WithRetries(0))
	require.NoError(t, err)
	_, err = c.GetIPGeolocation(context.Background(), "8.8.8.8")
	assertKeyHidden(t, err, numericKey)
}
