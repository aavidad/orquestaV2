package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
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
	return opesDrainDestinationPolicyFromProjectConfigFileV0(opesProjectConfigFromEnvBestEffortV0(), opesBaseURL, orquestaBaseURL, dryRun)
}

func opesDrainDestinationPolicyFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
	opesBaseURL string,
	orquestaBaseURL string,
	dryRun bool,
) (opesDrainDestinationPolicyV0, error) {
	if opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESBridgeProductiveConfirmV0, false) {
		return opesDrainDestinationPolicyV0{}, fmt.Errorf("opes_destination_productive_not_allowed")
	}
	opes, err := opesBridgeDestinationFromProjectConfigFileV0(config, "opes", opesBaseURL, dryRun)
	if err != nil {
		return opesDrainDestinationPolicyV0{}, err
	}
	orquesta, err := opesBridgeDestinationFromProjectConfigFileV0(config, "orquesta", orquestaBaseURL, dryRun)
	if err != nil {
		return opesDrainDestinationPolicyV0{}, err
	}
	if err := opesBridgeRequireRealOPESConfirmationFromProjectConfigFileV0(config, opes, dryRun); err != nil {
		return opesDrainDestinationPolicyV0{}, err
	}
	evidenceRef := opesBridgeStringValueFromProjectConfigFileV0(config, envOPESBridgeDestinationEvidenceV0)
	if evidenceRef != "" && !compactEvidenceRefV0(evidenceRef) {
		return opesDrainDestinationPolicyV0{}, fmt.Errorf("opes_destination_evidence_ref_invalid")
	}
	if opes.Category == "temporal" && evidenceRef == "" {
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
	return opesBridgeDestinationFromProjectConfigFileV0(opesProjectConfigFromEnvBestEffortV0(), kind, raw, dryRun)
}

func opesBridgeDestinationFromProjectConfigFileV0(
	config serverProjectConfigFileV0,
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
	category, err := opesBridgeDestinationCategoryFromProjectConfigFileV0(config, parsed)
	if err != nil {
		return opesDrainDestinationV0{}, err
	}
	return opesDrainDestinationV0{
		Category: category,
		URLRef:   commandPublicOpaqueRefV0(kind+"-base-url", raw),
	}, nil
}

func opesBridgeDestinationCategoryV0(parsed *url.URL) (string, error) {
	return opesBridgeDestinationCategoryFromProjectConfigFileV0(opesProjectConfigFromEnvBestEffortV0(), parsed)
}

func opesBridgeDestinationCategoryFromProjectConfigFileV0(config serverProjectConfigFileV0, parsed *url.URL) (string, error) {
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", fmt.Errorf("opes_destination_host_required")
	}
	if opesBridgeHostIsLoopbackV0(host) {
		return "loopback", nil
	}
	if opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESTemporalConfirmV0, false) {
		return "temporal", nil
	}
	return "", fmt.Errorf("opes_destination_confirmation_required")
}

func opesBridgeHostIsLoopbackV0(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func opesBridgeRequireRealOPESConfirmationV0(destination opesDrainDestinationV0, dryRun bool) error {
	return opesBridgeRequireRealOPESConfirmationFromProjectConfigFileV0(opesProjectConfigFromEnvBestEffortV0(), destination, dryRun)
}

func opesBridgeRequireRealOPESConfirmationFromProjectConfigFileV0(config serverProjectConfigFileV0, destination opesDrainDestinationV0, dryRun bool) error {
	if dryRun || destination.Category == "dry_run" {
		return nil
	}
	if opesBridgeBoolValueFromProjectConfigFileV0(config, envOPESTemporalConfirmV0, false) {
		return nil
	}
	return fmt.Errorf("opes_destination_confirmation_required")
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
