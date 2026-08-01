// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Debug endpoints per AI.md PART 6: pprof, expvar, and application
// introspection. The entire /debug/* tree is registered only when
// mode.IsDebugEnabled() is true — with --debug/DEBUG=true off, none of
// these routes exist on the mux at all, so they are unreachable rather
// than merely returning 404.
package main

import (
	"encoding/json"
	"expvar"
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/webappsgo/caspaste/src/config"
	"github.com/webappsgo/caspaste/src/mode"
	"github.com/webappsgo/caspaste/src/scheduler"
	"github.com/webappsgo/caspaste/src/storage"
)

// debugRouteEntry describes a single registered route for /debug/routes.
type debugRouteEntry struct {
	Pattern string `json:"pattern"`
}

// recordingMux wraps http.ServeMux to track every pattern registered on it,
// so /debug/routes (AI.md PART 6) can report the actual routing table
// instead of a hand-maintained list.
type recordingMux struct {
	*http.ServeMux
	routes []debugRouteEntry
}

// newRecordingMux creates an empty recordingMux.
func newRecordingMux() *recordingMux {
	return &recordingMux{ServeMux: http.NewServeMux()}
}

// Handle registers handler for pattern and records it.
func (m *recordingMux) Handle(pattern string, handler http.Handler) {
	m.routes = append(m.routes, debugRouteEntry{Pattern: pattern})
	m.ServeMux.Handle(pattern, handler)
}

// HandleFunc registers handler for pattern and records it.
func (m *recordingMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.routes = append(m.routes, debugRouteEntry{Pattern: pattern})
	m.ServeMux.HandleFunc(pattern, handler)
}

// registerDebugRoutes registers the full /debug/* endpoint set from
// AI.md PART 6. It is a no-op unless mode.IsDebugEnabled() is true.
func registerDebugRoutes(mux *recordingMux, cfg *config.YAMLConfig, db storage.DB, sched *scheduler.Scheduler) {
	if !mode.IsDebugEnabled() {
		return
	}

	// pprof endpoints
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	// expvar endpoint
	mux.Handle("/debug/vars", expvar.Handler())

	// Custom introspection endpoints
	mux.HandleFunc("/debug/config", handleDebugConfig(cfg))
	mux.HandleFunc("/debug/routes", handleDebugRoutesList(mux))
	mux.HandleFunc("/debug/cache", handleDebugCache)
	mux.HandleFunc("/debug/db", handleDebugDB(db))
	mux.HandleFunc("/debug/scheduler", handleDebugScheduler(sched))
	mux.HandleFunc("/debug/memory", handleDebugMemory)
	mux.HandleFunc("/debug/goroutines", handleDebugGoroutines)
}

// writeDebugJSON writes v as indented JSON, rejecting non-GET requests.
func writeDebugJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	data, _ := json.MarshalIndent(v, "", "  ")
	w.Write(data)
	w.Write([]byte("\n"))
}

// handleDebugConfig returns sanitized configuration (passwords/secrets redacted)
func handleDebugConfig(cfg *config.YAMLConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sanitized := map[string]interface{}{
			"server": map[string]interface{}{
				"title":       cfg.Server.Title,
				"tagline":     cfg.Server.TagLine,
				"description": cfg.Server.Description,
				"fqdn":        cfg.Server.FQDN,
				"public":      cfg.Server.Public,
				"listen":      cfg.Server.Listen,
				"port":        cfg.Server.Port,
				"timeouts": map[string]interface{}{
					"read":  cfg.Server.Timeouts.Read,
					"write": cfg.Server.Timeouts.Write,
					"idle":  cfg.Server.Timeouts.Idle,
				},
			},
			"database": map[string]interface{}{
				"driver":         cfg.Database.Driver,
				"source":         "[REDACTED]",
				"cleanup_period": cfg.Database.CleanupPeriod,
				"max_open_conns": cfg.Database.MaxOpenConns,
				"max_idle_conns": cfg.Database.MaxIdleConns,
			},
			"security": map[string]interface{}{
				"tls": map[string]interface{}{
					"min_version": cfg.Security.TLS.MinVersion,
					"cert_file":   cfg.Security.TLS.CertFile,
					"key_file":    cfg.Security.TLS.KeyFile,
				},
			},
			"logging": map[string]interface{}{
				"level": cfg.Logging.Level,
			},
			"mode": map[string]interface{}{
				"app_mode": mode.GetCurrentAppMode().String(),
				"debug":    mode.IsDebugEnabled(),
			},
		}

		writeDebugJSON(w, r, sanitized)
	}
}

// handleDebugRoutesList returns every route pattern registered on mux.
func handleDebugRoutesList(mux *recordingMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeDebugJSON(w, r, map[string]interface{}{
			"count":  len(mux.routes),
			"routes": mux.routes,
		})
	}
}

// handleDebugCache returns cache statistics.
// CasPaste has no standalone in-memory cache subsystem (HTTP caching is
// done via Cache-Control/ETag response headers, see src/web/etag.go), so
// this reports that honestly rather than fabricating hit/miss counters.
func handleDebugCache(w http.ResponseWriter, r *http.Request) {
	writeDebugJSON(w, r, map[string]interface{}{
		"enabled": false,
		"note":    "no in-memory cache subsystem; HTTP responses use Cache-Control/ETag headers instead",
	})
}

// handleDebugDB returns database connection pool statistics.
func handleDebugDB(db storage.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := db.Pool().Stats()
		writeDebugJSON(w, r, map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration_ms":     stats.WaitDuration.Milliseconds(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
			"max_open_connections": stats.MaxOpenConnections,
		})
	}
}

// handleDebugScheduler returns the built-in scheduler's task status.
func handleDebugScheduler(sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeDebugJSON(w, r, sched.GetStatus())
	}
}

// handleDebugMemory returns memory statistics
func handleDebugMemory(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	writeDebugJSON(w, r, map[string]interface{}{
		"alloc_bytes":       m.Alloc,
		"total_alloc_bytes": m.TotalAlloc,
		"sys_bytes":         m.Sys,
		"mallocs":           m.Mallocs,
		"frees":             m.Frees,
		"heap_alloc_bytes":  m.HeapAlloc,
		"heap_sys_bytes":    m.HeapSys,
		"heap_idle_bytes":   m.HeapIdle,
		"heap_inuse_bytes":  m.HeapInuse,
		"heap_released":     m.HeapReleased,
		"heap_objects":      m.HeapObjects,
		"stack_inuse_bytes": m.StackInuse,
		"stack_sys_bytes":   m.StackSys,
		"gc_runs":           m.NumGC,
		"gc_pause_ns":       m.PauseNs[(m.NumGC+255)%256],
	})
}

// handleDebugGoroutines returns goroutine count
func handleDebugGoroutines(w http.ResponseWriter, r *http.Request) {
	writeDebugJSON(w, r, map[string]interface{}{
		"count":     runtime.NumGoroutine(),
		"gomaxproc": runtime.GOMAXPROCS(0),
		"num_cpu":   runtime.NumCPU(),
	})
}
