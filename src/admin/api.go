
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package admin

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// jsonOK writes a JSON 200 response
func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

// jsonErr writes a JSON error response
func jsonErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// apiStatus returns overall server status
func (p *Panel) apiStatus(w http.ResponseWriter, r *http.Request) {
	totalPastes, _ := p.CountPastes()
	recentPastes, _ := p.CountPastesRecent()
	adminCount, _ := p.CountAdmins()

	jsonOK(w, map[string]interface{}{
		"status":        "ok",
		"uptime":        p.uptime(),
		"start_time":    p.cfg.StartTime.Format(time.RFC3339),
		"total_pastes":  totalPastes,
		"recent_pastes": recentPastes,
		"admin_count":   adminCount,
	})
}

// apiGetSettings returns display-safe settings (no secrets)
func (p *Panel) apiGetSettings(w http.ResponseWriter, r *http.Request) {
	if p.cfg.AppCfg == nil {
		jsonErr(w, http.StatusServiceUnavailable, "config not available")
		return
	}
	c := p.cfg.AppCfg
	jsonOK(w, map[string]interface{}{
		"title":           c.ServerTitle,
		"fqdn":            c.FQDN,
		"admin_name":      c.AdminName,
		"admin_mail":      c.AdminMail,
		"title_max_len":   c.TitleMaxLen,
		"body_max_len":    c.BodyMaxLen,
		"max_lifetime":    c.MaxLifeTime,
		"public":          c.Public,
		"api_version":     c.APIVersion,
		"mode":            c.Mode,
		"version":         c.Version,
		"build_commit":    c.BuildCommit,
		"build_date":      c.BuildDate,
	})
}

// apiPatchSettings accepts partial settings updates (JSON body)
func (p *Panel) apiPatchSettings(w http.ResponseWriter, r *http.Request) {
	jsonErr(w, http.StatusNotImplemented, "settings patching not yet implemented")
}

// apiInfo returns server info
func (p *Panel) apiInfo(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	info := map[string]interface{}{
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"goroutines": runtime.NumGoroutine(),
		"uptime":     p.uptime(),
		"start_time": p.cfg.StartTime.Format(time.RFC3339),
		"data_dir":   p.cfg.DataDir,
		"config_dir": p.cfg.ConfigDir,
		"backup_dir": p.cfg.BackupDir,
		"memory": map[string]interface{}{
			"heap_alloc":   ms.HeapAlloc,
			"heap_sys":     ms.HeapSys,
			"heap_objects": ms.HeapObjects,
			"gc_cycles":    ms.NumGC,
		},
	}
	if p.cfg.AppCfg != nil {
		info["version"] = p.cfg.AppCfg.Version
		info["build_commit"] = p.cfg.AppCfg.BuildCommit
		info["build_date"] = p.cfg.AppCfg.BuildDate
		info["mode"] = p.cfg.AppCfg.Mode
	}
	jsonOK(w, info)
}

// apiMetrics returns runtime and paste metrics
func (p *Panel) apiMetrics(w http.ResponseWriter, r *http.Request) {
	totalPastes, _ := p.CountPastes()
	recentPastes, _ := p.CountPastesRecent()
	adminCount, _ := p.CountAdmins()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	jsonOK(w, map[string]interface{}{
		"pastes": map[string]interface{}{
			"total":  totalPastes,
			"last24h": recentPastes,
		},
		"admins":     adminCount,
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"heap_alloc":   ms.HeapAlloc,
			"heap_sys":     ms.HeapSys,
			"heap_objects": ms.HeapObjects,
			"gc_cycles":    ms.NumGC,
		},
		"uptime": p.uptime(),
	})
}

// apiScheduler returns the list of scheduled tasks
func (p *Panel) apiScheduler(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	sched := p.sched
	p.mu.RUnlock()

	if sched == nil {
		jsonOK(w, []interface{}{})
		return
	}

	tasks := sched.ListTasks()
	result := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		lastRun := ""
		if !t.LastRun.IsZero() {
			lastRun = t.LastRun.Format(time.RFC3339)
		}
		nextRun := ""
		if !t.NextRun.IsZero() {
			nextRun = t.NextRun.Format(time.RFC3339)
		}
		result = append(result, map[string]interface{}{
			"id":          t.ID,
			"name":        t.Name,
			"description": t.Description,
			"schedule":    t.Schedule,
			"enabled":     t.Enabled,
			"last_run":    lastRun,
			"next_run":    nextRun,
			"last_status": string(t.LastStatus),
			"last_error":  t.LastError,
			"run_count":   t.RunCount,
			"fail_count":  t.FailCount,
		})
	}
	jsonOK(w, result)
}

// apiSchedulerRunNow triggers a scheduled task immediately
func (p *Panel) apiSchedulerRunNow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonErr(w, http.StatusBadRequest, "task id required")
		return
	}

	p.mu.RLock()
	sched := p.sched
	p.mu.RUnlock()

	if sched == nil {
		jsonErr(w, http.StatusServiceUnavailable, "scheduler not available")
		return
	}

	if err := sched.RunNow(id); err != nil {
		jsonErr(w, http.StatusNotFound, err.Error())
		return
	}
	jsonOK(w, map[string]string{"status": "triggered", "id": id})
}

// apiLogs returns the last N lines of the application log
func (p *Panel) apiLogs(w http.ResponseWriter, r *http.Request) {
	n := 100
	if s := r.URL.Query().Get("n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 1000 {
			n = v
		}
	}

	logFile := p.cfg.DataDir + "/logs/caspaste.log"
	tail, err := tailFile(logFile, n)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lines := strings.Split(tail, "\n")
	jsonOK(w, map[string]interface{}{"lines": lines, "count": len(lines)})
}

// apiListBackups returns metadata for all backup files
func (p *Panel) apiListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := listBackupFiles(p.cfg.BackupDir)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]map[string]interface{}, 0, len(backups))
	for _, b := range backups {
		result = append(result, map[string]interface{}{
			"name":     b.Name,
			"size":     b.Size,
			"mod_time": b.ModTime.Format(time.RFC3339),
		})
	}
	jsonOK(w, result)
}

// apiCreateBackup triggers a manual backup (stub — returns not implemented)
func (p *Panel) apiCreateBackup(w http.ResponseWriter, r *http.Request) {
	jsonErr(w, http.StatusNotImplemented, "backup creation not yet implemented")
}

// apiSSLInfo returns SSL/TLS configuration details
func (p *Panel) apiSSLInfo(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]interface{}{
		"tls":  r.TLS != nil,
		"acme": false,
	})
}

// apiEmailInfo returns SMTP configuration summary
func (p *Panel) apiEmailInfo(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]interface{}{
		"configured": false,
		"smtp_host":  "",
	})
}

// apiEmailTest sends a test email (stub)
func (p *Panel) apiEmailTest(w http.ResponseWriter, r *http.Request) {
	jsonErr(w, http.StatusNotImplemented, "email test not yet implemented")
}

// apiListTokens returns all admin API tokens (no raw values)
func (p *Panel) apiListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := p.listTokens()
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]map[string]interface{}, 0, len(tokens))
	for _, t := range tokens {
		var expires interface{}
		if t.ExpiresAt.Valid {
			expires = time.Unix(t.ExpiresAt.Int64, 0).Format(time.RFC3339)
		}
		var lastUsed interface{}
		if t.LastUsedAt.Valid {
			lastUsed = time.Unix(t.LastUsedAt.Int64, 0).Format(time.RFC3339)
		}
		result = append(result, map[string]interface{}{
			"id":           t.ID,
			"admin_id":     t.AdminID,
			"name":         t.Name,
			"prefix":       t.Prefix,
			"created_at":   time.Unix(t.CreatedAt, 0).Format(time.RFC3339),
			"last_used_at": lastUsed,
			"expires_at":   expires,
		})
	}
	jsonOK(w, result)
}

// apiCreateToken creates a new admin API token from a JSON body
func (p *Panel) apiCreateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		ExpireDays int    `json:"expire_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		jsonErr(w, http.StatusBadRequest, "name is required")
		return
	}

	adminID := currentAdminID(r)
	if err := p.createToken(adminID, req.Name, req.ExpireDays); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// apiRevokeToken deletes an admin API token by ID (id query param)
func (p *Panel) apiRevokeToken(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		jsonErr(w, http.StatusBadRequest, "id query parameter required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := p.revokeToken(id); err != nil {
		jsonErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// apiTorInfo returns Tor hidden service status
func (p *Panel) apiTorInfo(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]interface{}{
		"enabled":      false,
		"onion_address": "",
	})
}

// apiGeoIPInfo returns GeoIP configuration status
func (p *Panel) apiGeoIPInfo(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]interface{}{
		"enabled": false,
		"db_path": "",
	})
}
