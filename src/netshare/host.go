
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package netshare

import (
	"net"
	"net/http"
	"strings"

	"github.com/casjay-forks/caspaste/src/validation"
)

func GetHost(req *http.Request) string {
	// Try RFC 7239 Forwarded header first
	forwarded := req.Header.Get("Forwarded")
	if forwarded != "" {
		// Parse "Forwarded: for=192.0.2.60;proto=http;host=example.com"
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "host=") {
				host := strings.TrimPrefix(part, "host=")
				host = strings.Trim(host, "\"")
				if host != "" {
					return host
				}
			}
		}
	}

	// X-Forwarded-Host (common reverse proxy header)
	xHost := req.Header.Get("X-Forwarded-Host")
	if xHost != "" {
		// Take the first host if multiple are listed
		return strings.Split(xHost, ",")[0]
	}

	// Fallback to request Host header
	return req.Host
}

func GetProtocol(req *http.Request) string {
	// Try RFC 7239 Forwarded header first
	forwarded := req.Header.Get("Forwarded")
	if forwarded != "" {
		// Parse "Forwarded: for=192.0.2.60;proto=http;host=example.com"
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "proto=") {
				proto := strings.TrimPrefix(part, "proto=")
				proto = strings.Trim(proto, "\"")
				if proto != "" {
					return proto
				}
			}
		}
	}

	// X-Forwarded-Proto (common reverse proxy header)
	xProto := req.Header.Get("X-Forwarded-Proto")
	if xProto != "" {
		// Take the first protocol if multiple are listed
		return strings.Split(xProto, ",")[0]
	}

	// X-Forwarded-Ssl (some proxies use this: on/off)
	xSsl := req.Header.Get("X-Forwarded-Ssl")
	if validation.IsTruthy(xSsl) {
		return "https"
	}

	// X-Scheme (alternative header used by some proxies)
	xScheme := req.Header.Get("X-Scheme")
	if xScheme != "" {
		return xScheme
	}

	// Check if request came over TLS
	if req.TLS != nil {
		return "https"
	}

	// Check URL scheme if available
	if req.URL.Scheme != "" {
		return req.URL.Scheme
	}

	// Default to http
	return "http"
}

// BuildPasteURL constructs the full URL for a paste
// Format: {proto}://{fqdn}:{port}/{pasteID}
// Strips port if it's 80 (http) or 443 (https)
func BuildPasteURL(req *http.Request, pasteID string) string {
	proto := GetProtocol(req)
	host := GetHost(req)

	// Strip standard ports (80 for http, 443 for https)
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		if len(parts) == 2 {
			port := parts[1]
			// Strip port 80 for http or port 443 for https
			if (proto == "http" && port == "80") || (proto == "https" && port == "443") {
				host = parts[0]
			}
		}
	}

	return proto + "://" + host + "/" + pasteID
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// IPv4 private ranges:
	// - 10.0.0.0/8
	// - 172.16.0.0/12
	// - 192.168.0.0/16
	// - 127.0.0.0/8 (localhost)
	privateIPv4Ranges := []struct {
		min net.IP
		max net.IP
	}{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
		{net.ParseIP("127.0.0.0"), net.ParseIP("127.255.255.255")},
	}

	// Check IPv4
	if ip.To4() != nil {
		for _, r := range privateIPv4Ranges {
			if bytesInRange(ip.To4(), r.min.To4(), r.max.To4()) {
				return true
			}
		}
		return false
	}

	// IPv6 private ranges
	// fc00::/7 (Unique Local Addresses)
	// fe80::/10 (Link-Local)
	// ::1/128 (Localhost)
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return true
	}

	// Check fc00::/7 (ULA)
	if len(ip) == 16 && (ip[0]&0xfe) == 0xfc {
		return true
	}

	return false
}

// bytesInRange checks if IP is within min-max range
func bytesInRange(ip, min, max net.IP) bool {
	if len(ip) != len(min) || len(ip) != len(max) {
		return false
	}
	for i := range ip {
		if ip[i] < min[i] || ip[i] > max[i] {
			return false
		}
	}
	return true
}

// GetClientAddrTrusted extracts client IP address from request
// Set trustProxy=true to always trust proxy headers
// If trustProxy=false (default), only trusts proxy headers from private IPs
// This provides security by default while supporting common reverse proxy setups
func GetClientAddrTrusted(req *http.Request, trustProxy bool) net.IP {
	// Get the direct connection IP
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return nil
	}
	remoteIP := net.ParseIP(host)

	// Determine if we should trust proxy headers
	shouldTrust := trustProxy || isPrivateIP(remoteIP)

	// If not trusting proxy headers, use direct connection IP only
	if !shouldTrust {
		return remoteIP
	}

	// Try RFC 7239 Forwarded header first
	forwarded := req.Header.Get("Forwarded")
	if forwarded != "" {
		// Parse "Forwarded: for=192.0.2.60;proto=http;host=example.com"
		parts := strings.Split(forwarded, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "for=") {
				forVal := strings.TrimPrefix(part, "for=")
				forVal = strings.Trim(forVal, "\"")
				// Remove port if present (e.g., "192.0.2.60:47011" or "[2001:db8::1]:47011")
				if strings.Contains(forVal, "]:") {
					// IPv6 with port: [2001:db8::1]:47011
					forVal = strings.TrimPrefix(forVal, "[")
					forVal = strings.Split(forVal, "]:")[0]
				} else if strings.Contains(forVal, ":") && strings.Count(forVal, ":") == 1 {
					// IPv4 with port: 192.0.2.60:47011
					forVal = strings.Split(forVal, ":")[0]
				}
				if ip := net.ParseIP(forVal); ip != nil {
					return ip
				}
			}
		}
	}

	// X-Real-IP (common in nginx)
	xReal := req.Header.Get("X-Real-IP")
	if xReal != "" {
		if ip := net.ParseIP(strings.TrimSpace(xReal)); ip != nil {
			return ip
		}
	}

	// X-Forwarded-For (common in many proxies, takes the first IP)
	xFor := req.Header.Get("X-Forwarded-For")
	if xFor != "" {
		// Take the first IP from the list
		firstIP := strings.TrimSpace(strings.Split(xFor, ",")[0])
		if ip := net.ParseIP(firstIP); ip != nil {
			return ip
		}
	}

	// CF-Connecting-IP (Cloudflare specific)
	cfIP := req.Header.Get("CF-Connecting-IP")
	if cfIP != "" {
		if ip := net.ParseIP(strings.TrimSpace(cfIP)); ip != nil {
			return ip
		}
	}

	// True-Client-IP (Akamai and Cloudflare)
	trueIP := req.Header.Get("True-Client-IP")
	if trueIP != "" {
		if ip := net.ParseIP(strings.TrimSpace(trueIP)); ip != nil {
			return ip
		}
	}

	// Fallback: use remote IP we already extracted
	return remoteIP
}

// GetClientAddr extracts client IP address using direct connection only
// This is the safe default - use GetClientAddrTrusted when behind a reverse proxy
func GetClientAddr(req *http.Request) net.IP {
	return GetClientAddrTrusted(req, false)
}
