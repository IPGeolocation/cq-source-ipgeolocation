package ipgeolocation

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

func TestClientHTTP2Requests(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, 2, r.ProtoMajor)
		assert.Equal(t, "test-key", r.URL.Query().Get("apiKey"))
		assert.Equal(t, "cq-http2-test", r.UserAgent())
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v3/ipgeo":
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "1.1.1.1", r.URL.Query().Get("ip"))
			assert.Equal(t, "security,abuse", r.URL.Query().Get("include"))
			_, _ = w.Write([]byte(`{"ip":"1.1.1.1","location":{"country_name":"Australia"}}`))
		case "/v3/user-agent":
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var body map[string]string
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, map[string]string{"uaString": "test-ua"}, body)
			_, _ = w.Write([]byte(`{"user_agent_string":"test-ua","name":"TestBrowser"}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	transport := &http2.Transport{
		TLSClientConfig: server.Client().Transport.(*http.Transport).TLSClientConfig.Clone(),
	}
	t.Cleanup(transport.CloseIdleConnections)
	c, err := NewClient("test-key",
		WithEndpoint(server.URL),
		WithHTTPClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}),
		WithUserAgent("cq-http2-test"),
		WithRetries(0),
	)
	require.NoError(t, err)

	geo, err := c.GetIPGeolocation(context.Background(), "1.1.1.1", "security", "abuse")
	require.NoError(t, err)
	assert.Equal(t, "1.1.1.1", geo.IP)
	assert.Equal(t, "Australia", geo.Location.CountryName)

	ua, err := c.GetUserAgent(context.Background(), "test-ua")
	require.NoError(t, err)
	assert.Equal(t, "test-ua", ua.UserAgentString)
	assert.Equal(t, "TestBrowser", ua.Name)
}

func TestHTTP2ResponseFraming(t *testing.T) {
	for _, tc := range []struct {
		name        string
		lengths     []string
		wantLengths []string
	}{
		{name: "valid", lengths: []string{"2"}, wantLengths: []string{"2"}},
		{name: "conflicting", lengths: []string{"2", "1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { _ = listener.Close() })
			serverDone := make(chan error, 1)
			go func() {
				serverDone <- func() error {
					conn, err := listener.Accept()
					if err != nil {
						return err
					}
					defer conn.Close()
					if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
						return err
					}
					preface := make([]byte, len(http2.ClientPreface))
					if _, err := io.ReadFull(conn, preface); err != nil {
						return err
					}
					framer := http2.NewFramer(conn, conn)
					if err := framer.WriteSettings(); err != nil {
						return err
					}
					for {
						frame, err := framer.ReadFrame()
						if err != nil {
							return err
						}
						if headers, ok := frame.(*http2.HeadersFrame); ok {
							// Write raw fields: net/http sanitizes duplicate lengths before sending them.
							var block bytes.Buffer
							encoder := hpack.NewEncoder(&block)
							if err := encoder.WriteField(hpack.HeaderField{Name: ":status", Value: "200"}); err != nil {
								return err
							}
							for _, value := range tc.lengths {
								if err := encoder.WriteField(hpack.HeaderField{Name: "content-length", Value: value}); err != nil {
									return err
								}
							}
							if err := framer.WriteHeaders(http2.HeadersFrameParam{
								StreamID: headers.StreamID, EndHeaders: true, BlockFragment: block.Bytes(),
							}); err != nil {
								return err
							}
							return framer.WriteData(headers.StreamID, true, []byte("{}"))
						}
					}
				}()
			}()
			transport := &http2.Transport{
				AllowHTTP: true,
				DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, network, addr)
				},
			}
			t.Cleanup(transport.CloseIdleConnections)
			client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
			resp, err := client.Get("http://" + listener.Addr().String())
			require.NoError(t, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "{}", string(body))
			assert.Equal(t, tc.wantLengths, resp.Header.Values("Content-Length"))
			require.NoError(t, <-serverDone)
		})
	}
}
