package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alvaroalonsobabbel/echo/store"
	"github.com/stretchr/testify/assert"
)

const (
	example        = `{"data":{"type":"endpoints",%s"attributes":{"verb":"GET","path":"/greeting","response":{"code":200,"headers":{"Content-Type":"application/json"},"body":"\"{ \"message\": \"Hello, world\" }\""}}}}`
	exampleRename  = `{"data":{"type":"endpoints",%s"attributes":{"verb":"GET","path":"/post_it","response":{"code":201,"headers":{"Accept":"test/plain","x-api-key":"superdupersecret"},"body":"Your secrets are not so safe"}}}}`
	exampleError   = `{"data":{"type":"endpoints","attributes":{"verb":"GETS","path":"/greeting","response":{"code":200,"headers":{},"body":"\"{ \"message\": \"Hello, world\" }\""}}}}`
	expectedSeeded = `{"data":[{"type":"endpoints","id":1,"attributes":{"verb":"GET","path":"/revert_entropy","response":{"code":200,"headers":{"Content-Type":"application/json"},"body":"\"{ \"message\": \"INSUFFICIENT DATA FOR MEANINGFUL ANSWER\" }\""}}},{"type":"endpoints","id":2,"attributes":{"verb":"POST","path":"/post_it","response":{"code":201,"headers":{"Accept":"test/plain","x-api-key":"superdupersecret"},"body":"Your secrets are not so safe"}}},{"type":"endpoints","id":3,"attributes":{"verb":"PUT","path":"/fail","response":{"code":400,"headers":{"Accept":"test/plain","Content-Type":"application/json"},"body":"\"{\"error\": \"something went horribly wrong :(\" }\""}}},{"type":"endpoints","id":4,"attributes":{"verb":"DELETE","path":"/fake_delete","response":{"code":204,"headers":{},"body":""}}}]}`
)

func TestServer(t *testing.T) {
	tests := []struct {
		name           string
		seed           bool
		requestMethod  string
		requestPath    string
		requestBody    string
		wantResCode    int
		wantResBody    string
		wantResHeaders map[string]string
	}{
		{
			name:           "GET /endpoints returns an empty array",
			requestMethod:  "GET",
			requestPath:    "/endpoints",
			wantResCode:    200,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    `{"data":[]}`,
		},
		{
			name:           "GET /endpoints when seeded returns all the endpoints",
			seed:           true,
			requestMethod:  "GET",
			requestPath:    "/endpoints",
			wantResCode:    200,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    expectedSeeded,
		},
		{
			name:           "GET root returns 404",
			requestMethod:  "GET",
			requestPath:    "/",
			wantResCode:    404,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    "{\"errors\":[{\"code\":\"Not Found\", \"detail\":\"Requested page `/` does not exist\"}]}",
		},
		{
			name:           "not defined endpoint returns 404",
			requestMethod:  "GET",
			requestPath:    "/thisendpointdoesnotexists",
			wantResCode:    404,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    "{\"errors\":[{\"code\":\"Not Found\", \"detail\":\"Requested page `/thisendpointdoesnotexists` does not exist\"}]}",
		},
		{
			name:           "POST /endpoints creates a new endpoint",
			requestMethod:  "POST",
			requestPath:    "/endpoints",
			requestBody:    fmt.Sprintf(example, ""),
			wantResCode:    201,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    fmt.Sprintf(example, `"id":1,`),
		},
		{
			name:           "PATCH /endpoints/{id} updates the existing endpoint",
			seed:           true,
			requestMethod:  "PATCH",
			requestPath:    "/endpoints/2",
			requestBody:    fmt.Sprintf(exampleRename, ""),
			wantResCode:    201,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    fmt.Sprintf(exampleRename, `"id":2,`),
		},
		{
			name:           "DELETE /endpoints/{id} updates the existing endpoint",
			seed:           true,
			requestMethod:  "DELETE",
			requestPath:    "/endpoints/1",
			wantResCode:    204,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
		},
		{
			name:           "GET /revert_entropy returns custom endpoint info",
			seed:           true,
			requestMethod:  "GET",
			requestPath:    "/revert_entropy",
			wantResCode:    200,
			wantResHeaders: map[string]string{"Content-Type": "application/json"},
			wantResBody:    "{ \"message\": \"INSUFFICIENT DATA FOR MEANINGFUL ANSWER\" }",
		},
		{
			name:           "Error case",
			requestMethod:  "POST",
			requestPath:    "/endpoints",
			requestBody:    exampleError,
			wantResCode:    400,
			wantResHeaders: map[string]string{"Content-Type": "application/vnd.api+json"},
			wantResBody:    "{\"errors\":[{\"code\":\"Bad Request\", \"detail\":\"Key: 'Endpoint.Attributes.Verb' Error:Field validation for 'Verb' failed on the 'oneof' tag\"}]}",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := store.New()
			assert.NoError(t, err)
			defer store.Close()

			if test.seed {
				err := store.Seed()
				assert.NoError(t, err)
			}

			server := httptest.NewServer(New(store))
			defer server.Close()

			req, err := http.NewRequest(test.requestMethod, server.URL+test.requestPath, strings.NewReader(test.requestBody))
			assert.NoError(t, err)
			res, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)

			assert.Equal(t, test.wantResCode, res.StatusCode)
			for k, v := range test.wantResHeaders {
				assert.Equal(t, v, res.Header.Get(k))
			}
			body, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			defer res.Body.Close()
			assert.Equal(t, test.wantResBody, strings.TrimSuffix(string(body), "\n"))
		})
	}
}