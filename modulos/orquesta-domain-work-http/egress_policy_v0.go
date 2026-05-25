package orquestadomainworkhttp

import (
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

const (
	EgressModeSmokeLocalV0                = "smoke_local"
	EgressModeAllowlistV0                 = "allowlist"
	DestinationClassLoopbackTemporalV0    = "loopback_temporal"
	DestinationClassAllowlistedExternalV0 = "allowlisted_external"
	DestinationClassInternalNetworkV0     = "internal_network"
	DestinationClassMetadataServiceV0     = "metadata_service"
	DestinationClassExternalUnknownV0     = "external_unknown"
)

type EgressPolicyV0 struct {
	Mode         string
	AllowedHosts []string
}

type DestinationV0 struct {
	BaseURL        string
	Scheme         string
	Host           string
	Hostname       string
	Port           string
	PolicyMode     string
	Classification string
}

func SmokeLocalEgressPolicyV0() EgressPolicyV0 {
	return EgressPolicyV0{Mode: EgressModeSmokeLocalV0}
}

func AllowlistEgressPolicyV0(hosts ...string) EgressPolicyV0 {
	return EgressPolicyV0{Mode: EgressModeAllowlistV0, AllowedHosts: hosts}
}

func NormalizeDestinationV0(raw string, policy EgressPolicyV0) (DestinationV0, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return DestinationV0{}, ErrorV0{Code: ErrDomainWorkHTTPBaseURLRequiredV0}
	}
	parsed, err := url.Parse(raw)
	if err != nil || !validParsedBaseURLV0(parsed) {
		return DestinationV0{}, ErrorV0{Code: ErrDomainWorkHTTPBaseURLInvalidV0}
	}
	classification := classifyDestinationHostV0(parsed.Hostname())
	destination := DestinationV0{
		BaseURL:        raw,
		Scheme:         parsed.Scheme,
		Host:           strings.ToLower(parsed.Host),
		Hostname:       strings.ToLower(parsed.Hostname()),
		Port:           parsed.Port(),
		PolicyMode:     strings.TrimSpace(policy.Mode),
		Classification: classification,
	}
	if err := authorizeDestinationV0(destination, policy); err != nil {
		return DestinationV0{}, err
	}
	return destination, nil
}

func effectiveEgressPolicyV0(policy EgressPolicyV0) EgressPolicyV0 {
	if strings.TrimSpace(policy.Mode) == "" && len(policy.AllowedHosts) == 0 {
		return SmokeLocalEgressPolicyV0()
	}
	return policy
}

func validParsedBaseURLV0(parsed *url.URL) bool {
	if parsed == nil || parsed.User != nil || parsed.Scheme == "" || parsed.Host == "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	hostname := strings.TrimSpace(parsed.Hostname())
	if hostname == "" || strings.ContainsAny(hostname, " \t\r\n") {
		return false
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value <= 0 || value > 65535 {
			return false
		}
	}
	return true
}

func authorizeDestinationV0(destination DestinationV0, policy EgressPolicyV0) error {
	switch strings.TrimSpace(policy.Mode) {
	case EgressModeSmokeLocalV0:
		if destination.Classification == DestinationClassLoopbackTemporalV0 {
			return nil
		}
	case EgressModeAllowlistV0:
		if destinationAllowedByHostListV0(destination, policy.AllowedHosts) {
			return nil
		}
	case "":
		return ErrorV0{Code: ErrDomainWorkHTTPEgressPolicyRequiredV0}
	default:
		return ErrorV0{Code: ErrDomainWorkHTTPEgressPolicyRequiredV0}
	}
	return ErrorV0{Code: ErrDomainWorkHTTPEgressDestinationDeniedV0}
}

func classifyDestinationHostV0(hostname string) string {
	host := strings.ToLower(strings.TrimSpace(hostname))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return DestinationClassLoopbackTemporalV0
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		if host == "metadata.google.internal" {
			return DestinationClassMetadataServiceV0
		}
		return DestinationClassExternalUnknownV0
	}
	if addr.IsLoopback() {
		return DestinationClassLoopbackTemporalV0
	}
	if isMetadataAddressV0(addr) {
		return DestinationClassMetadataServiceV0
	}
	if addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified() {
		return DestinationClassInternalNetworkV0
	}
	return DestinationClassExternalUnknownV0
}

func isMetadataAddressV0(addr netip.Addr) bool {
	return addr == netip.MustParseAddr("169.254.169.254") ||
		addr == netip.MustParseAddr("fd00:ec2::254")
}

func destinationAllowedByHostListV0(destination DestinationV0, allowedHosts []string) bool {
	for _, raw := range allowedHosts {
		allowed := strings.ToLower(strings.TrimSpace(raw))
		if allowed == "" || strings.ContainsAny(allowed, "/?#@") {
			continue
		}
		if strings.Contains(allowed, ":") {
			host, port, err := net.SplitHostPort(allowed)
			if err == nil && strings.EqualFold(host, destination.Hostname) && port == destination.Port {
				return true
			}
			if allowed == destination.Host {
				return true
			}
			continue
		}
		if allowed == destination.Hostname {
			return true
		}
	}
	return false
}
