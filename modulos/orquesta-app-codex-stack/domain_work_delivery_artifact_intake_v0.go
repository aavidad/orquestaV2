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

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type domainWorkDeliveryArtifactIntakeV0 struct {
	Body        string
	ContentKind string
	FileRef     string
}

func readDomainWorkDeliveryArtifactV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
	artifactType string,
) (domainWorkDeliveryArtifactIntakeV0, error) {
	path, fileRef, ok, err := selectDomainWorkDeliveryArtifactFileV0(
		descriptor.ProjectWorkDir,
		ack.Files,
		artifactType,
	)
	if err != nil || !ok {
		return domainWorkDeliveryArtifactIntakeV0{}, err
	}
	return readDomainWorkDeliveryArtifactFileV0(path, fileRef, artifactType)
}

func selectDomainWorkDeliveryArtifactFileV0(
	projectWorkDir string,
	files orquestaruntimecodex.EvidenceListV0,
	artifactType string,
) (string, string, bool, error) {
	fallbackPath := ""
	fallbackRef := ""
	for _, file := range files {
		fileRef := strings.TrimSpace(string(file))
		path, ok := safeDomainWorkDeliveryFilePathV0(projectWorkDir, fileRef)
		if !ok {
			return "", "", false, domainWorkArtifactIntakeErrorV0(
				"domain_work_artifact_unreadable",
				"files.path",
				"invalid_declared_path",
			)
		}
		if fallbackPath == "" {
			fallbackPath = path
			fallbackRef = fileRef
		}
		if domainWorkDeliveryArtifactFileRefMatchesV0(fileRef, artifactType) {
			return path, fileRef, true, nil
		}
	}
	if fallbackPath == "" {
		return "", "", false, nil
	}
	return fallbackPath, fallbackRef, true, nil
}

func domainWorkDeliveryArtifactFileRefMatchesV0(fileRef string, artifactType string) bool {
	artifactType = strings.TrimSpace(artifactType)
	if artifactType == "" {
		return false
	}
	candidates := []string{fileRef}
	for _, segment := range strings.Split(filepath.ToSlash(fileRef), "/") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		candidates = append(candidates, segment)
		if ext := filepath.Ext(segment); ext != "" {
			candidates = append(candidates, strings.TrimSuffix(segment, ext))
		}
	}
	for _, candidate := range candidates {
		if domainWorkDeliveryArtifactTypeMatchesV0(candidate, artifactType) {
			return true
		}
	}
	return false
}

func readDomainWorkDeliveryArtifactFileV0(
	path string,
	fileRef string,
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
	file, err := os.Open(path)
	if err != nil {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_unreadable",
			"files.path",
			"artifact_file_unreadable",
		)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return domainWorkDeliveryArtifactIntakeV0{}, domainWorkArtifactIntakeErrorV0(
			"domain_work_artifact_unreadable",
			"files.path",
			"artifact_file_unreadable",
		)
	}
	if domainWorkDeliveryArtifactLooksBinaryV0(data) {
		return domainWorkDeliveryArtifactIntakeV0{
			ContentKind: "binary_file_ref",
			FileRef:     strings.TrimSpace(fileRef),
		}, nil
	}
	body := strings.TrimSpace(string(data))
	kind := domainWorkDeliveryArtifactContentKindV0(body)
	return domainWorkDeliveryArtifactIntakeV0{Body: body, ContentKind: kind, FileRef: strings.TrimSpace(fileRef)}, nil
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

func domainWorkDeliveryArtifactContentKindV0(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "text"
	}
	if (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) && json.Valid([]byte(trimmed)) {
		return "json"
	}
	return "text"
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

func domainWorkArtifactIntakeErrorV0(code, field, reason string) error {
	return fmt.Errorf("%s field=%s reason=%s", code, field, reason)
}
