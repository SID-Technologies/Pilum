package webhook

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/sid-technologies/pilum/lib/errors"
)

// privateRanges contains CIDR ranges that should be blocked for webhook URLs.
var privateRanges = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16", // link-local / cloud metadata
	"127.0.0.0/8",
	"::1/128",
	"fc00::/7",  // IPv6 unique local
	"fe80::/10", // IPv6 link-local
}

var parsedPrivateRanges []*net.IPNet

func init() {
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid CIDR in privateRanges: %s", cidr))
		}
		parsedPrivateRanges = append(parsedPrivateRanges, network)
	}
}

// isPrivateIP returns true if the IP falls within any blocked CIDR range.
func isPrivateIP(ip net.IP) bool {
	for _, network := range parsedPrivateRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// validateWebhookURL checks that a webhook URL is safe to send requests to.
// It requires HTTPS (except for localhost/127.0.0.1 for local testing),
// and blocks requests to literal private/reserved IP addresses.
// DNS-level checks (rebinding protection) are handled by safeTransport in webhook.go.
func validateWebhookURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.Wrap(err, "invalid URL")
	}

	// Require http or https scheme
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return errors.New("unsupported scheme %q (only https and http are allowed)", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return errors.New("url has no hostname")
	}

	// Allow http only for localhost
	isLocalhost := hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
	if scheme == "http" && !isLocalhost {
		return errors.New("http is only allowed for localhost (use https for remote URLs)")
	}

	// Skip IP checks for localhost (allow local testing)
	if isLocalhost {
		return nil
	}

	// Block literal private IP addresses
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return errors.New("hostname %q resolves to private IP %s", hostname, ip)
		}
	}

	return nil
}
