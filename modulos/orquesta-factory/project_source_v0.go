package orquestafactory

import (
	"net/url"
	"strings"
)

const (
	ProjectSourceKindNewV0       = "new"
	ProjectSourceKindGitHubV0    = "github"
	ProjectSourceKindLocalPathV0 = "local_path"
	DefaultProjectSourceKindV0   = ProjectSourceKindNewV0
)

type ProjectSourceSpecV0 struct {
	Kind       string `json:"kind"`
	GitURL     string `json:"git_url,omitempty"`
	Branch     string `json:"branch,omitempty"`
	LocalPath  string `json:"local_path,omitempty"`
	ProjectRef string `json:"project_ref,omitempty"`
}

func NormalizeProjectSourceKindV0(value string, source ProjectSourceRequestV0) string {
	kind := strings.ToLower(strings.TrimSpace(value))
	if kind != "" {
		return kind
	}
	if strings.TrimSpace(source.GitURL) != "" {
		return ProjectSourceKindGitHubV0
	}
	if strings.TrimSpace(source.LocalPath) != "" {
		return ProjectSourceKindLocalPathV0
	}
	return DefaultProjectSourceKindV0
}

func ProjectSourceKindSupportedV0(value string) bool {
	return containsV0(value, ProjectSourceKindNewV0, ProjectSourceKindGitHubV0, ProjectSourceKindLocalPathV0)
}

func normalizeProjectSourceV0(source ProjectSourceRequestV0) ProjectSourceSpecV0 {
	return ProjectSourceSpecV0{
		Kind:       NormalizeProjectSourceKindV0(source.Kind, source),
		GitURL:     strings.TrimSpace(source.GitURL),
		Branch:     strings.TrimSpace(source.Branch),
		LocalPath:  strings.TrimSpace(source.LocalPath),
		ProjectRef: strings.TrimSpace(source.ProjectRef),
	}
}

func validateProjectSourceV0(source ProjectSourceRequestV0) []ValidationIssue {
	spec := normalizeProjectSourceV0(source)
	var issues []ValidationIssue
	if !ProjectSourceKindSupportedV0(spec.Kind) {
		issues = append(issues, issue(ErrAppSpecInvalida, "project_source.kind", "origen de proyecto no soportado"))
		return issues
	}
	if spec.GitURL != "" && spec.LocalPath != "" {
		issues = append(issues, issue(ErrOpcionIncompatible, "project_source", "elige github o ruta local, no ambos"))
	}
	switch spec.Kind {
	case ProjectSourceKindGitHubV0:
		issues = append(issues, validateGitHubProjectSourceV0(spec)...)
	case ProjectSourceKindLocalPathV0:
		if spec.LocalPath == "" {
			issues = append(issues, issue(ErrAppSpecInvalida, "project_source.local_path", "ruta local requerida"))
		}
	case ProjectSourceKindNewV0:
		if spec.GitURL != "" || spec.LocalPath != "" {
			issues = append(issues, issue(ErrOpcionIncompatible, "project_source.kind", "new no admite git_url ni local_path"))
		}
	}
	return issues
}

func validateRequestKindProjectSourceV0(req AppSpecRequestV0) []ValidationIssue {
	if !RequestKindNeedsExistingProjectSourceV0(req.RequestKind) {
		return nil
	}
	source := normalizeProjectSourceV0(req.ProjectSource)
	if source.Kind != ProjectSourceKindNewV0 {
		return nil
	}
	return []ValidationIssue{issue(ErrAppSpecInvalida, "project_source.kind", "este tipo de peticion requiere github o ruta local")}
}

func RequestKindNeedsExistingProjectSourceV0(kind string) bool {
	switch NormalizeRequestKindV0(kind) {
	case RequestKindAnalizarAppV0,
		RequestKindModificarAppExistenteV0,
		RequestKindRevisarCodigoV0,
		RequestKindSeguridadV0,
		RequestKindMigracionRefactorV0:
		return true
	default:
		return false
	}
}

func validateGitHubProjectSourceV0(source ProjectSourceSpecV0) []ValidationIssue {
	if source.GitURL == "" {
		return []ValidationIssue{issue(ErrAppSpecInvalida, "project_source.git_url", "URL de GitHub requerida")}
	}
	if projectSourceURLHasCredentialsV0(source.GitURL) {
		return []ValidationIssue{issue(ErrOpcionIncompatible, "project_source.git_url", "no incluyas credenciales en la URL")}
	}
	return nil
}

func projectSourceURLHasCredentialsV0(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed.User != nil
}
