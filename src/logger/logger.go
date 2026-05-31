
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package logger

import (
	"encoding/json"
	"fmt"
	"github.com/casjay-forks/caspaste/src/netshare"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

type LogFormat struct {
	// Access log format: apache, nginx, text, json
	Access string
	// Error log format: text, json
	Error string
	// Server log format: text, json
	Server string
	// Debug log format: text, json
	Debug string
}

type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
)

type Logger struct {
	TimeFormat string
	Format     LogFormat
	Level      LogLevel
	
	// File writers - always write regardless of level
	serverFile io.Writer
	errorFile  io.Writer
	accessFile io.Writer
	debugFile  io.Writer
	
	// Console writers - filtered by level
	stdout     io.Writer
	stderr     io.Writer
	
	debugMode  bool
}

func New(timeFormat string) Logger {
	return Logger{
		TimeFormat: timeFormat,
		// Default to info level
		Level: LogLevelInfo,
		Format: LogFormat{
			Access: "apache",
			Error:  "text",
			Server: "text",
			Debug:  "text",
		},
		stdout:     os.Stdout,
		stderr:     os.Stderr,
	}
}

// SetFormat sets the log format for each log type
func (l *Logger) SetFormat(format LogFormat) {
	l.Format = format
}

// SetLevel sets the minimum log level (info, warn, error)
func (l *Logger) SetLevel(level string) {
	switch level {
	case "warn":
		l.Level = LogLevelWarn
	case "error":
		l.Level = LogLevelError
	default:
		l.Level = LogLevelInfo
	}
}

// SetWriter sets both stdout and stderr to the same writer
func (l *Logger) SetWriter(w io.Writer) {
	l.stdout = w
	l.stderr = w
}

// SetWriters sets stdout and stderr separately (for console output)
func (l *Logger) SetWriters(stdout, stderr io.Writer) {
	l.stdout = stdout
	l.stderr = stderr
}

// SetFileWriters sets the file writers for each log type
func (l *Logger) SetFileWriters(server, errorLog io.Writer) {
	l.serverFile = server
	l.errorFile = errorLog
}

// SetAccessLogWriter sets the writer for HTTP access logs
func (l *Logger) SetAccessLogWriter(w io.Writer) {
	l.accessFile = w
}

// SetDebugWriter sets the writer for debug logs
func (l *Logger) SetDebugWriter(w io.Writer) {
	l.debugFile = w
}

// SetDebugMode enables or disables debug logging
func (l *Logger) SetDebugMode(enabled bool) {
	l.debugMode = enabled
}

// Debug writes debug messages (only if debug mode is enabled)
func (l *Logger) Debug(msg string) {
	if !l.debugMode || l.debugFile == nil {
		return
	}
	
	if l.Format.Debug == "json" {
		entry := map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "DEBUG",
			"message": msg,
		}
		data, _ := json.Marshal(entry)
		// Always write to file
		fmt.Fprintln(l.debugFile, string(data))
	} else {
		// Always write to file
		fmt.Fprintln(l.debugFile, time.Now().Format(l.TimeFormat), "[DEBUG]  ", msg)
	}
}

func getTrace() string {
	trace := ""

	for i := 2; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok {
			trace = trace + file + "#" + strconv.Itoa(line) + ": "

		} else {
			return trace
		}
	}
}

func (cfg Logger) Info(msg string) {
	// Format the message
	var output string
	if cfg.Format.Server == "json" {
		entry := map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "INFO",
			"message": msg,
		}
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = fmt.Sprintf("%s [INFO]    %s", time.Now().Format(cfg.TimeFormat), msg)
	}
	
	// Always write to file
	if cfg.serverFile != nil {
		fmt.Fprintln(cfg.serverFile, output)
	}
	
	// Only write to stdout if level permits (info level)
	if cfg.Level <= LogLevelInfo && cfg.stdout != nil {
		fmt.Fprintln(cfg.stdout, output)
	}
}

func (cfg Logger) Warn(msg string) {
	// Format the message
	var output string
	if cfg.Format.Server == "json" {
		entry := map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "WARN",
			"message": msg,
		}
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = fmt.Sprintf("%s [WARN]    %s", time.Now().Format(cfg.TimeFormat), msg)
	}
	
	// Always write to file
	if cfg.serverFile != nil {
		fmt.Fprintln(cfg.serverFile, output)
	}
	
	// Only write to stdout if level permits (warn or lower)
	if cfg.Level <= LogLevelWarn && cfg.stdout != nil {
		fmt.Fprintln(cfg.stdout, output)
	}
}

func (cfg Logger) Error(e error) {
	// Format the message
	var output string
	if cfg.Format.Error == "json" {
		entry := map[string]interface{}{
			"time":    time.Now().Format(time.RFC3339),
			"level":   "ERROR",
			"trace":   getTrace(),
			"message": e.Error(),
		}
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = fmt.Sprintf("%s [ERROR]   %s%s", time.Now().Format(cfg.TimeFormat), getTrace(), e.Error())
	}
	
	// Always write to file
	if cfg.errorFile != nil {
		fmt.Fprintln(cfg.errorFile, output)
	}
	
	// Always write errors to stderr (errors always shown)
	if cfg.stderr != nil {
		fmt.Fprintln(cfg.stderr, output)
	}
}

func (cfg Logger) HttpRequest(req *http.Request, code int) {
	clientIP := netshare.GetClientAddr(req).String()
	method := req.Method
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path = path + "?" + req.URL.RawQuery
	}
	referer := req.Referer()
	if referer == "" {
		referer = "-"
	}
	userAgent := req.UserAgent()
	if userAgent == "" {
		userAgent = "-"
	}
	
	// Write to access.log file - HTTP request logs
	if cfg.accessFile != nil {
		switch cfg.Format.Access {
		case "json":
			entry := map[string]interface{}{
				"time":       time.Now().Format(time.RFC3339),
				"client_ip":  clientIP,
				"method":     method,
				"path":       path,
				"protocol":   req.Proto,
				"status":     code,
				"referer":    referer,
				"user_agent": userAgent,
			}
			data, _ := json.Marshal(entry)
			fmt.Fprintln(cfg.accessFile, string(data))
			
		case "nginx":
			// Nginx Combined Log Format
			timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")
			fmt.Fprintf(cfg.accessFile, "%s - - [%s] \"%s %s %s\" %d 0 \"%s\" \"%s\"\n",
				clientIP, timestamp, method, path, req.Proto, code, referer, userAgent)
			
		case "text":
			// Simple text format
			timestamp := time.Now().Format(cfg.TimeFormat)
			fmt.Fprintf(cfg.accessFile, "%s %s %s %s %d %s\n",
				timestamp, clientIP, method, path, code, userAgent)
			
		// "apache" or unspecified
		default:
			// Apache Combined Log Format (default)
			timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")
			fmt.Fprintf(cfg.accessFile, "%s - - [%s] \"%s %s %s\" %d - \"%s\" \"%s\"\n",
				clientIP, timestamp, method, path, req.Proto, code, referer, userAgent)
		}
	}
}

func (cfg Logger) HttpError(req *http.Request, e error) {
	clientIP := netshare.GetClientAddr(req).String()
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path = path + "?" + req.URL.RawQuery
	}

	// Format the message
	var output string
	if cfg.Format.Error == "json" {
		entry := map[string]interface{}{
			"time":       time.Now().Format(time.RFC3339),
			"level":      "ERROR",
			"client_ip":  clientIP,
			"method":     req.Method,
			"path":       path,
			"user_agent": req.UserAgent(),
			"trace":      getTrace(),
			"error":      e.Error(),
		}
		data, _ := json.Marshal(entry)
		output = string(data)
	} else {
		output = fmt.Sprintf("%s [ERROR]   %s %s %s (User-Agent: %s) Error: %s%s",
			time.Now().Format(cfg.TimeFormat), clientIP, req.Method, path,
			req.UserAgent(), getTrace(), e.Error())
	}

	// Always write to error log file
	if cfg.errorFile != nil {
		fmt.Fprintln(cfg.errorFile, output)
	}

	// Write to stderr if configured
	if cfg.stderr != nil {
		fmt.Fprintln(cfg.stderr, output)
	}
}
