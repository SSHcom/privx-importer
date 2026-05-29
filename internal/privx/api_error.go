package privx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError contains full HTTP/API error information for callers.
type APIError struct {
	// Method is the HTTP method used for the request.
	Method string

	// Path is the PrivX API path used for the request.
	Path string

	// Request is the request payload sent to the API.
	Request any

	// StatusCode is the HTTP response status code.
	StatusCode int

	// Message is the parsed high-level error message.
	Message string

	// Body is the raw response body returned by the API.
	Body string
}

// NewAPIError creates a new API error from HTTP response data.
func NewAPIError(
	method string,
	path string,
	request any,
	statusCode int,
	body []byte,
) *APIError {
	err := &APIError{
		Method:     method,
		Path:       path,
		Request:    request,
		StatusCode: statusCode,
		Body:       string(body),
	}

	err.DecodeMessage()

	return err
}

func (e *APIError) Error() string {
	var b strings.Builder

	fmt.Fprintf(&b, "privx api error\n")
	fmt.Fprintf(&b, "  method: %s\n", e.Method)
	fmt.Fprintf(&b, "  path: %s\n", e.Path)
	fmt.Fprintf(&b, "  status: %d %s", e.StatusCode, http.StatusText(e.StatusCode))

	msg := strings.TrimSpace(e.Message)
	if msg != "" {
		fmt.Fprintf(&b, "\n")
		fmt.Fprintf(&b, "%s", indentLines(msg, "  "))
	}

	return b.String()
}

// DecodeMessage attempts to extract a human-readable message from Body.
func (e *APIError) DecodeMessage() {
	if strings.TrimSpace(e.Body) == "" {
		return
	}

	var obj map[string]any

	if err := json.Unmarshal([]byte(e.Body), &obj); err != nil {
		return
	}

	for _, key := range []string{
		"error_description",
		"message",
		"error",
		"detail",
	} {
		v, ok := obj[key].(string)
		if !ok {
			continue
		}

		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		e.Message = v
		return
	}
}
