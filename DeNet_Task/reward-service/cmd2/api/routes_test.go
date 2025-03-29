package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestCORS checks for CORS middleware correctly set headers
func TestCORS(t *testing.T) {
	router := gin.Default()
	app := &Config{}
	app.routes()

	req, _ := http.NewRequest("OPTIONS", "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", resp.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Accept, Authorization, Content-Type, X-CSRF-Token", resp.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "true", resp.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "300", resp.Header().Get("Access-Control-Max-Age"))

	assert.Equal(t, 204, resp.Code)
}

// TestPingRoute  checks routes for correct handling
func TestPingRoute(t *testing.T) {
	router := gin.Default()
	app := &Config{}
	app.routes()

	req, _ := http.NewRequest("GET", "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var response map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Nil(t, err)
	assert.Equal(t, "pong", response["message"])
}
