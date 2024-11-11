package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alvaroalonsobabbel/echo/server"
	"github.com/Alvaroalonsobabbel/echo/store"
	"github.com/stretchr/testify/assert"
)

func TestAcceptance(t *testing.T) {
	store, err := store.New()
	assert.NoError(t, err)
	defer store.Close()

	server := httptest.NewServer(server.New(store))
	defer server.Close()

	tests := []struct {
		name           string
		reqPath        string
		reqMethod      string
		reqBody        string
		wantResCode    int
		wantResBody    string
		wantResHeaders map[string]string
	}{
		{
			name:           "1 - Client requests non-existing path",
			reqPath:        "/hello",
			reqMethod:      http.MethodGet,
			wantResCode:    http.StatusNotFound,
			wantResBody:    "{\"errors\":[{\"code\":\"Not Found\", \"detail\":\"Requested page `/hello` does not exist\"}]}",
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
		},
		{
			name:           "2 - Client creates an endpoint",
			reqPath:        "/endpoints",
			reqMethod:      http.MethodPost,
			reqBody:        `{"data":{"type":"endpoints","attributes":{"verb":"GET","path":"/hello","response":{"code":200,"headers":{"Content-Type":"application/json"},"body":"\"{ \"message\": \"Hello, world\" }\""}}}}`,
			wantResCode:    http.StatusCreated,
			wantResBody:    `{"data":{"type":"endpoints","id":1,"attributes":{"verb":"GET","path":"/hello","response":{"code":200,"headers":{"Content-Type":"application/json"},"body":"\"{ \"message\": \"Hello, world\" }\""}}}}`,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
		},
		{
			name:           "3 - Client requests the recently created endpoint",
			reqPath:        "/hello",
			reqMethod:      http.MethodGet,
			wantResCode:    http.StatusOK,
			wantResBody:    `{"message":"Hello, world"}`,
			wantResHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:           "4 - Client requests the endpoint on the same path, but with different HTTP verb",
			reqPath:        "/hello",
			reqMethod:      http.MethodPost,
			wantResCode:    http.StatusNotFound,
			wantResBody:    "{\"errors\":[{\"code\":\"Not Found\", \"detail\":\"Requested page `/hello` does not exist\"}]}",
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(test.reqMethod, server.URL+test.reqPath, strings.NewReader(test.reqBody))
			assert.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			got, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.wantResCode, resp.StatusCode)
			assert.JSONEq(t, test.wantResBody, string(got))
			for k, v := range test.wantResHeaders {
				assert.Equal(t, resp.Header.Get(k), v)
			}
		})
	}
}
