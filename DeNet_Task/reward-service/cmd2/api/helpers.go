package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type jsonResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// readJSON tries to read the body of a request and converts it into JSON.
func (app *Config) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1048576

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)

	err := dec.Decode(data)
	if err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return ErrSingleJSON
	}

	return nil
}

// writeJSON takes a response status code and arbitrary data and writes a json response to the client.
func (app *Config) writeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// errorJSON takes an error, and optionally a response status code, and generates and sends a json error response.
func (app *Config) errorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()

	app.writeJSON(w, statusCode, payload) //nolint:errcheck
}
