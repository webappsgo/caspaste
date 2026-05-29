// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package web

import (
	"github.com/casjay-forks/caspaste/src/netshare"
	"io"
	"net/http"
	"strings"
)

func (data *Data) handleRobotsTxt(rw http.ResponseWriter, req *http.Request) error {
	var robotsTxt strings.Builder

	// User-agent: * rules
	robotsTxt.WriteString("User-agent: *\n")

	// Allow directive
	if data.SiteRobotsAllow != "" {
		robotsTxt.WriteString("Allow: " + data.SiteRobotsAllow + "\n")
	}

	// Disallow directive(s)
	if data.SiteRobotsDeny != "" {
		for _, path := range strings.Split(data.SiteRobotsDeny, ",") {
			robotsTxt.WriteString("Disallow: " + strings.TrimSpace(path) + "\n")
		}
	}

	// Add sitemap
	proto := netshare.GetProtocol(req)
	host := netshare.GetHost(req)
	robotsTxt.WriteString("Sitemap: " + proto + "://" + host + "/sitemap.xml\n")

	// Block AI bots individually
	for _, agent := range data.SiteRobotsAgentsDeny {
		robotsTxt.WriteString("\nUser-agent: " + agent + "\n")
		robotsTxt.WriteString("Disallow: /\n")
	}

	// Write response
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := io.WriteString(rw, robotsTxt.String())
	if err != nil {
		return err
	}

	return nil
}

func (data *Data) handleSitemap(rw http.ResponseWriter, req *http.Request) error {
	// Check if sitemap is allowed
	if data.SiteRobotsDeny == "/" {
		return netshare.ErrNotFound
	}

	// Get protocol and host
	proto := netshare.GetProtocol(req)
	host := netshare.GetHost(req)

	// Generate sitemap.xml
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>`
	sitemapXML = sitemapXML + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n"
	sitemapXML = sitemapXML + "<url><loc>" + proto + "://" + host + "/" + "</loc></url>\n"
	sitemapXML = sitemapXML + "<url><loc>" + proto + "://" + host + "/server/about" + "</loc></url>\n"
	sitemapXML = sitemapXML + "<url><loc>" + proto + "://" + host + "/docs/apiv1" + "</loc></url>\n"
	sitemapXML = sitemapXML + "<url><loc>" + proto + "://" + host + "/docs/libraries" + "</loc></url>\n"
	sitemapXML = sitemapXML + "</urlset>\n"

	// Write response
	rw.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, err := io.WriteString(rw, sitemapXML)
	if err != nil {
		return err
	}

	return nil
}
