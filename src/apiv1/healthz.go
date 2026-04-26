// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/casjay-forks/caspaste/src/httputil"
)

type healthzResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Version   string `json:"version"`
	Database  string `json:"database"`
	Uptime    int64  `json:"uptime"`
}

var startTime = time.Now()

// GET /api/v1/healthz - health check per AI.md PART 13
// Supports content negotiation per AI.md PART 14:
// - Default: JSON
// - .txt extension or Accept: text/plain: plain text
func (data *Data) handleHealthz(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		rw.Header().Set("Allow", "GET")
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return nil
	}

	healthData := healthzResponse{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
		Version:   data.Version,
		Database:  "connected",
		Uptime:    int64(time.Since(startTime).Seconds()),
	}

	// Try to ping database
	_, err := data.DB.PasteDeleteExpired()
	if err != nil {
		healthData.Status = "degraded"
		healthData.Database = "error"
	}

	// Determine response format per AI.md PART 14
	format := httputil.GetAPIResponseFormat(req)

	// Set status code
	statusCode := http.StatusOK
	if err != nil {
		statusCode = http.StatusServiceUnavailable
	}

	// Return response based on format per AI.md PART 14, 16
	switch format {
	case httputil.FormatText:
		// Plain text response per AI.md PART 16
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(statusCode)
		if healthData.Status == "healthy" {
			fmt.Fprintf(rw, "OK: healthy\n")
			fmt.Fprintf(rw, "status: %s\n", healthData.Status)
			fmt.Fprintf(rw, "database: %s\n", healthData.Database)
			fmt.Fprintf(rw, "version: %s\n", healthData.Version)
			fmt.Fprintf(rw, "uptime: %d\n", healthData.Uptime)
		} else {
			fmt.Fprintf(rw, "ERROR: DEGRADED: %s (database: %s)\n", healthData.Status, healthData.Database)
		}
	default:
		// JSON response per AI.md PART 16 unified format
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(statusCode)
		resp := APIResponse{
			OK:   healthData.Status == "healthy",
			Data: healthData,
		}
		if healthData.Status != "healthy" {
			resp.Error = "DEGRADED"
			resp.Message = fmt.Sprintf("Service degraded: database %s", healthData.Database)
		}
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		rw.Write(jsonData)
		rw.Write([]byte("\n"))
	}

	return nil
}
