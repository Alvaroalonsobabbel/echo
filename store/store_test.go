package store

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndpointsVerify(t *testing.T) {
	happyPath := "happy path"
	tests := []struct {
		name   string
		modify func(e *Endpoint)
	}{
		{happyPath, func(*Endpoint) {}},
		{"incorrect type attribute", func(e *Endpoint) { e.Type = "test" }},
		{"empty type attribute", func(e *Endpoint) { e.Type = "" }},
		{"incorrect verb attribute", func(e *Endpoint) { e.Attributes.Verb = "CUAK" }},
		{"empty verb attribute", func(e *Endpoint) { e.Attributes.Verb = "" }},
		{"incorrect path attribute", func(e *Endpoint) { e.Attributes.Path = "@@" }},
		{"empty path attribute", func(e *Endpoint) { e.Attributes.Path = "" }},
		{"incorrect code attribute", func(e *Endpoint) { e.Attributes.Response.Code = 600 }},
		{"empty code attribute", func(e *Endpoint) { e.Attributes.Response.Code = 0 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint := newTestEndpoint()
			test.modify(endpoint)
			if test.name == happyPath {
				assert.NoError(t, endpoint.Verify())
				return
			}
			assert.Error(t, endpoint.Verify())
		})
	}
}

func TestStorage(t *testing.T) {
	store, err := New()
	assert.NoError(t, err)
	defer store.Close()
	err = store.Seed()
	assert.NoError(t, err)

	testEndpoint := newTestEndpoint()

	t.Run("FetchEndpoints return all endpoints", func(t *testing.T) {
		assertLenEndpoints(t, 4, store)
	})

	t.Run("CreateEndpoint creates a new endpoint", func(t *testing.T) {
		created, err := store.CreateEndpoint(testEndpoint)
		assert.NoError(t, err)
		assert.NotZero(t, created.Data.ID)
		assert.Equal(t, testEndpoint.Attributes, created.Data.Attributes)
		assertLenEndpoints(t, 5, store)
	})

	t.Run("UpdateEndpoint updates an existing endpoint", func(t *testing.T) {
		updated, err := store.UpdateEndpoint("2", testEndpoint)
		assert.NoError(t, err)
		assert.Equal(t, testEndpoint.Attributes, updated.Data.Attributes)
	})

	t.Run("UpdateEndpoint returns nil when updating an endpoint that does not exist", func(t *testing.T) {
		updated, err := store.UpdateEndpoint("12", testEndpoint)
		assert.NoError(t, err)
		assert.Nil(t, updated)
	})

	t.Run("DeleteEndpoint deletes an existing endpoint", func(t *testing.T) {
		ok, err := store.DeleteEndpoint("5")
		assert.NoError(t, err)
		assert.True(t, ok)
		assertLenEndpoints(t, 4, store)
	})

	t.Run("DeleteEndpoint return false when given wrong id", func(t *testing.T) {
		ok, err := store.DeleteEndpoint("15")
		assert.NoError(t, err)
		assert.False(t, ok)
		assertLenEndpoints(t, 4, store)
	})

	t.Run("FindEndpoint finds an endpoint by given Verb and Path", func(t *testing.T) {
		e, err := store.FindEndpoint(http.MethodGet, "/revert_entropy")
		assert.NoError(t, err)

		assert.NotNil(t, e)
		assert.Equal(t, http.StatusOK, e.Code)
		assert.Equal(t, map[string]string{"Content-Type": "application/json"}, e.Headers)
		assert.Equal(t, `"{ "message": "INSUFFICIENT DATA FOR MEANINGFUL ANSWER" }"`, e.Body)
	})

	t.Run("FindEndpoint returns nil when not finding and enpoint", func(t *testing.T) {
		e, err := store.FindEndpoint(http.MethodGet, "/noluck")
		assert.NoError(t, err)
		assert.Nil(t, e)
	})
}

func assertLenEndpoints(t testing.TB, want int, s *Store) {
	e, err := s.FetchEndpoints()
	assert.NoError(t, err)
	assert.Equal(t, want, len(e.Data))
}

func newTestEndpoint() *Endpoint {
	return &Endpoint{
		Type: "endpoints",
		ID:   1,
		Attributes: Attributes{
			Verb: "GET",
			Path: "/hello",
			Response: Response{
				Code:    200,
				Headers: map[string]string{"Accept": "application/json"},
			},
		},
	}
}
