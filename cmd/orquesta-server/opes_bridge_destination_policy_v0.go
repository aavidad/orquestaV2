package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

type opesDrainDestinationPolicyV0 struct {
	OPESDestination        opesDrainDestinationV0
	OrquestaDestination    opesDrainDestinationV0
	DestinationEvidenceRef string
}

type opesDrainDestinationV0 struct {
	Category string
	URLRef   string
}

func opesDrainDestinationPolicyFromEnvV0(
	opesBaseURL string,
	orquestaBaseURL string,
	dryRun bool,
) (opesDrainDestinationPolicyV0, error) {
	opes, err := opesBridgeDestinationFromURLV0("opes", opesBaseURL, dryRun)
	if err != nil {
		return opesDrainDestinationPolicyV0{}, err
	}
	orquesta, err := opesBridgeDestinationFromURLV0("orquesta", orquestaBaseURL, dryRun)
	if err != nil {
		return opesDrainDestinationPolicyV0{}, err
	}
	evidenceRef := strings.TrimSpace(os.Getenv(envOPESBridgeDestinationEvidenceV0))
	if needsProductiveEvidenceV0(opes, orquesta) && !compactEvidenceRefV0(evidenceRef) {
		return opesDrainDestinationPolicyV0{}, fmt.Errorf("opes_destination_evidence_ref_required")
	}
	if evidenceRef == "" {
		evidenceRef = opesBridgeDestinationEvidenceRefV0(opes.Category + "|" + orquesta.Category)
	}
	return opesDrainDestinationPolicyV0{
		OPESDestination:        opes,
		OrquestaDestination:    orquesta,
		DestinationEvidenceRef: evidenceRef,
	}, nil
}

func opesBridgeDestinationFromURLV0(
	kind string,
	raw string,
	dryRun bool,
) (opesDrainDestinationV0, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return opesDrainDestinationV0{}, fmt.Errorf("%s_destination_required", kind)
	}
	if dryRun && raw == "dry-run" {
		return opesDrainDestinationV0{Category: "dry_run", URLRef: commandPublicOpaqueRefV0(kind+"-base-url", raw)}, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return opesDrainDestinationV0{}, fmt.Errorf("%s_destination_invalid", kind)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return opesDrainDestinationV0{}, fmt.Errorf("%s_destination_scheme_not_allowed", kind)
	}
	if parsed.User != nil {
		return opesDrainDestinationV0{}, fmt.Errorf("%s_destination_credentials_not_allowed", kind)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return opesDrainDestinationV0{}, fmt.Errorf("%s_destination_query_not_allowed", kind)
	}
	category, err := opesBridgeDestinationCategoryV0(parsed)
	if err != nil {
		return opesDrainDestinationV0{}, err
	}
	return opesDrainDestinationV0{
		Category: category,
		URLRef:   commandPublicOpaqueRefV0(kind+"-base-url", raw),
	}, nil
}

func opesBridgeDestinationCategoryV0(parsed *url.URL) (string, error) {
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", fmt.Errorf("opes_destination_host_required")
	}
	if opesBridgeHostIsLoopbackV0(host) {
		return "loopback", nil
	}
	if strings.TrimSpace(os.Getenv(envOPESBridgeProductiveConfirmV0)) == "1" {
		return "productive", nil
	}
	if strings.TrimSpace(os.Getenv(envOPESTemporalConfirmV0)) == "1" {
		return "temporal", nil
	}
	return "", fmt.Errorf("opes_destination_confirmation_required")
}

func opesBridgeHostIsLoopbackV0(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

func needsProductiveEvidenceV0(destinations ...opesDrainDestinationV0) bool {
	for _, destination := range destinations {
		if destination.Category == "productive" {
			return true
		}
	}
	return false
}

func compactEvidenceRefV0(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > 96 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func opesBridgeDestinationEvidenceRefV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "opes-destination-evidence-ref-" + hex.EncodeToString(sum[:])[:16]
}
