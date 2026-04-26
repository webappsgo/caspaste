// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var startTime = time.Now()

// Pattern: /healthz - HTML health check page
func (data *Data) handleHealthz(rw http.ResponseWriter, req *http.Request) error {
	uptime := int64(time.Since(startTime).Seconds())
	uptimeStr := formatUptime(uptime)

	// Try database check
	dbStatus := "Connected"
	statusClass := "healthy"
	_, err := data.DB.PasteDeleteExpired()
	if err != nil {
		dbStatus = "Error"
		statusClass = "degraded"
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Health Check - ` + data.ServerTitle + `</title>
	<style>
		* { box-sizing: border-box; margin: 0; padding: 0; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 1rem;
		}
		.container {
			background: white;
			border-radius: 12px;
			box-shadow: 0 20px 60px rgba(0,0,0,0.3);
			padding: 2rem;
			max-width: 600px;
			width: 100%;
		}
		h1 {
			color: #2d3748;
			font-size: 2rem;
			margin-bottom: 1.5rem;
			text-align: center;
		}
		.status {
			text-align: center;
			padding: 1.5rem;
			border-radius: 8px;
			margin-bottom: 2rem;
			font-size: 1.5rem;
			font-weight: bold;
		}
		.status.healthy {
			background: #c6f6d5;
			color: #22543d;
		}
		.status.degraded {
			background: #fed7d7;
			color: #742a2a;
		}
		.info-grid {
			display: grid;
			grid-template-columns: auto 1fr;
			gap: 0.75rem 1.5rem;
			margin-bottom: 1.5rem;
		}
		.label {
			font-weight: 600;
			color: #4a5568;
		}
		.value {
			color: #2d3748;
			font-family: 'Courier New', monospace;
		}
		.footer {
			text-align: center;
			color: #718096;
			font-size: 0.875rem;
			margin-top: 2rem;
			padding-top: 1rem;
			border-top: 1px solid #e2e8f0;
		}
		.footer a {
			color: #667eea;
			text-decoration: none;
		}
		.footer a:hover {
			text-decoration: underline;
		}
		@media (max-width: 480px) {
			h1 { font-size: 1.5rem; }
			.status { font-size: 1.25rem; padding: 1rem; }
			.info-grid { grid-template-columns: 1fr; gap: 0.5rem; }
			.label::after { content: ": "; }
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🏥 Health Check</h1>
		<div class="status ` + statusClass + `">
			` + func() string {
		if statusClass == "healthy" {
			return "✓ HEALTHY"
		}
		return "⚠ DEGRADED"
	}() + `
		</div>
		<div class="info-grid">
			<span class="label">Server</span>
			<span class="value">` + data.ServerTitle + `</span>

			<span class="label">Version</span>
			<span class="value">` + data.Version + `</span>

			<span class="label">FQDN</span>
			<span class="value">` + data.FQDN + `</span>

			<span class="label">Database</span>
			<span class="value">` + dbStatus + `</span>

			<span class="label">Uptime</span>
			<span class="value">` + uptimeStr + `</span>

			<span class="label">Timestamp</span>
			<span class="value">` + time.Now().Format("2006-01-02 15:04:05 MST") + `</span>
		</div>
		<div class="footer">
			<a href="/">← Back to CasPaste</a> |
			<a href="/api/v1/healthz">JSON API</a> |
			<a href="/server/about">About</a>
		</div>
	</div>
</body>
</html>`

	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	if statusClass == "degraded" {
		rw.WriteHeader(http.StatusServiceUnavailable)
	} else {
		rw.WriteHeader(http.StatusOK)
	}
	rw.Write([]byte(html))
	return nil
}

// formatUptime formats seconds into human-readable uptime
// Examples: "20 seconds", "1 hour and 20 minutes", "3 years, 34 weeks, 6 days, 21 hours, 32 minutes"
func formatUptime(totalSeconds int64) string {
	if totalSeconds < 1 {
		return "just started"
	}

	parts := []string{}

	// Years (365 days)
	years := totalSeconds / (365 * 24 * 3600)
	if years > 0 {
		if years == 1 {
			parts = append(parts, "1 year")
		} else {
			parts = append(parts, fmt.Sprintf("%d years", years))
		}
		totalSeconds %= (365 * 24 * 3600)
	}

	// Weeks (7 days)
	weeks := totalSeconds / (7 * 24 * 3600)
	if weeks > 0 {
		if weeks == 1 {
			parts = append(parts, "1 week")
		} else {
			parts = append(parts, fmt.Sprintf("%d weeks", weeks))
		}
		totalSeconds %= (7 * 24 * 3600)
	}

	// Days
	days := totalSeconds / (24 * 3600)
	if days > 0 {
		if days == 1 {
			parts = append(parts, "1 day")
		} else {
			parts = append(parts, fmt.Sprintf("%d days", days))
		}
		totalSeconds %= (24 * 3600)
	}

	// Hours
	hours := totalSeconds / 3600
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
		totalSeconds %= 3600
	}

	// Minutes
	minutes := totalSeconds / 60
	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
		totalSeconds %= 60
	}

	// Seconds
	if totalSeconds > 0 || len(parts) == 0 {
		if totalSeconds == 1 {
			parts = append(parts, "1 second")
		} else {
			parts = append(parts, fmt.Sprintf("%d seconds", totalSeconds))
		}
	}

	// Join with commas and "and" before last item
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + " and " + parts[1]
	}
	// 3+ parts: "a, b, c, and d"
	return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
}
