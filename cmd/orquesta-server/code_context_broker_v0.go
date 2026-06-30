package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func codeContextBrokerFromEnvV0(config orquestaserver.ConfigV0) orquestacontext.CodeContextQueryPortV0 {
	providerKind := envOrDefaultV0(envCodebaseBrokerProviderKindV0, orquestacontext.CodeContextProviderKindFallbackRGV0)
	var provider orquestacontext.CodeContextProviderPortV0
	if providerKind == orquestacontext.CodeContextProviderKindFallbackRGV0 &&
		strings.TrimSpace(config.ProjectWorkDir) != "" {
		provider = serverRGCodeContextProviderV0{
			RootDir: strings.TrimSpace(config.ProjectWorkDir),
			Command: "rg",
		}
	}
	return orquestacontext.NewCodeContextBrokerV0(orquestacontext.CodeContextBrokerConfigV0{
		Provider:               provider,
		Cache:                  orquestacontext.NewInMemoryCodeContextCacheV0(),
		ProviderRef:            "provider-ref-orquesta-code-context-central",
		ProviderKind:           providerKind,
		ExternalIndexerEnabled: boolEnvOrDefaultV0(envCodebaseBrokerExternalIndexerEnabledV0, false),
		MaxConcurrent:          intEnvOrDefaultV0(envCodebaseBrokerMaxConcurrentV0, 4),
		Timeout:                time.Duration(intEnvOrDefaultV0(envCodebaseBrokerTimeoutMSV0, 3000)) * time.Millisecond,
	})
}

type serverRGCodeContextProviderV0 struct {
	RootDir string
	Command string
}

func (provider serverRGCodeContextProviderV0) QueryCodeContextV0(
	ctx context.Context,
	query orquestacontext.CodeContextQueryV0,
) (orquestacontext.CodeContextResultV0, error) {
	root := strings.TrimSpace(provider.RootDir)
	if root == "" {
		return orquestacontext.CodeContextResultV0{}, errors.New("code_context_root_required")
	}
	command := strings.TrimSpace(provider.Command)
	if command == "" {
		command = "rg"
	}
	args := serverRGCodeContextArgsV0(query)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = root
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return serverRGCodeContextResultV0(query, nil), nil
		}
		return orquestacontext.CodeContextResultV0{}, err
	}
	return serverRGCodeContextResultV0(query, serverRGCodeContextHitsV0(stdout.String(), query)), nil
}

func serverRGCodeContextArgsV0(query orquestacontext.CodeContextQueryV0) []string {
	args := []string{
		"--line-number",
		"--no-heading",
		"--color", "never",
		"--smart-case",
		"-F",
		"--",
		strings.TrimSpace(query.Query),
	}
	scopes := serverRGCodeContextScopesV0(query.Scope)
	if len(scopes) == 0 {
		scopes = []string{"cmd", "modulos", "docs", "scripts", "AGENTS.md"}
	}
	args = append(args, scopes...)
	return args
}

func serverRGCodeContextScopesV0(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = filepath.Clean(strings.TrimSpace(scope))
		if scope == "." || scope == "" || strings.HasPrefix(scope, "..") || filepath.IsAbs(scope) {
			continue
		}
		out = append(out, scope)
	}
	return out
}

func serverRGCodeContextHitsV0(output string, query orquestacontext.CodeContextQueryV0) []orquestacontext.CodeContextHitV0 {
	lines := strings.Split(output, "\n")
	maxResults := query.MaxResults
	if maxResults <= 0 {
		maxResults = 8
	}
	hits := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, line := range lines {
		if len(hits) >= maxResults {
			break
		}
		path, lineNo, snippet, ok := parseRGCodeContextLineV0(line)
		if !ok {
			continue
		}
		hits = append(hits, orquestacontext.CodeContextHitV0{
			HitRef:  "code-context-rg-hit-" + strconv.Itoa(len(hits)+1),
			Kind:    "text_match",
			Path:    path,
			Line:    lineNo,
			Summary: "coincidencia literal servida por rg central",
			Snippet: snippet,
			Score:   1,
		})
	}
	return hits
}

func parseRGCodeContextLineV0(line string) (string, int, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", 0, "", false
	}
	first := strings.Index(line, ":")
	if first <= 0 {
		return "", 0, "", false
	}
	second := strings.Index(line[first+1:], ":")
	if second < 0 {
		return "", 0, "", false
	}
	second += first + 1
	lineNo, err := strconv.Atoi(line[first+1 : second])
	if err != nil {
		return "", 0, "", false
	}
	return line[:first], lineNo, strings.TrimSpace(line[second+1:]), true
}

func serverRGCodeContextResultV0(
	query orquestacontext.CodeContextQueryV0,
	hits []orquestacontext.CodeContextHitV0,
) orquestacontext.CodeContextResultV0 {
	return orquestacontext.CodeContextResultV0{
		SchemaVersion: orquestacontext.CodeContextResultSchemaVersionV0,
		Estado:        orquestacontext.CodeContextEstadoOKV0,
		RequestRef:    query.RequestRef,
		CorrelationID: query.CorrelationID,
		RepositoryRef: query.RepositoryRef,
		WorktreeRef:   query.WorktreeRef,
		CommitRef:     query.CommitRef,
		QueryKind:     query.QueryKind,
		ProviderRef:   "provider-ref-rg-central",
		ProviderKind:  orquestacontext.CodeContextProviderKindFallbackRGV0,
		IndexerPolicy: orquestacontext.CodeContextProviderPolicyCentralOnlyV0,
		Results:       hits,
		EvidenceRefs:  []string{"evidence-ref-code-context-rg-central-v0"},
	}
}
