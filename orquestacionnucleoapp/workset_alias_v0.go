package orquestacionnucleoapp

import (
	"path/filepath"
	"strconv"
	"strings"
)

const (
	WorksetAliasIssueEmptyAliasV0   = "alias_vacio"
	WorksetAliasIssueUnknownAliasV0 = "alias_desconocido"
)

type WorksetAliasIssueV0 struct {
	Code    string
	Field   string
	Alias   string
	Message string
}

func ResolveWorksetAliasesV0(
	allowedAliases map[string][]string,
	requestedAliases []string,
) ([]string, []WorksetAliasIssueV0) {
	var paths []string
	var issues []WorksetAliasIssueV0
	seen := make(map[string]struct{})

	for index, rawAlias := range requestedAliases {
		alias := strings.TrimSpace(rawAlias)
		if alias == "" {
			issues = append(issues, worksetAliasIssueV0(
				WorksetAliasIssueEmptyAliasV0,
				worksetAliasIssueFieldV0(index),
				"",
				"alias vacio",
			))
			continue
		}

		aliasPaths, ok := allowedAliases[alias]
		if !ok {
			issues = append(issues, worksetAliasIssueV0(
				WorksetAliasIssueUnknownAliasV0,
				worksetAliasIssueFieldV0(index),
				alias,
				"alias desconocido",
			))
			continue
		}

		paths = appendWorksetAliasPathsV0(paths, seen, aliasPaths)
	}

	return paths, issues
}

func appendWorksetAliasPathsV0(
	paths []string,
	seen map[string]struct{},
	rawPaths []string,
) []string {
	for _, rawPath := range rawPaths {
		path := normalizeWorksetAliasPathV0(rawPath)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func normalizeWorksetAliasPathV0(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func worksetAliasIssueV0(code string, field string, alias string, message string) WorksetAliasIssueV0 {
	return WorksetAliasIssueV0{
		Code:    code,
		Field:   field,
		Alias:   alias,
		Message: message,
	}
}

func worksetAliasIssueFieldV0(index int) string {
	return "requested_aliases[" + strconv.Itoa(index) + "]"
}
