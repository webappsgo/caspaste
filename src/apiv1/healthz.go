// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

var startTime = time.Now()

// healthzProject holds project identity fields
type healthzProject struct {
	Name        string `json:"name"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
}

// healthzBuild holds build metadata
type healthzBuild struct {
	Commit string `json:"commit"`
	Date   string `json:"date"`
}

// healthzCluster holds cluster state
type healthzCluster struct {
	Enabled bool `json:"enabled"`
}

// healthzTor holds Tor feature state
type healthzTor struct {
	Enabled  bool   `json:"enabled"`
	Running  bool   `json:"running"`
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
}

// healthzFeatures holds feature flags
type healthzFeatures struct {
	Tor                healthzTor `json:"tor"`
	GeoIP              bool       `json:"geoip"`
	SyntaxHighlighting bool       `json:"syntax_highlighting"`
	MultiUser          bool       `json:"multi_user"`
}

// healthzChecks holds component health check results
type healthzChecks struct {
	Database  string `json:"database"`
	Cache     string `json:"cache"`
	Disk      string `json:"disk"`
	Scheduler string `json:"scheduler"`
}

// healthzStats holds aggregate stats
type healthzStats struct {
	RequestsTotal     int64 `json:"requests_total"`
	Requests24h       int64 `json:"requests_24h"`
	ActiveConnections int64 `json:"active_connections"`
	PastesTotal       int64 `json:"pastes_total"`
	Pastes24h         int64 `json:"pastes_24h"`
}

// healthzResponse is the flat healthz payload per AI.md PART 13
type healthzResponse struct {
	Project   healthzProject  `json:"project"`
	Status    string          `json:"status"`
	Version   string          `json:"version"`
	GoVersion string          `json:"go_version"`
	Build     healthzBuild    `json:"build"`
	Uptime    string          `json:"uptime"`
	Mode      string          `json:"mode"`
	Timestamp time.Time       `json:"timestamp"`
	Cluster   healthzCluster  `json:"cluster"`
	Features  healthzFeatures `json:"features"`
	Checks    healthzChecks   `json:"checks"`
	Stats     healthzStats    `json:"stats"`
}

// formatUptime formats seconds into a human-readable compact string (e.g. "2d 5h 30m")
func formatUptime(totalSeconds int64) string {
	if totalSeconds < 1 {
		return "just started"
	}

	parts := []string{}

	years := totalSeconds / (365 * 24 * 3600)
	if years > 0 {
		if years == 1 {
			parts = append(parts, "1y")
		} else {
			parts = append(parts, fmt.Sprintf("%dy", years))
		}
		totalSeconds %= (365 * 24 * 3600)
	}

	weeks := totalSeconds / (7 * 24 * 3600)
	if weeks > 0 {
		if weeks == 1 {
			parts = append(parts, "1w")
		} else {
			parts = append(parts, fmt.Sprintf("%dw", weeks))
		}
		totalSeconds %= (7 * 24 * 3600)
	}

	days := totalSeconds / (24 * 3600)
	if days > 0 {
		if days == 1 {
			parts = append(parts, "1d")
		} else {
			parts = append(parts, fmt.Sprintf("%dd", days))
		}
		totalSeconds %= (24 * 3600)
	}

	hours := totalSeconds / 3600
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1h")
		} else {
			parts = append(parts, fmt.Sprintf("%dh", hours))
		}
		totalSeconds %= 3600
	}

	minutes := totalSeconds / 60
	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1m")
		} else {
			parts = append(parts, fmt.Sprintf("%dm", minutes))
		}
		totalSeconds %= 60
	}

	if totalSeconds > 0 || len(parts) == 0 {
		if totalSeconds == 1 {
			parts = append(parts, "1s")
		} else {
			parts = append(parts, fmt.Sprintf("%ds", totalSeconds))
		}
	}

	return strings.Join(parts, " ")
}

// checkDisk verifies the data directory is writable by creating and removing a temp file.
func (data *Data) checkDisk() string {
	dir := data.DataDir
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.CreateTemp(dir, ".healthz-*")
	if err != nil {
		return "error: " + err.Error()
	}
	f.Close()
	os.Remove(f.Name())
	return "ok"
}

// checkScheduler returns the scheduler health string.
func (data *Data) checkScheduler() string {
	if data.SchedulerStatus != nil {
		return data.SchedulerStatus()
	}
	return "ok"
}

// buildHealthz assembles the healthz response, performing live checks
func (data *Data) buildHealthz() healthzResponse {
	status := "healthy"
	dbStatus := "ok"

	_, err := data.DB.PasteDeleteExpired()
	if err != nil {
		status = "degraded"
		dbStatus = "error"
	}

	diskStatus := data.checkDisk()
	if diskStatus != "ok" {
		status = "degraded"
	}

	schedulerStatus := data.checkScheduler()
	if schedulerStatus != "ok" {
		status = "degraded"
	}

	var pastesTotal int64

	uptime := formatUptime(int64(time.Since(startTime).Seconds()))

	return healthzResponse{
		Project: healthzProject{
			Name:        data.ServerTitle,
			Tagline:     data.ServerTagline,
			Description: data.ServerDescription,
		},
		Status:    status,
		Version:   data.Version,
		GoVersion: runtime.Version(),
		Build: healthzBuild{
			Commit: data.BuildCommit,
			Date:   data.BuildDate,
		},
		Uptime:    uptime,
		Mode:      data.Mode,
		Timestamp: time.Now().UTC(),
		Cluster: healthzCluster{
			Enabled: false,
		},
		Features: healthzFeatures{
			Tor: healthzTor{
				Enabled:  false,
				Running:  false,
				Status:   "disabled",
				Hostname: "",
			},
			GeoIP:              false,
			SyntaxHighlighting: true,
			MultiUser:          false,
		},
		Checks: healthzChecks{
			Database:  dbStatus,
			Cache:     "n/a",
			Disk:      diskStatus,
			Scheduler: schedulerStatus,
		},
		Stats: healthzStats{
			RequestsTotal:     0,
			Requests24h:       0,
			ActiveConnections: 0,
			PastesTotal:       pastesTotal,
			Pastes24h:         0,
		},
	}
}

// GET /api/v1/server/healthz — health check per AI.md PART 13
// Content negotiation:
//   - Accept: text/plain or path ends in .txt → plain text key:value
//   - Default → flat JSON (no APIResponse envelope)
func (data *Data) handleHealthz(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		rw.Header().Set("Allow", "GET")
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return nil
	}

	h := data.buildHealthz()

	statusCode := http.StatusOK
	if h.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	acceptsText := strings.Contains(req.Header.Get("Accept"), "text/plain") ||
		strings.HasSuffix(req.URL.Path, ".txt")

	if acceptsText {
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(statusCode)
		fmt.Fprintf(rw, "status: %s\n", h.Status)
		fmt.Fprintf(rw, "version: %s\n", h.Version)
		fmt.Fprintf(rw, "go_version: %s\n", h.GoVersion)
		fmt.Fprintf(rw, "build_commit: %s\n", h.Build.Commit)
		fmt.Fprintf(rw, "build_date: %s\n", h.Build.Date)
		fmt.Fprintf(rw, "uptime: %s\n", h.Uptime)
		fmt.Fprintf(rw, "mode: %s\n", h.Mode)
		fmt.Fprintf(rw, "database: %s\n", h.Checks.Database)
		fmt.Fprintf(rw, "cache: %s\n", h.Checks.Cache)
		fmt.Fprintf(rw, "disk: %s\n", h.Checks.Disk)
		fmt.Fprintf(rw, "scheduler: %s\n", h.Checks.Scheduler)
		fmt.Fprintf(rw, "pastes_total: %d\n", h.Stats.PastesTotal)
		return nil
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	jsonData, _ := json.MarshalIndent(h, "", "  ")
	rw.Write(jsonData)
	rw.Write([]byte("\n"))
	return nil
}
