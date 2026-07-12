package response

import (
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// Envelope is the standard API response wrapper for all endpoints.
type Envelope struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func requestID(c *gin.Context) string {
	return requestid.Get(c)
}

// OK responds with 200 and a data payload.
func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Envelope{
		Success:   true,
		Message:   message,
		Data:      data,
		RequestID: requestID(c),
	})
}

// Created responds with 201 and the created resource.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{
		Success:   true,
		Data:      data,
		RequestID: requestID(c),
	})
}

// NoContent responds with 204 and no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest responds with 400 and an error message.
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Envelope{
		Success:   false,
		Error:     message,
		RequestID: requestID(c),
	})
}

// NotFound responds with 404 and an error message.
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Envelope{
		Success:   false,
		Error:     message,
		RequestID: requestID(c),
	})
}

// InternalError responds with 500 and an error message.
func InternalError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, Envelope{
		Success:   false,
		Error:     err.Error(),
		RequestID: requestID(c),
	})
}

// Conflict responds with 409 and an error message.
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Envelope{
		Success:   false,
		Error:     message,
		RequestID: requestID(c),
	})
}
