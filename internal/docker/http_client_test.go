package docker

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateHealthAction(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "Health status - starting",
			input:    []byte(`{"Action": "health_status", "HealthStatus": "starting"}`),
			expected: []byte(`{"Action": "health_status: running", "HealthStatus": "starting"}`),
		},
		{
			name:     "Health status - unhealthy",
			input:    []byte(`{"Action": "health_status", "HealthStatus": "unhealthy"}`),
			expected: []byte(`{"Action": "health_status: unhealthy", "HealthStatus": "unhealthy"}`),
		},
		{
			name:     "Health status - healthy",
			input:    []byte(`{"Action": "health_status", "HealthStatus": "healthy"}`),
			expected: []byte(`{"Action": "health_status: healthy", "HealthStatus": "healthy"}`),
		},
		{
			name:     "No health_status action",
			input:    []byte(`{"Action": "create"}`),
			expected: []byte(`{"Action": "create"}`),
		},
		{
			name:     "No Action target",
			input:    []byte(`{"Action": "health_status"}`),
			expected: []byte(`{"Action": "health_status"}`),
		},
		{
			name:     "No Action field",
			input:    []byte(`{"Etc": "etc"}`),
			expected: []byte(`{"Etc": "etc"}`),
		},
		{
			name:     "Empty input as is",
			input:    []byte(``),
			expected: []byte(``),
		},
		{
			name:     "Invalid JSON as is",
			input:    []byte(`{"Action": "`),
			expected: []byte(`{"Action": "`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := updateHealthAction(tt.input)
			if !bytes.Equal(output, tt.expected) {
				t.Errorf("expected %s, got %s", tt.expected, output)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		reqPath      string
		bodyToReturn string
		wantBody     string
	}{
		{
			name:         "Podman's starting health status as health_status: running",
			reqPath:      "/events",
			bodyToReturn: `{"Action": "health_status", "HealthStatus": "starting"}` + "\n",
			wantBody:     `{"Action": "health_status: running", "HealthStatus": "starting"}` + "\n",
		},
		{
			name:         "Another endpoints are not modified",
			reqPath:      "/something",
			bodyToReturn: `{"key": "value"}`,
			wantBody:     `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, tt.reqPath, r.URL.Path)

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.bodyToReturn))
			}))

			defer srv.Close()

			customTransport := &dockerClientTransport{
				transport: http.DefaultTransport,
			}

			client := &http.Client{
				Transport: customTransport,
			}

			resp, err := client.Get(srv.URL + tt.reqPath)
			require.NoError(t, err)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, tt.wantBody, string(body))
		})
	}
}
