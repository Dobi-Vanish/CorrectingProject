package httputils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reward-service/pkg/consts"
	"reward-service/pkg/errormsg"
)

type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadJSON reads JSON sent information.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	maxBytes := consts.Megabyte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return errormsg.ErrJSONDecode
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errormsg.ErrJSONMustContain
	}

	return nil
}

// WriteJSON write JSON response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if len(headers) > 0 {
		for key, values := range headers[0] {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err = w.Write(out); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

// ErrorJSON godoc
// @Summary Return error response in JSON format
// @Description Helper function to send standardized error responses
// @Tags Utilities
// @Produce json
// @Param err query string true "Error message"
// @Param status query int false "HTTP status code" default(400)
// @Success 400 {object} JSONResponse "Standard error response"
// @Success 401 {object} JSONResponse "Unauthorized error"
// @Success 403 {object} JSONResponse "Forbidden error"
// @Success 404 {object} JSONResponse "Not found error"
// @Success 500 {object} JSONResponse "Internal server error"
// @Router /error [get].
func ErrorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	payload := JSONResponse{
		Error:   true,
		Message: err.Error(),
	}
	_ = WriteJSON(w, statusCode, payload)
}
