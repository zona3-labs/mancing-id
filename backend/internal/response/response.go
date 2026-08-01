package response

import (
	"net/http"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// Envelope is the standard API response wrapper for successful endpoints.
type Envelope struct {
	Success   bool   `json:"success" example:"true"`
	Message   string `json:"message,omitempty"`
	Data      any    `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorEnvelope is retained for legacy handlers. New Catalog errors use ProblemDetails.
type ErrorEnvelope struct {
	Success   bool   `json:"success" example:"false"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// ProblemDetails is the RFC 9457 representation used for expected API errors.
type ProblemDetails struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	Instance  string `json:"instance,omitempty"`
	Code      string `json:"code"`
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

// BadRequest responds with the legacy 400 error envelope.
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Envelope{
		Success:   false,
		Error:     message,
		RequestID: requestID(c),
	})
}

// NotFound responds with the legacy 404 error envelope.
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Envelope{
		Success:   false,
		Error:     message,
		RequestID: requestID(c),
	})
}

// InternalError responds with 500 and an error message.
func InternalError(c *gin.Context, err error) {
	_ = err
	Problem(c, http.StatusInternalServerError, "internal_error", "Internal Server Error", "an internal error occurred")
}

// Conflict responds with the legacy 409 error envelope.
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Envelope{
		Success:   false,
		Error:     message,
		RequestID: requestID(c),
	})
}

func Problem(c *gin.Context, status int, code, title, detail string) {
	c.Header("Content-Type", "application/problem+json")
	c.JSON(status, ProblemDetails{
		Type:      "https://mancing.id/problems/catalog/" + code,
		Title:     title,
		Status:    status,
		Detail:    detail,
		Instance:  c.Request.URL.Path,
		Code:      code,
		RequestID: requestID(c),
	})
}
