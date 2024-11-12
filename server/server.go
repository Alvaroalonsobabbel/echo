package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Alvaroalonsobabbel/echo/store"
	"github.com/go-playground/validator/v10"
)

const (
	getEndpointsPath    = "GET /endpoints"
	postEndpoinstPath   = "POST /endpoints"
	patchEndpointsPath  = "PATCH /endpoints/{id}"
	deleteEndpointsPath = "DELETE /endpoints/{id}"

	errorMessage = `{"errors":[{"code":"%s", "detail":"%s"}]}`
)

func New(store *store.Store) http.Handler {
	handle := &handlers{store, validator.New()}
	mux := http.NewServeMux()

	mux.HandleFunc(getEndpointsPath, handle.fetchEndpoints())
	mux.HandleFunc(postEndpoinstPath, handle.createEndpoint())
	mux.HandleFunc(patchEndpointsPath, handle.updateEndpoint())
	mux.HandleFunc(deleteEndpointsPath, handle.deleteEndpoint())
	mux.HandleFunc("/", handle.all())

	return withVndHeaderMiddleware(mux)
}

func withVndHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		next.ServeHTTP(w, r)
	})
}

type handlers struct {
	*store.Store
	*validator.Validate
}

func (h *handlers) fetchEndpoints() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		e, err := h.FetchEndpoints()
		if err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("unable to fetch endpoints: %v", err))
			return
		}
		if err := json.NewEncoder(w).Encode(e); err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("error serializing endpoints: %v", err))
			return
		}
	}
}

func (h *handlers) createEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := h.unmarshalAndVerify(r)
		if err != nil {
			replyWithErr(w, http.StatusBadRequest, err.Error())
			return
		}
		ok, err := h.FindEndpoint(e.Attributes.Verb, e.Attributes.Path)
		if err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("error finding endpoint: %v", err))
			return
		}
		if ok != nil {
			replyWithErr(w, http.StatusConflict, fmt.Sprintf("the requested endpoint `%s %s` already exists", e.Attributes.Verb, e.Attributes.Path))
			return
		}
		created, err := h.CreateEndpoint(e)
		if err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("unable to create endpoint: %v", err))
			return
		}
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(created); err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("error encpding endpoint: %v", err))
			return
		}
	}
}

func (h *handlers) updateEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := h.unmarshalAndVerify(r)
		if err != nil {
			replyWithErr(w, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := h.UpdateEndpoint(r.PathValue("id"), e)
		if err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("unable to update endpoint: %v", err))
			return
		}
		if updated == nil {
			replyWithErr(w, http.StatusNotFound, fmt.Sprintf("Requested Endpoint with ID `%s` does not exist", r.PathValue("id")))
			return
		}
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(updated); err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("error encoding endpoint: %v", err))
			return
		}
	}
}

func (h *handlers) deleteEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, err := h.DeleteEndpoint(r.PathValue("id"))
		if err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("unable to delete endpoint: %v", err))
			return
		}
		if ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		replyWithErr(w, http.StatusNotFound, fmt.Sprintf("Requested Endpoint with ID `%s` does not exist", r.PathValue("id")))
	}
}

func (h *handlers) all() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := h.FindEndpoint(r.Method, r.URL.Path)
		if err != nil {
			replyWithErr(w, http.StatusInternalServerError, fmt.Sprintf("error finding endpoint: %v", err))
			return
		}
		if e == nil {
			replyWithErr(w, http.StatusNotFound, fmt.Sprintf("Requested page `%s` does not exist", r.URL.Path))
			return
		}
		serve(w, e)
	}
}

func (h *handlers) unmarshalAndVerify(r *http.Request) (*store.Endpoint, error) {
	e := &store.One{}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(e); err != nil {
		return nil, fmt.Errorf("Unable to decode request body: %v", err)
	}
	if err := h.Struct(e.Data); err != nil {
		return nil, err
	}
	return e.Data, nil
}

func serve(w http.ResponseWriter, r *store.Response) {
	// Remove application/vnd.api+json passed by middleware.
	w.Header().Del("Content-Type")

	r.Body = strings.TrimPrefix(r.Body, "\"")
	r.Body = strings.TrimSuffix(r.Body, "\"")
	for k, v := range r.Headers {
		w.Header().Add(k, v)
	}
	w.WriteHeader(r.Code)

	fmt.Fprint(w, r.Body)
}

func replyWithErr(w http.ResponseWriter, code int, err string) {
	if code == http.StatusInternalServerError {
		log.Printf("internal error: %s", err)
		err = "Something went horribly wrong :("
	}
	w.WriteHeader(code)
	fmt.Fprintf(w, errorMessage, http.StatusText(code), err)
}
