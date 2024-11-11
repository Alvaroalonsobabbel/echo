package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	_ "github.com/mattn/go-sqlite3" // SQL driver
)

const dbSchema = `CREATE TABLE IF NOT EXISTS endpoints (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL, verb TEXT NOT NULL,
  path TEXT NOT NULL, code INTEGER NOT NULL,
  headers TEXT NOT NULL, body TEXT NOT NULL
)`

const seedDB = `INSERT INTO endpoints (
  type, verb, path, code, headers, body
)
VALUES
  (
    'endpoints', 'GET', '/revert_entropy', 200,
    '{"Content-Type": "application/json"}',
    '"{ "message": "INSUFFICIENT DATA FOR MEANINGFUL ANSWER" }"'
  ),
  (
    'endpoints', 'POST', '/post_it', 201,
    '{"Accept": "test/plain", "x-api-key":"superdupersecret"}',
    'Your secrets are not so safe'
  ),
  (
    'endpoints', 'PUT', '/fail', 400,
    '{"Accept": "test/plain", "Content-Type": "application/json"}',
    '"{"error": "something went horribly wrong :(" }"'
  ),
  (
    'endpoints', 'DELETE', '/fake_delete',
    204, '{}',
    ''
  )`

const (
	createEndpointQuery    = `INSERT INTO endpoints ( verb, path, code, headers, body, type ) VALUES ( ?, ?, ?, ?, ?, ? ) RETURNING id, type, verb, path, code, headers, body`
	updateEndpointQuery    = `UPDATE endpoints SET type = ?, verb = ?, path = ?, code = ?, headers = ?, body = ? WHERE id = ? RETURNING id, type, verb, path, code, headers, body`
	fetchEndpointsQuery    = "SELECT * FROM endpoints ORDER by id"
	fetchEndpointByIDQuery = "SELECT * FROM endpoints WHERE id = ?;"
	deleteEndpointQuery    = "DELETE FROM endpoints WHERE id = ?"
	findEndpointQuery      = "SELECT code, headers, body FROM endpoints WHERE verb = ? AND path = ?"
)

type One struct {
	Data *Endpoint `json:"data" validate:"required"`
}

type Many struct {
	Data []*Endpoint `json:"data"`
}

type Endpoint struct {
	Type       string     `json:"type" validate:"required,oneof=endpoints"`
	ID         int        `json:"id"`
	Attributes Attributes `json:"attributes" validate:"required"`
}

func (e *Endpoint) Verify() error {
	return validator.New().Struct(e)
}

type Attributes struct {
	Verb     string   `json:"verb" validate:"required,oneof=GET HEAD OPTIONS TRACE PUT DELETE POST PATCH CONNECT"`
	Path     string   `json:"path" validate:"required,uri"`
	Response Response `json:"response" validate:"required"`
}

type Response struct {
	Code    int               `json:"code" validate:"required,gte=100,lte=599"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (r *Response) Serve(w http.ResponseWriter) {
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

type Store struct {
	db *sql.DB
}

func New() (*Store, error) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("unable to open DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping DB: %v", err)
	}

	if _, err := db.Exec(dbSchema); err != nil {
		return nil, fmt.Errorf("unable to create tables: %v", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Seed() error {
	if _, err := s.db.Exec(seedDB); err != nil {
		return fmt.Errorf("unable to seed db: %v", err)
	}
	return nil
}

func (s *Store) FetchEndpoints() (*Many, error) {
	rows, err := s.db.Query(fetchEndpointsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	data := []*Endpoint{}
	for rows.Next() {
		e := &Endpoint{}
		var headers string
		if err := rows.Scan(
			&e.ID,
			&e.Type,
			&e.Attributes.Verb,
			&e.Attributes.Path,
			&e.Attributes.Response.Code,
			&headers,
			&e.Attributes.Response.Body,
		); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(headers), &e.Attributes.Response.Headers); err != nil {
			return nil, err
		}

		data = append(data, e)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &Many{Data: data}, nil
}

func (s *Store) CreateEndpoint(endpoint *Endpoint) (*One, error) {
	h, err := json.Marshal(endpoint.Attributes.Response.Headers)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRow(createEndpointQuery,
		endpoint.Attributes.Verb,
		endpoint.Attributes.Path,
		endpoint.Attributes.Response.Code,
		string(h),
		endpoint.Attributes.Response.Body,
		endpoint.Type,
	)
	e := &Endpoint{}
	var headers string
	err = row.Scan(
		&e.ID,
		&e.Type,
		&e.Attributes.Verb,
		&e.Attributes.Path,
		&e.Attributes.Response.Code,
		&headers,
		&e.Attributes.Response.Body,
	)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(headers), &e.Attributes.Response.Headers)

	return &One{Data: e}, err
}

func (s *Store) DeleteEndpoint(id string) (bool, error) {
	result, err := s.db.Exec(deleteEndpointQuery, id)
	if err != nil {
		return false, err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affectedRows == 0 {
		return false, nil
	}

	return true, nil
}

func (s *Store) FindEndpoint(verb, path string) (*Response, error) {
	row := s.db.QueryRow(findEndpointQuery, verb, path)
	r := &Response{}
	var headers string
	if err := row.Scan(&r.Code, &headers, &r.Body); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	err := json.Unmarshal([]byte(headers), &r.Headers)

	return r, err
}

func (s *Store) UpdateEndpoint(id string, endpoint *Endpoint) (*One, error) {
	h, err := json.Marshal(endpoint.Attributes.Response.Headers)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRow(updateEndpointQuery,
		endpoint.Type,
		endpoint.Attributes.Verb,
		endpoint.Attributes.Path,
		endpoint.Attributes.Response.Code,
		string(h),
		endpoint.Attributes.Response.Body,
		id,
	)
	e := &Endpoint{}
	var headers string
	err = row.Scan(
		&e.ID,
		&e.Type,
		&e.Attributes.Verb,
		&e.Attributes.Path,
		&e.Attributes.Response.Code,
		&headers,
		&e.Attributes.Response.Body,
	)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(headers), &e.Attributes.Response.Headers)

	return &One{Data: e}, err
}
