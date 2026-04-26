// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

// Package geoip provides GeoIP support per AI.md PART 20
// Uses sapics/ip-location-db databases (MMDB format)
package geoip

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// Database URLs from ip-location-db (no API key required)
const (
	ASNDatabaseURL     = "https://cdn.jsdelivr.net/npm/@ip-location-db/asn-mmdb/asn.mmdb"
	CountryDatabaseURL = "https://cdn.jsdelivr.net/npm/@ip-location-db/geo-whois-asn-country-mmdb/geo-whois-asn-country.mmdb"
	CityDatabaseURL    = "https://cdn.jsdelivr.net/npm/@ip-location-db/dbip-city-mmdb/dbip-city-ipv4.mmdb"
)

// Config holds GeoIP configuration
type Config struct {
	Enabled       bool
	Dir           string
	DenyCountries []string
	ASNEnabled    bool
	CountryEnabled bool
	CityEnabled   bool
}

// DefaultConfig returns the default GeoIP configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		Dir:            "",
		DenyCountries:  []string{},
		ASNEnabled:     true,
		CountryEnabled: true,
		CityEnabled:    false,
	}
}

// Result represents a GeoIP lookup result
type Result struct {
	IP          string `json:"ip"`
	CountryCode string `json:"country_code,omitempty"`
	Country     string `json:"country,omitempty"`
	City        string `json:"city,omitempty"`
	Region      string `json:"region,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	ASN         uint   `json:"asn,omitempty"`
	ASNOrg      string `json:"asn_org,omitempty"`
	Blocked     bool   `json:"blocked"`
}

// MMDB record types for deserialization
type countryRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

type asnRecord struct {
	AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// Client handles GeoIP lookups
type Client struct {
	config     *Config
	enabled    bool
	lastUpdate time.Time
	denySet    map[string]bool
	countryDB  *maxminddb.Reader
	asnDB      *maxminddb.Reader
	mu         sync.RWMutex
}

// NewClient creates a new GeoIP client
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Build deny set for fast lookup
	denySet := make(map[string]bool)
	for _, code := range cfg.DenyCountries {
		denySet[code] = true
	}

	return &Client{
		config:  cfg,
		enabled: cfg.Enabled,
		denySet: denySet,
	}
}

// IsEnabled returns true if GeoIP is enabled
func (c *Client) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// LoadDatabase opens the MMDB database files
func (c *Client) LoadDatabase() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.Dir == "" {
		return fmt.Errorf("GeoIP directory not configured")
	}

	// Close existing readers
	if c.countryDB != nil {
		c.countryDB.Close()
		c.countryDB = nil
	}
	if c.asnDB != nil {
		c.asnDB.Close()
		c.asnDB = nil
	}

	// Open country database
	if c.config.CountryEnabled {
		countryPath := filepath.Join(c.config.Dir, "country.mmdb")
		if _, err := os.Stat(countryPath); err == nil {
			db, err := maxminddb.Open(countryPath)
			if err != nil {
				return fmt.Errorf("failed to open country database: %w", err)
			}
			c.countryDB = db
		}
	}

	// Open ASN database
	if c.config.ASNEnabled {
		asnPath := filepath.Join(c.config.Dir, "asn.mmdb")
		if _, err := os.Stat(asnPath); err == nil {
			db, err := maxminddb.Open(asnPath)
			if err != nil {
				return fmt.Errorf("failed to open ASN database: %w", err)
			}
			c.asnDB = db
		}
	}

	return nil
}

// Close closes the MMDB database readers
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.countryDB != nil {
		c.countryDB.Close()
		c.countryDB = nil
	}
	if c.asnDB != nil {
		c.asnDB.Close()
		c.asnDB = nil
	}
}

// SetEnabled enables or disables GeoIP
func (c *Client) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// SetDenyCountries sets the list of denied country codes
func (c *Client) SetDenyCountries(codes []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.denySet = make(map[string]bool)
	for _, code := range codes {
		c.denySet[code] = true
	}
	c.config.DenyCountries = codes
}

// IsCountryDenied checks if a country code is in the deny list
func (c *Client) IsCountryDenied(code string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.denySet[code]
}

// Lookup performs a GeoIP lookup for an IP address
func (c *Client) Lookup(ipStr string) (*Result, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("GeoIP is disabled")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	result := &Result{
		IP: ipStr,
	}

	// Check if IP is private/localhost
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		result.CountryCode = "XX"
		result.Country = "Private Network"
		return result, nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Country lookup
	if c.countryDB != nil {
		var record countryRecord
		err := c.countryDB.Lookup(ip, &record)
		if err == nil {
			result.CountryCode = record.Country.ISOCode
			if name, ok := record.Country.Names["en"]; ok {
				result.Country = name
			}
		}
	}

	// ASN lookup
	if c.asnDB != nil {
		var record asnRecord
		err := c.asnDB.Lookup(ip, &record)
		if err == nil {
			result.ASN = record.AutonomousSystemNumber
			result.ASNOrg = record.AutonomousSystemOrganization
		}
	}

	// Default if no database matched
	if result.CountryCode == "" {
		result.CountryCode = "XX"
		result.Country = "Unknown"
	}

	// Check deny list
	result.Blocked = c.denySet[result.CountryCode]

	return result, nil
}

// LookupRequest extracts IP from HTTP request and performs lookup
func (c *Client) LookupRequest(r *http.Request) (*Result, error) {
	ip := GetClientIP(r)
	return c.Lookup(ip)
}

// GetClientIP extracts the client IP from an HTTP request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// UpdateDatabases downloads the latest GeoIP databases
func (c *Client) UpdateDatabases() error {
	if c.config.Dir == "" {
		return fmt.Errorf("GeoIP directory not configured")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(c.config.Dir, 0755); err != nil {
		return fmt.Errorf("failed to create GeoIP directory: %w", err)
	}

	var errs []error

	// Download ASN database
	if c.config.ASNEnabled {
		if err := downloadFile(ASNDatabaseURL, filepath.Join(c.config.Dir, "asn.mmdb")); err != nil {
			errs = append(errs, fmt.Errorf("ASN database: %w", err))
		}
	}

	// Download Country database
	if c.config.CountryEnabled {
		if err := downloadFile(CountryDatabaseURL, filepath.Join(c.config.Dir, "country.mmdb")); err != nil {
			errs = append(errs, fmt.Errorf("country database: %w", err))
		}
	}

	// Download City database
	if c.config.CityEnabled {
		if err := downloadFile(CityDatabaseURL, filepath.Join(c.config.Dir, "city.mmdb")); err != nil {
			errs = append(errs, fmt.Errorf("city database: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("database update errors: %v", errs)
	}

	c.mu.Lock()
	c.lastUpdate = time.Now()
	c.mu.Unlock()

	// Reload databases after update
	return c.LoadDatabase()
}

// downloadFile downloads a file from URL to the specified path
func downloadFile(url, destPath string) error {
	// Create temp file
	tmpPath := destPath + ".tmp"

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// GetLastUpdate returns the last database update time
func (c *Client) GetLastUpdate() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdate
}

// GetConfig returns the current configuration (for display)
func (c *Client) GetConfig() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"enabled":        c.enabled,
		"dir":            c.config.Dir,
		"deny_countries": c.config.DenyCountries,
		"asn_enabled":    c.config.ASNEnabled,
		"country_enabled": c.config.CountryEnabled,
		"city_enabled":   c.config.CityEnabled,
		"last_update":    c.lastUpdate,
	}
}
