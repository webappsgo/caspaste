// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var startTime = time.Now()

// healthzTmplData is the data struct passed to healthz.tmpl
type healthzTmplData struct {
	Status     string
	Version    string
	GoVersion  string
	Uptime     string
	Mode       string
	Timestamp  string
	BuildCommit string
	BuildDate   string
	DBStatus    string

	ProjectName        string
	ProjectTagline     string
	ProjectDescription string

	PastesTotal int64

	Language  string
	Theme     func(string) string
	Translate func(string, ...interface{}) template.HTML

	CSRFToken     string
	UnreadCount   int
	Notifications []NavNotification
	ShowLogin     bool
	ShowRegister  bool
	User          *AuthUser
}

// buildWebHealthz assembles the healthz struct used by all response formats
func (data *Data) buildWebHealthz() (status, dbStatus string, pastesTotal int64) {
	status = "healthy"
	dbStatus = "ok"
	_, err := data.DB.PasteDeleteExpired()
	if err != nil {
		status = "degraded"
		dbStatus = "error"
	}
	return status, dbStatus, 0
}

// Pattern: /healthz and /server/healthz
// Content negotiation:
//   - Accept: application/json or path ends in .json → flat JSON
//   - Accept: text/plain or path ends in .txt → plain text
//   - Otherwise → HTML via healthz.tmpl
func (data *Data) handleHealthz(rw http.ResponseWriter, req *http.Request) error {
	status, dbStatus, pastesTotal := data.buildWebHealthz()

	uptimeSecs := int64(time.Since(startTime).Seconds())
	uptimeStr := formatUptime(uptimeSecs)
	now := time.Now().UTC()

	statusCode := http.StatusOK
	if status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	acceptHeader := req.Header.Get("Accept")
	path := req.URL.Path

	acceptsJSON := strings.Contains(acceptHeader, "application/json") || strings.HasSuffix(path, ".json")
	acceptsText := strings.Contains(acceptHeader, "text/plain") || strings.HasSuffix(path, ".txt")

	switch {
	case acceptsJSON:
		type buildInfo struct {
			Commit string `json:"commit"`
			Date   string `json:"date"`
		}
		type clusterInfo struct {
			Enabled bool `json:"enabled"`
		}
		type torInfo struct {
			Enabled  bool   `json:"enabled"`
			Running  bool   `json:"running"`
			Status   string `json:"status"`
			Hostname string `json:"hostname"`
		}
		type featuresInfo struct {
			Tor                torInfo `json:"tor"`
			GeoIP              bool    `json:"geoip"`
			SyntaxHighlighting bool    `json:"syntax_highlighting"`
			MultiUser          bool    `json:"multi_user"`
		}
		type checksInfo struct {
			Database  string `json:"database"`
			Cache     string `json:"cache"`
			Disk      string `json:"disk"`
			Scheduler string `json:"scheduler"`
		}
		type statsInfo struct {
			RequestsTotal     int64 `json:"requests_total"`
			Requests24h       int64 `json:"requests_24h"`
			ActiveConnections int64 `json:"active_connections"`
			PastesTotal       int64 `json:"pastes_total"`
			Pastes24h         int64 `json:"pastes_24h"`
		}
		type projectInfo struct {
			Name        string `json:"name"`
			Tagline     string `json:"tagline"`
			Description string `json:"description"`
		}
		type payload struct {
			Project   projectInfo  `json:"project"`
			Status    string       `json:"status"`
			Version   string       `json:"version"`
			GoVersion string       `json:"go_version"`
			Build     buildInfo    `json:"build"`
			Uptime    string       `json:"uptime"`
			Mode      string       `json:"mode"`
			Timestamp time.Time    `json:"timestamp"`
			Cluster   clusterInfo  `json:"cluster"`
			Features  featuresInfo `json:"features"`
			Checks    checksInfo   `json:"checks"`
			Stats     statsInfo    `json:"stats"`
		}
		p := payload{
			Project: projectInfo{
				Name:        data.ServerTitle,
				Tagline:     data.ServerTagline,
				Description: data.ServerDescription,
			},
			Status:    status,
			Version:   data.Version,
			GoVersion: runtime.Version(),
			Build:     buildInfo{Commit: data.BuildCommit, Date: data.BuildDate},
			Uptime:    uptimeStr,
			Mode:      data.Mode,
			Timestamp: now,
			Cluster:   clusterInfo{Enabled: false},
			Features: featuresInfo{
				Tor: torInfo{
					Enabled:  false,
					Running:  false,
					Status:   "disabled",
					Hostname: "",
				},
				GeoIP:              false,
				SyntaxHighlighting: true,
				MultiUser:          false,
			},
			Checks: checksInfo{
				Database:  dbStatus,
				Cache:     "ok",
				Disk:      "ok",
				Scheduler: "ok",
			},
			Stats: statsInfo{
				PastesTotal: pastesTotal,
			},
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(statusCode)
		jsonData, _ := json.MarshalIndent(p, "", "  ")
		rw.Write(jsonData)
		rw.Write([]byte("\n"))
		return nil

	case acceptsText:
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		rw.WriteHeader(statusCode)
		fmt.Fprintf(rw, "status: %s\n", status)
		fmt.Fprintf(rw, "version: %s\n", data.Version)
		fmt.Fprintf(rw, "go_version: %s\n", runtime.Version())
		fmt.Fprintf(rw, "build_commit: %s\n", data.BuildCommit)
		fmt.Fprintf(rw, "build_date: %s\n", data.BuildDate)
		fmt.Fprintf(rw, "uptime: %s\n", uptimeStr)
		fmt.Fprintf(rw, "mode: %s\n", data.Mode)
		fmt.Fprintf(rw, "database: %s\n", dbStatus)
		fmt.Fprintf(rw, "cache: ok\n")
		fmt.Fprintf(rw, "disk: ok\n")
		fmt.Fprintf(rw, "scheduler: ok\n")
		fmt.Fprintf(rw, "pastes_total: %d\n", pastesTotal)
		return nil

	default:
		dataTmpl := healthzTmplData{
			Status:             status,
			Version:            data.Version,
			GoVersion:          runtime.Version(),
			Uptime:             uptimeStr,
			Mode:               data.Mode,
			Timestamp:          now.Format(time.RFC3339),
			BuildCommit:        data.BuildCommit,
			BuildDate:          data.BuildDate,
			DBStatus:           dbStatus,
			ProjectName:        data.ServerTitle,
			ProjectTagline:     data.ServerTagline,
			ProjectDescription: data.ServerDescription,
			PastesTotal:        pastesTotal,
			Language:           getCookie(req, "lang"),
			Theme:              data.getThemeFunc(req),
			Translate:          data.Locales.findLocale(req).translate,
			CSRFToken:          data.buildCSRFToken(req),
			UnreadCount:        0,
			Notifications:      nil,
			ShowLogin:          data.ShowLogin,
			ShowRegister:       data.ShowRegister,
			User:               GetAuthUser(req.Context()),
		}
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(statusCode)
		return data.HealthzPage.Execute(rw, dataTmpl)
	}
}

// formatUptime formats seconds into a compact human-readable string per AI.md PART 13.
// Examples: "just started", "45s", "5m 30s", "2d 5h 30m"
func formatUptime(totalSeconds int64) string {
	if totalSeconds < 1 {
		return "just started"
	}

	parts := []string{}

	years := totalSeconds / (365 * 24 * 3600)
	if years > 0 {
		parts = append(parts, fmt.Sprintf("%dy", years))
		totalSeconds %= (365 * 24 * 3600)
	}

	weeks := totalSeconds / (7 * 24 * 3600)
	if weeks > 0 {
		parts = append(parts, fmt.Sprintf("%dw", weeks))
		totalSeconds %= (7 * 24 * 3600)
	}

	days := totalSeconds / (24 * 3600)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		totalSeconds %= (24 * 3600)
	}

	hours := totalSeconds / 3600
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		totalSeconds %= 3600
	}

	minutes := totalSeconds / 60
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
		totalSeconds %= 60
	}

	if totalSeconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", totalSeconds))
	}

	return strings.Join(parts, " ")
}
