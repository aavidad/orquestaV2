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

type codeContextBrokerWiringV0 struct {
	Query      orquestacontext.CodeContextQueryPortV0
	ToolLeases orquestacontext.CodeContextToolLeaseListPortV0
}

func codeContextBrokerFromEnvV0(config orquestaserver.ConfigV0) orquestacontext.CodeContextQueryPortV0 {
	return codeContextBrokerWiringFromEnvV0(config).Query
}

func codeContextBrokerWiringFromEnvV0(config orquestaserver.ConfigV0) codeContextBrokerWiringV0 {
	providerKind := envOrDefaultV0(envCodebaseBrokerProviderKindV0, orquestacontext.CodeContextProviderKindFallbackRGV0)
	var provider orquestacontext.CodeContextProviderPortV0
	if providerKind == orquestacontext.CodeContextProviderKindFallbackRGV0 &&
		strings.TrimSpace(config.ProjectWorkDir) != "" {
		provider = serverRGCodeContextProviderV0{
			RootDir: strings.TrimSpace(config.ProjectWorkDir),
			Command: "rg",
		}
	}
	memoryLeaseStore := orquestacontext.NewInMemoryCodeContextToolLeaseStoreV0()
	var leaseStore orquestacontext.CodeContextToolLeaseListPortV0 = memoryLeaseStore
	var leasePort orquestacontext.CodeContextToolLeasePortV0
	if providerKind == orquestacontext.CodeContextProviderKindCodebaseMCPV0 {
		leasePort = memoryLeaseStore
	}
	cache := orquestacontext.CodeContextCachePortV0(orquestacontext.NewInMemoryCodeContextCacheV0())
	stateDir := strings.TrimSpace(envOrDefaultV0(envCodebaseBrokerStateDirV0, ""))
	if stateDir != "" {
		fileLeaseStore := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
		leaseStore = fileLeaseStore
		cache = newServerFileCodeContextCacheV0(filepath.Join(stateDir, serverCodeContextCacheFileV0))
		if providerKind == orquestacontext.CodeContextProviderKindCodebaseMCPV0 {
			leasePort = fileLeaseStore
		}
	}
	return codeContextBrokerWiringV0{
		Query: orquestacontext.NewCodeContextBrokerV0(orquestacontext.CodeContextBrokerConfigV0{
			Provider:               provider,
			Cache:                  cache,
			ProviderRef:            "provider-ref-orquesta-code-context-central",
			ProviderKind:           providerKind,
			ExternalIndexerEnabled: boolEnvOrDefaultV0(envCodebaseBrokerExternalIndexerEnabledV0, false),
			MaxConcurrent:          intEnvOrDefaultV0(envCodebaseBrokerMaxConcurrentV0, 4),
			Timeout:                time.Duration(intEnvOrDefaultV0(envCodebaseBrokerTimeoutMSV0, 3000)) * time.Millisecond,
			ToolLeasePort:          leasePort,
		}),
		ToolLeases: leaseStore,
	}
}

type serverRGCodeContextProviderV0 struct {
	RootDir string
	Command string
}

const (
	defaultServerRGCodeContextOutputMaxBytesV0 = 262144
	minServerRGCodeContextOutputMaxBytesV0     = 65536
)

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
	stdout := serverLimitedBufferV0{maxBytes: serverRGCodeContextOutputLimitV0(query)}
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

func serverRGCodeContextOutputLimitV0(query orquestacontext.CodeContextQueryV0) int {
	if query.MaxBytes <= 0 {
		return defaultServerRGCodeContextOutputMaxBytesV0
	}
	limit := query.MaxBytes * 4
	if limit < minServerRGCodeContextOutputMaxBytesV0 {
		return minServerRGCodeContextOutputMaxBytesV0
	}
	if limit > defaultServerRGCodeContextOutputMaxBytesV0 {
		return defaultServerRGCodeContextOutputMaxBytesV0
	}
	return limit
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

type serverLimitedBufferV0 struct {
	buffer   bytes.Buffer
	maxBytes int
	overflow bool
}

func (buffer *serverLimitedBufferV0) Write(chunk []byte) (int, error) {
	if buffer.maxBytes <= 0 {
		buffer.maxBytes = defaultServerRGCodeContextOutputMaxBytesV0
	}
	remaining := buffer.maxBytes - buffer.buffer.Len()
	if remaining > 0 {
		if len(chunk) <= remaining {
			_, _ = buffer.buffer.Write(chunk)
		} else {
			_, _ = buffer.buffer.Write(chunk[:remaining])
			buffer.overflow = true
		}
	} else if len(chunk) > 0 {
		buffer.overflow = true
	}
	return len(chunk), nil
}

func (buffer *serverLimitedBufferV0) Len() int {
	if buffer == nil {
		return 0
	}
	return buffer.buffer.Len()
}

func (buffer *serverLimitedBufferV0) String() string {
	if buffer == nil {
		return ""
	}
	return buffer.buffer.String()
}
