// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/webappsgo/caspaste/src/httputil"
	"github.com/webappsgo/caspaste/src/netshare"
	"github.com/webappsgo/caspaste/src/storage"
)

// APIResponse is the unified response format per AI.md PART 16
type APIResponse struct {
	OK      bool        `json:"ok"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorInfo contains error code and message for consistent error handling
type ErrorInfo struct {
	Code    int
	ErrCode string
	Message string
}

// getErrorInfo maps errors to their codes and messages per AI.md PART 16
func getErrorInfo(e error) ErrorInfo {
	var eTmp429 *netshare.RateLimitError

	switch {
	case e == netshare.ErrBadRequest:
		return ErrorInfo{400, "BAD_REQUEST", "Invalid request format"}
	case e == netshare.ErrUnauthorized:
		return ErrorInfo{401, "UNAUTHORIZED", "Authentication required"}
	case e == storage.ErrNotFoundID:
		return ErrorInfo{404, "NOT_FOUND", "Paste not found"}
	case e == netshare.ErrNotFound:
		return ErrorInfo{404, "NOT_FOUND", "Resource not found"}
	case e == netshare.ErrMethodNotAllowed:
		return ErrorInfo{405, "METHOD_NOT_ALLOWED", "Method not allowed"}
	case e == netshare.ErrPayloadTooLarge:
		return ErrorInfo{413, "BAD_REQUEST", "Payload too large"}
	case e == netshare.ErrTooManyRequests:
		return ErrorInfo{429, "RATE_LIMITED", "Too many requests"}
	case errors.As(e, &eTmp429):
		return ErrorInfo{429, "RATE_LIMITED", "Too many requests"}
	default:
		return ErrorInfo{500, "SERVER_ERROR", "Internal server error"}
	}
}

func (data *Data) writeError(rw http.ResponseWriter, req *http.Request, e error) (int, error) {
	errInfo := getErrorInfo(e)

	// Set special headers for certain errors
	if e == netshare.ErrUnauthorized {
		rw.Header().Add("WWW-Authenticate", "Basic")
	}

	var eTmp429 *netshare.RateLimitError
	if errors.As(e, &eTmp429) {
		rw.Header().Set("Retry-After", strconv.FormatInt(eTmp429.RetryAfter, 10))
	}

	// Check response format per AI.md PART 14 content negotiation
	format := httputil.GetAPIResponseFormat(req)

	rw.WriteHeader(errInfo.Code)

	switch format {
	case httputil.FormatText:
		// Text response per AI.md PART 16: ERROR: {code}: {message}
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(rw, "ERROR: %s: %s\n", errInfo.ErrCode, errInfo.Message)
	default:
		// JSON response per AI.md PART 16
		rw.Header().Set("Content-Type", "application/json")
		resp := APIResponse{
			OK:      false,
			Error:   errInfo.ErrCode,
			Message: errInfo.Message,
		}
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		rw.Write(jsonData)
		rw.Write([]byte("\n"))
	}

	return errInfo.Code, nil
}

// writeSuccess writes a success response with content negotiation per AI.md PART 14, 16
// For JSON: {"ok": true, "data": {...}}
// For text: OK: {message}\n{data...}
func writeSuccess(w http.ResponseWriter, r *http.Request, data interface{}, textMsg string, textData string) error {
	format := httputil.GetAPIResponseFormat(r)

	switch format {
	case httputil.FormatText:
		// Text response per AI.md PART 16
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if textMsg != "" {
			fmt.Fprintf(w, "OK: %s\n", textMsg)
		}
		if textData != "" {
			fmt.Fprint(w, textData)
			if textData[len(textData)-1] != '\n' {
				fmt.Fprint(w, "\n")
			}
		}
		return nil
	default:
		// JSON response per AI.md PART 16
		return writeJSON(w, APIResponse{
			OK:   true,
			Data: data,
		})
	}
}

// writeJSON writes a JSON response with proper formatting per AI.md PART 14
// ALL JSON responses MUST be indented (2 spaces) and end with newline
func writeJSON(w http.ResponseWriter, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	// Single trailing newline per AI.md
	_, err = w.Write([]byte("\n"))
	return err
}
