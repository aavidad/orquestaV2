package orquestaappcodexstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const (
	domainWorkArtifactDefaultMaxBytesV0 = 256 * 1024
	domainWorkArtifactJSONMaxBytesV0    = 512 * 1024
)

type domainWorkDeliveryArtifactIntakeV0 struct {
	Body        string
	ContentKind string
}

func readDomainWorkDeliveryArtifactV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	artifactType string,
) (domainWorkDeliveryArtifactIntakeV0, error) {
	for _, file := range ack.Files {
		path, ok := safeDomainWorkDeliveryFilePathV0(descriptor.ProjectWorkDir, string(file))
		if !ok {
			return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
				"domain_work_artifact_unreadable",
				"files.path",
				"invalid_declared_path",
			)
		}
		return readDomainWorkDeliveryArtifactFileV0(path, artifactType)
	}
	return domainWorkDeliveryArtifactIntakeV0{}, nil
}

func readDomainWorkDeliveryArtifactFileV0(
	path string,
	artifactType string,
) (domainWorkDeliveryArtifactIntakeV0, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_unreadable",
			"files.path",
			"artifact_file_unreadable",
		)
	}
	limit := domainWorkDeliveryArtifactMaxBytesV0(artifactType)
	file, err := os.Open(path)
	if err != nil {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_unreadable",
			"files.path",
			"artifact_file_unreadable",
		)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_unreadable",
			"files.path",
			"artifact_file_unreadable",
		)
	}
	if len(data) > limit {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_too_large",
			"files.content",
			"artifact_limit_exceeded",
		)
	}
	if domainWorkDeliveryArtifactLooksBinaryV0(data) {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_binary_requires_attachment",
			"files.content",
			"binary_payload_must_use_ref",
		)
	}
	body := strings.TrimSpace(string(data))
	kind, err := domainWorkDeliveryArtifactContentKindV0(body)
	if err != nil {
		return domainWorkDeliveryArtifactIntakeV0{}, err
	}
	return domainWorkDeliveryArtifactIntakeV0{Body: body, ContentKind: kind}, nil
}

func safeDomainWorkDeliveryFilePathV0(baseDir string, rel string) (string, bool) {
	baseDir = strings.TrimSpace(baseDir)
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if baseDir == "" || rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") ||
		filepath.IsAbs(rel) {
		return "", false
	}
	path := filepath.Join(baseDir, filepath.FromSlash(rel))
	cleanBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", false
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	if cleanPath != cleanBase && !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) {
		return "", false
	}
	return cleanPath, true
}

func domainWorkDeliveryArtifactMaxBytesV0(artifactType string) int {
	switch strings.TrimSpace(artifactType) {
	case orquestadomainwork.DomainDocumentPlanArtifactTypeV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicExpansionPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeAssembledTopicV0:
		return domainWorkArtifactJSONMaxBytesV0
	default:
		return domainWorkArtifactDefaultMaxBytesV0
	}
}

func domainWorkDeliveryArtifactContentKindV0(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "text", nil
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if !json.Valid([]byte(trimmed)) {
			return "", domainWorkArtifactIntakeErrorV0(
				"domain_work_artifact_unreadable",
				"files.content",
				"invalid_structured_json",
			)
		}
		return "json", nil
	}
	return "text", nil
}

func domainWorkDeliveryArtifactLooksBinaryV0(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return true
	}
	control := 0
	for _, b := range data {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			control++
		}
	}
	return control > 0 && control*20 > len(data)
}

func validateDomainWorkArtifactPayloadFieldsForIntakeV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) error {
	for _, field := range fields {
		if domainWorkArtifactFieldNameRequiresAttachmentV0(field.Name) {
			return domainWorkArtifactIntakeErrorV0(
				"domain_work_artifact_binary_requires_attachment",
				"payload_fields."+strings.TrimSpace(field.Name),
				"binary_payload_must_use_ref",
			)
		}
		if domainWorkArtifactFieldSensitiveV0(field) {
			return domainWorkArtifactIntakeErrorV0(
				"domain_work_artifact_payload_sensitive",
				"payload_fields."+strings.TrimSpace(field.Name),
				"sensitive_payload_blocked",
			)
		}
	}
	return nil
}

func domainWorkArtifactFieldSensitiveV0(field orquestadomainwork.DomainWorkFieldV0) bool {
	if domainWorkArtifactSensitiveFieldNameV0(field.Name) {
		return true
	}
	if domainWorkArtifactSensitiveTextV0(field.Value) || domainWorkArtifactSensitiveTextV0(string(field.ValueJSON)) {
		return true
	}
	for _, value := range field.Values {
		if domainWorkArtifactSensitiveTextV0(value) {
			return true
		}
	}
	return false
}

func domainWorkArtifactSensitiveFieldNameV0(name string) bool {
	switch normalizeDomainWorkDeliveryAliasV0(name) {
	case "access_token", "refresh_token", "api_key", "secret", "client_secret",
		"password", "credential", "prompt", "completion", "transcript",
		"raw_http", "http_request", "http_response", "provider_payload",
		"provider_response", "model_payload":
		return true
	default:
		return false
	}
}

func domainWorkArtifactFieldNameRequiresAttachmentV0(name string) bool {
	switch normalizeDomainWorkDeliveryAliasV0(name) {
	case "image_data_uri", "data_uri", "binary", "binary_payload", "file_bytes":
		return true
	default:
		return false
	}
}

func domainWorkArtifactSensitiveTextV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, fragment := range []string{
		"/home/", "/users/", "\\users\\", "$home", "~/", "bearer ",
		"access_token=", "refresh_token=", "api_key=", "client_secret=",
		"secret=", "password=", "authorization:", "sk-", "\"prompt\"",
		"prompt=", "\"transcript\"", "transcript=",
	} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return domainWorkArtifactRawHTTPPayloadV0(lower)
}

func domainWorkArtifactRawHTTPPayloadV0(lower string) bool {
	return strings.HasPrefix(lower, "http/1.") ||
		((strings.HasPrefix(lower, "get ") || strings.HasPrefix(lower, "post ") ||
			strings.HasPrefix(lower, "put ") || strings.HasPrefix(lower, "delete ")) &&
			strings.Contains(lower, "\nhost:"))
}

func domainWorkArtifactIntakeErrorV0(code, field, reason string) error {
	return fmt.Errorf("%s field=%s reason=%s", code, field, reason)
}
