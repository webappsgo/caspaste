
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package metric provides Prometheus metrics per AI.md PART 21.
// All metric follow Prometheus naming conventions with caspaste_ prefix.
package metric

import (
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config holds metrics configuration per AI.md PART 21
type Config struct {
	// Enabled controls whether metrics are active
	Enabled bool
	// Endpoint path for metrics (default: /metrics)
	Endpoint string
	// IncludeSystem includes system metrics (CPU, memory, disk)
	IncludeSystem bool
	// IncludeRuntime includes Go runtime metrics
	IncludeRuntime bool
	// Token for optional bearer token authentication
	Token string
	// DurationBuckets for request duration histogram
	DurationBuckets []float64
	// SizeBuckets for request/response size histogram
	SizeBuckets []float64
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() Config {
	return Config{
		Enabled:         false,
		Endpoint:        "/metrics",
		IncludeSystem:   true,
		IncludeRuntime:  true,
		Token:           "",
		DurationBuckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		SizeBuckets:     []float64{100, 1000, 10000, 100000, 1000000, 10000000},
	}
}

var (
	// Application metrics (REQUIRED per AI.md PART 21)
	AppInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "caspb_app_info",
			Help: "Application information",
		},
		[]string{"version", "commit", "build_date", "go_version"},
	)

	AppUptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	AppStartTime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_app_start_timestamp",
			Help: "Application start timestamp",
		},
	)

	// HTTP metrics (REQUIRED per AI.md PART 21)
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "caspb_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "caspb_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "path"},
	)

	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "caspb_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
		},
		[]string{"method", "path"},
	)

	HTTPActiveRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	// Database metrics (REQUIRED per AI.md PART 21)
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "caspb_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
		[]string{"operation", "table"},
	)

	DBConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_db_connections_open",
			Help: "Number of open database connections",
		},
	)

	DBConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_db_connections_in_use",
			Help: "Number of database connections in use",
		},
	)

	DBErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_db_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "error_type"},
	)

	// Authentication metrics (REQUIRED per AI.md PART 21)
	AuthAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_auth_attempts_total",
			Help: "Total authentication attempts",
		},
		[]string{"method", "status"},
	)

	AuthSessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_auth_sessions_active",
			Help: "Number of active sessions",
		},
	)

	// Cache metrics (optional)
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache"},
	)

	CacheEvictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_cache_evictions_total",
			Help: "Total number of cache evictions",
		},
		[]string{"cache"},
	)

	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "caspb_cache_size",
			Help: "Current cache size (items)",
		},
		[]string{"cache"},
	)

	CacheBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "caspb_cache_bytes",
			Help: "Current cache size (bytes)",
		},
		[]string{"cache"},
	)

	// Rate limiting metrics
	RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_ratelimit_requests_total",
			Help: "Total rate-limited requests",
		},
		[]string{"limit", "status"},
	)

	RateLimitBlockedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "caspb_ratelimit_blocked_total",
			Help: "Requests blocked by rate limiter",
		},
		[]string{"limit"},
	)

	// Go runtime metrics (if include_runtime: true)
	GoGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_go_goroutines",
			Help: "Current number of goroutines",
		},
	)

	GoMemAllocBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_go_mem_alloc_bytes",
			Help: "Bytes allocated and in use (heap)",
		},
	)

	GoMemSysBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_go_mem_sys_bytes",
			Help: "Total bytes obtained from system",
		},
	)

	GoGCRunsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "caspb_go_gc_runs_total",
			Help: "Total garbage collection runs",
		},
	)

	GoGCPauseTotalSeconds = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "caspb_go_gc_pause_total_seconds",
			Help: "Total time spent in GC pauses",
		},
	)

	// Paste-specific metrics
	PastesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_pastes_total",
			Help: "Total number of pastes",
		},
	)

	PastesCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "caspb_pastes_created_total",
			Help: "Total number of pastes created",
		},
	)

	PastesDeletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "caspb_pastes_deleted_total",
			Help: "Total number of pastes deleted",
		},
	)

	PastesBytesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "caspb_pastes_bytes_total",
			Help: "Total bytes of all pastes",
		},
	)
)

var (
	// Path normalization regexes
	uuidRegex      = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	numericIDRegex = regexp.MustCompile(`/[0-9]+(?:/|$)`)
	// CasPaste paste ID pattern (alphanumeric)
	pasteIDRegex = regexp.MustCompile(`/[a-zA-Z0-9]{6,}(?:/|$)`)

	// Global state
	startTime time.Time
	config    Config
	mu        sync.RWMutex
	lastGC    uint32
)

// Init initializes the metrics system
func Init(cfg Config, version, commit, buildDate string) {
	mu.Lock()
	defer mu.Unlock()

	config = cfg
	startTime = time.Now()

	if !cfg.Enabled {
		return
	}

	// Set application info
	AppInfo.WithLabelValues(version, commit, buildDate, runtime.Version()).Set(1)
	AppStartTime.SetToCurrentTime()

	// Start uptime updater
	go updateUptimeLoop()

	// Start runtime metrics collector if enabled
	if cfg.IncludeRuntime {
		go collectRuntimeMetricsLoop()
	}
}

// updateUptimeLoop updates the uptime metric every second
func updateUptimeLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.RLock()
		if !config.Enabled {
			mu.RUnlock()
			return
		}
		mu.RUnlock()

		AppUptime.Set(time.Since(startTime).Seconds())
	}
}

// collectRuntimeMetricsLoop collects Go runtime metrics every 15 seconds
func collectRuntimeMetricsLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.RLock()
		if !config.Enabled || !config.IncludeRuntime {
			mu.RUnlock()
			return
		}
		mu.RUnlock()

		collectRuntimeMetrics()
	}
}

// collectRuntimeMetrics collects current Go runtime metrics
func collectRuntimeMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	GoGoroutines.Set(float64(runtime.NumGoroutine()))
	GoMemAllocBytes.Set(float64(m.Alloc))
	GoMemSysBytes.Set(float64(m.Sys))

	// Track GC runs (delta)
	if m.NumGC > lastGC {
		GoGCRunsTotal.Add(float64(m.NumGC - lastGC))
		lastGC = m.NumGC
	}
}

// NormalizePath normalizes URL path for consistent metric labels
// Replaces dynamic segments (UUIDs, IDs, paste IDs) with placeholders
func NormalizePath(path string) string {
	// Replace UUIDs
	path = uuidRegex.ReplaceAllString(path, ":id")
	// Replace numeric IDs
	path = numericIDRegex.ReplaceAllString(path, "/:id/")
	// Replace paste IDs (for paste routes)
	if len(path) > 1 && path[0] == '/' {
		path = pasteIDRegex.ReplaceAllString(path, "/:paste_id/")
	}
	return path
}

// Handler returns the Prometheus metrics HTTP handler with optional auth
func Handler(cfg Config) http.Handler {
	promHandler := promhttp.Handler()

	if cfg.Token == "" {
		return promHandler
	}

	// Wrap with token authentication
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + cfg.Token

		if auth != expected {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		promHandler.ServeHTTP(w, r)
	})
}

// ResponseWriter wraps http.ResponseWriter to capture status and size
type ResponseWriter struct {
	http.ResponseWriter
	Status int
	Size   int
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(status int) {
	rw.Status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write captures the response size
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.Size += n
	return n, err
}

// NewResponseWriter creates a new metrics response writer
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		Status:         http.StatusOK,
	}
}

// Middleware creates HTTP metrics middleware per AI.md PART 21
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip metrics endpoint itself
			if r.URL.Path == cfg.Endpoint {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Track active requests
			HTTPActiveRequests.Inc()
			defer HTTPActiveRequests.Dec()

			// Get normalized path
			path := NormalizePath(r.URL.Path)

			// Record request size
			if r.ContentLength > 0 {
				HTTPRequestSize.WithLabelValues(r.Method, path).Observe(float64(r.ContentLength))
			}

			// Wrap response writer
			rw := NewResponseWriter(w)

			// Process request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.Status)

			HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
			HTTPResponseSize.WithLabelValues(r.Method, path).Observe(float64(rw.Size))
		})
	}
}

// RecordDBQuery records a database query metric
func RecordDBQuery(operation, table string, duration time.Duration, err error) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	DBQueriesTotal.WithLabelValues(operation, table).Inc()
	DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

	if err != nil {
		errType := classifyDBError(err)
		DBErrors.WithLabelValues(operation, errType).Inc()
	}
}

// classifyDBError classifies a database error type
func classifyDBError(err error) string {
	errStr := err.Error()
	switch {
	case contains(errStr, "connection"):
		return "connection"
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "constraint"):
		return "constraint"
	case contains(errStr, "duplicate"):
		return "duplicate"
	default:
		return "other"
	}
}

// contains checks if s contains substr (case-insensitive)
func contains(s, substr string) bool {
	return regexp.MustCompile(`(?i)` + regexp.QuoteMeta(substr)).MatchString(s)
}

// RecordAuth records an authentication attempt
func RecordAuth(method, status string) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	AuthAttempts.WithLabelValues(method, status).Inc()
}

// SetActiveSessions sets the number of active sessions
func SetActiveSessions(count float64) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	AuthSessionsActive.Set(count)
}

// UpdateDBConnections updates database connection metrics
func UpdateDBConnections(open, inUse int) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	DBConnectionsOpen.Set(float64(open))
	DBConnectionsInUse.Set(float64(inUse))
}

// RecordRateLimit records a rate limit event
func RecordRateLimit(limitType, status string) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	RateLimitRequestsTotal.WithLabelValues(limitType, status).Inc()
	if status == "limited" {
		RateLimitBlockedTotal.WithLabelValues(limitType).Inc()
	}
}

// RecordPasteCreated records a paste creation
func RecordPasteCreated() {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	PastesCreatedTotal.Inc()
}

// RecordPasteDeleted records a paste deletion
func RecordPasteDeleted() {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	PastesDeletedTotal.Inc()
}

// UpdatePasteStats updates paste statistics
func UpdatePasteStats(total int64, totalBytes int64) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	PastesTotal.Set(float64(total))
	PastesBytesTotal.Set(float64(totalBytes))
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheName string) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	CacheHits.WithLabelValues(cacheName).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheName string) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	CacheMisses.WithLabelValues(cacheName).Inc()
}

// UpdateCacheSize updates cache size metrics
func UpdateCacheSize(cacheName string, items int, bytes int64) {
	mu.RLock()
	enabled := config.Enabled
	mu.RUnlock()

	if !enabled {
		return
	}

	CacheSize.WithLabelValues(cacheName).Set(float64(items))
	CacheBytes.WithLabelValues(cacheName).Set(float64(bytes))
}

// IsEnabled returns whether metrics are enabled
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return config.Enabled
}

// GetConfig returns the current metrics configuration
func GetConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return config
}
