package calltypes

// JSONResponse API response
// @Description API response.
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ErrorResponse represents standard error response
// @name ErrorResponse.
type ErrorResponse struct {
	Error   bool   `example:"true"              json:"error"`
	Message string `example:"Error description" json:"message"`
}
