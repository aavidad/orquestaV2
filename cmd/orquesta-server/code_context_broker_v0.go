package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaserver "orquesta/modulos/orquesta-server"
)

type codeContextBrokerWiringV0 struct {
	Query             orquestacontext.CodeContextQueryPortV0
	ToolLeases        orquestacontext.CodeContextToolLeaseListPortV0
	ToolOwnerObserver serverCodeContextToolOwnerObserverV0
}

func codeContextBrokerWiringFromEnvV0(
	config orquestaserver.ConfigV0,
	projectConfigs ...serverProjectConfigFileV0,
) codeContextBrokerWiringV0 {
	projectConfig := serverProjectConfigFileV0{}
	if len(projectConfigs) > 0 {
		projectConfig = projectConfigs[0]
	} else {
		projectConfig = projectConfigFromServerConfigBestEffortV0(config)
	}
	providerKind := stringProjectConfigOrEnvOrDefaultV0(
		envCodebaseBrokerProviderKindV0,
		projectConfig.CodebaseBroker.ProviderKind,
		orquestacontext.CodeContextProviderKindFallbackRGV0,
	)
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
	var ownerObserver serverCodeContextToolOwnerObserverV0
	stateDir := codebaseBrokerStateDirFromProjectConfigFileV0(projectConfig)
	if stateDir != "" {
		ownerRegistry := newServerFileCodeContextToolOwnerRegistryV0(stateDir)
		fileLeaseStore := newServerFileCodeContextToolLeaseStoreV0(filepath.Join(stateDir, serverCodeContextLeasesFileV0))
		leaseStore = fileLeaseStore
		cache = newServerFileCodeContextCacheV0(filepath.Join(stateDir, serverCodeContextCacheFileV0))
		ownerObserver = serverFileCodeContextToolOwnerObserverV0{Registry: ownerRegistry}
		if providerKind == orquestacontext.CodeContextProviderKindCodebaseMCPV0 {
			leasePort = fileLeaseStore
			provider = serverCodebaseMemoryCLIProviderV0{
				RootDir:     strings.TrimSpace(config.ProjectWorkDir),
				ProjectName: serverCodebaseMemoryProjectNameV0(config.ProjectWorkDir, codebaseBrokerProjectNameFromProjectConfigFileV0(projectConfig)),
				Command:     codebaseBrokerCommandFromProjectConfigFileV0(projectConfig),
				Registry:    ownerRegistry,
			}
		}
	}
	return codeContextBrokerWiringV0{
		Query: orquestacontext.NewCodeContextBrokerV0(orquestacontext.CodeContextBrokerConfigV0{
			Provider:               provider,
			Cache:                  cache,
			ProviderRef:            "provider-ref-orquesta-code-context-central",
			ProviderKind:           providerKind,
			ExternalIndexerEnabled: codebaseBrokerExternalIndexerEnabledFromProjectConfigFileV0(projectConfig),
			MaxConcurrent:          codebaseBrokerMaxConcurrentFromProjectConfigFileV0(projectConfig),
			Timeout:                codebaseBrokerTimeoutFromProjectConfigFileV0(projectConfig),
			ToolLeasePort:          leasePort,
		}),
		ToolLeases:        leaseStore,
		ToolOwnerObserver: ownerObserver,
	}
}

type serverCodebaseMemoryCLIProviderV0 struct {
	RootDir     string
	ProjectName string
	Command     string
	Registry    serverFileCodeContextToolOwnerRegistryV0
	Clock       func() time.Time
}

func (provider serverCodebaseMemoryCLIProviderV0) QueryCodeContextV0(
	ctx context.Context,
	query orquestacontext.CodeContextQueryV0,
) (orquestacontext.CodeContextResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	projectName := strings.TrimSpace(provider.ProjectName)
	if projectName == "" {
		return orquestacontext.CodeContextResultV0{}, errors.New("codebase_memory_project_required")
	}
	toolName, payload, err := serverCodebaseMemoryCLIToolPayloadV0(projectName, query)
	if err != nil {
		return orquestacontext.CodeContextResultV0{}, err
	}
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return orquestacontext.CodeContextResultV0{}, err
	}
	command := strings.TrimSpace(provider.Command)
	if command == "" {
		command = "codebase-memory-mcp"
	}
	cmd := exec.CommandContext(ctx, command, "cli", toolName, string(payloadData))
	if root := strings.TrimSpace(provider.RootDir); root != "" {
		cmd.Dir = root
	}
	configureDetachedProcessV0(cmd)
	outputLimit := serverRGCodeContextOutputLimitV0(query)
	stdout := serverLimitedBufferV0{maxBytes: outputLimit}
	stderr := serverLimitedBufferV0{maxBytes: outputLimit}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return orquestacontext.CodeContextResultV0{}, err
	}
	if err := provider.writeOwnerMarkerV0(query, cmd.Process.Pid); err != nil {
		_ = signalProcessGroupKillV0(cmd.Process.Pid)
		_ = cmd.Wait()
		return orquestacontext.CodeContextResultV0{}, err
	}
	err = cmd.Wait()
	rawJSON, ok := serverCodebaseMemoryJSONLineV0(stdout.String(), stderr.String())
	if err != nil && !ok {
		return orquestacontext.CodeContextResultV0{}, err
	}
	if !ok {
		return orquestacontext.CodeContextResultV0{}, errors.New("codebase_memory_cli_json_missing")
	}
	result, parseErr := serverCodebaseMemoryCLIResultV0(query, toolName, rawJSON)
	if parseErr != nil {
		return orquestacontext.CodeContextResultV0{}, parseErr
	}
	return result, nil
}

func (provider serverCodebaseMemoryCLIProviderV0) writeOwnerMarkerV0(
	query orquestacontext.CodeContextQueryV0,
	pid int,
) error {
	if strings.TrimSpace(provider.Registry.dir) == "" || pid <= 0 {
		return nil
	}
	now := time.Now().UTC()
	if provider.Clock != nil {
		now = provider.Clock().UTC()
	}
	return provider.Registry.WriteCodeContextToolOwnerMarkerV0(serverCodeContextToolOwnerMarkerV0{
		OwnerRef:        orquestacontext.CodeContextToolOwnerRefV0(query),
		ToolRef:         "provider-ref-orquesta-code-context-central",
		ProviderKind:    orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		PID:             pid,
		StartedAt:       now.Format(time.RFC3339),
		LastHeartbeatAt: now.Format(time.RFC3339),
		ActiveRequests:  1,
		EvidenceRefs:    []string{"evidence-ref-codebase-memory-cli-owner"},
	})
}

func serverCodebaseMemoryCLIToolPayloadV0(
	projectName string,
	query orquestacontext.CodeContextQueryV0,
) (string, map[string]any, error) {
	switch query.QueryKind {
	case orquestacontext.CodeContextQueryKindArchitectureV0:
		return "get_architecture", map[string]any{"project": projectName}, nil
	case orquestacontext.CodeContextQueryKindSearchV0,
		orquestacontext.CodeContextQueryKindSymbolV0,
		orquestacontext.CodeContextQueryKindRepoMapV0,
		orquestacontext.CodeContextQueryKindCallersV0,
		orquestacontext.CodeContextQueryKindImportsV0,
		orquestacontext.CodeContextQueryKindModuleExportsV0,
		orquestacontext.CodeContextQueryKindRelevantSnippetsV0,
		"":
		limit := query.MaxResults
		if limit <= 0 {
			limit = 8
		}
		return "search_graph", map[string]any{
			"project": projectName,
			"query":   query.Query,
			"limit":   limit,
		}, nil
	default:
		return "", nil, errors.New("codebase_memory_query_kind_unsupported")
	}
}

func serverCodebaseMemoryCLIResultV0(
	query orquestacontext.CodeContextQueryV0,
	toolName string,
	rawJSON string,
) (orquestacontext.CodeContextResultV0, error) {
	switch toolName {
	case "search_graph":
		return serverCodebaseMemorySearchGraphResultV0(query, rawJSON)
	case "get_architecture":
		return serverCodebaseMemoryArchitectureResultV0(query, rawJSON)
	default:
		return orquestacontext.CodeContextResultV0{}, errors.New("codebase_memory_tool_unsupported")
	}
}

func serverCodebaseMemorySearchGraphResultV0(
	query orquestacontext.CodeContextQueryV0,
	rawJSON string,
) (orquestacontext.CodeContextResultV0, error) {
	var payload struct {
		Results []struct {
			Name          string  `json:"name"`
			QualifiedName string  `json:"qualified_name"`
			Label         string  `json:"label"`
			FilePath      string  `json:"file_path"`
			StartLine     int     `json:"start_line"`
			Rank          float64 `json:"rank"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return orquestacontext.CodeContextResultV0{}, err
	}
	hits := make([]orquestacontext.CodeContextHitV0, 0, len(payload.Results))
	for idx, item := range payload.Results {
		hits = append(hits, orquestacontext.CodeContextHitV0{
			HitRef:  "code-context-codebase-hit-" + strconv.Itoa(idx+1),
			Kind:    strings.TrimSpace(item.Label),
			Path:    strings.TrimSpace(item.FilePath),
			Line:    item.StartLine,
			Symbol:  strings.TrimSpace(item.Name),
			Summary: strings.TrimSpace(item.QualifiedName),
			Score:   item.Rank,
		})
	}
	return serverCodebaseMemoryResultV0(query, hits, []string{"evidence-ref-codebase-memory-cli-search-graph"}), nil
}

func serverCodebaseMemoryArchitectureResultV0(
	query orquestacontext.CodeContextQueryV0,
	rawJSON string,
) (orquestacontext.CodeContextResultV0, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return orquestacontext.CodeContextResultV0{}, err
	}
	snippet := strings.TrimSpace(rawJSON)
	return serverCodebaseMemoryResultV0(query, []orquestacontext.CodeContextHitV0{{
		HitRef:  "code-context-codebase-architecture-1",
		Kind:    "architecture",
		Summary: "resumen de arquitectura servido por codebase-memory-mcp cli",
		Snippet: snippet,
	}}, []string{"evidence-ref-codebase-memory-cli-architecture"}), nil
}

func serverCodebaseMemoryResultV0(
	query orquestacontext.CodeContextQueryV0,
	hits []orquestacontext.CodeContextHitV0,
	evidence []string,
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
		ProviderRef:   "provider-ref-orquesta-code-context-central",
		ProviderKind:  orquestacontext.CodeContextProviderKindCodebaseMCPV0,
		IndexerPolicy: orquestacontext.CodeContextProviderPolicyCentralOnlyV0,
		Results:       hits,
		EvidenceRefs:  evidence,
	}
}

func serverCodebaseMemoryJSONLineV0(outputs ...string) (string, bool) {
	for idx := len(outputs) - 1; idx >= 0; idx-- {
		lines := strings.Split(outputs[idx], "\n")
		for lineIdx := len(lines) - 1; lineIdx >= 0; lineIdx-- {
			line := strings.TrimSpace(lines[lineIdx])
			if line == "" || !strings.HasPrefix(line, "{") || !json.Valid([]byte(line)) {
				continue
			}
			return line, true
		}
	}
	return "", false
}

func serverCodebaseMemoryProjectNameV0(root string, override string) string {
	if value := strings.TrimSpace(override); value != "" {
		return value
	}
	cleaned := filepath.Clean(strings.TrimSpace(root))
	cleaned = strings.Trim(cleaned, `/\`)
	if cleaned == "." || cleaned == "" {
		return ""
	}
	replacer := strings.NewReplacer("/", "-", `\`, "-")
	return strings.Trim(replacer.Replace(cleaned), "-")
}

type serverRGCodeContextProviderV0 struct {
	RootDir string
	Command string
}

const (
	defaultServerRGCodeContextOutputMaxBytesV0 = 262144
	minServerRGCodeContextOutputMaxBytesV0     = 65536
	maxServerGoCodeContextSnippetBytesV0       = 700
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
	if query.QueryKind == orquestacontext.CodeContextQueryKindRepoMapV0 {
		return provider.queryRepoMapCodeContextV0(ctx, root, command, query)
	}
	switch query.QueryKind {
	case orquestacontext.CodeContextQueryKindCallersV0,
		orquestacontext.CodeContextQueryKindImportsV0,
		orquestacontext.CodeContextQueryKindModuleExportsV0,
		orquestacontext.CodeContextQueryKindRelevantSnippetsV0:
		hits, err := serverStructuredCodeContextHitsV0(root, query)
		if err != nil {
			return orquestacontext.CodeContextResultV0{}, err
		}
		return serverRGCodeContextResultWithEvidenceV0(
			query,
			hits,
			[]string{"evidence-ref-code-context-go-structured-fallback-v0"},
		), nil
	}
	if _, err := exec.LookPath(command); err != nil {
		hits, fallbackErr := serverGoCodeContextHitsV0(root, query)
		if fallbackErr != nil {
			return orquestacontext.CodeContextResultV0{}, fallbackErr
		}
		return serverRGCodeContextResultWithEvidenceV0(
			query,
			hits,
			[]string{"evidence-ref-code-context-go-fallback-v0"},
		), nil
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
		if scope == "" || strings.HasPrefix(scope, "..") || filepath.IsAbs(scope) {
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

func serverGoCodeContextHitsV0(root string, query orquestacontext.CodeContextQueryV0) ([]orquestacontext.CodeContextHitV0, error) {
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return nil, err
	}
	maxResults := query.MaxResults
	if maxResults <= 0 {
		maxResults = 8
	}
	scopes := serverRGCodeContextScopesV0(query.Scope)
	if len(scopes) == 0 {
		scopes = []string{"cmd", "modulos", "docs", "scripts", "AGENTS.md"}
	}
	hits := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, scope := range scopes {
		if len(hits) >= maxResults {
			break
		}
		target := filepath.Join(root, filepath.Clean(scope))
		if !serverCodeContextPathWithinRootV0(root, target) {
			continue
		}
		if err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil {
				return nil
			}
			if len(hits) >= maxResults {
				return filepath.SkipAll
			}
			if entry.IsDir() {
				if serverGoCodeContextSkipDirV0(entry.Name()) && path != target {
					return filepath.SkipDir
				}
				return nil
			}
			if !serverCodeContextPathWithinRootV0(root, path) {
				return nil
			}
			fileHits, err := serverGoCodeContextFileHitsV0(root, path, query, maxResults-len(hits))
			if err != nil {
				return nil
			}
			hits = append(hits, fileHits...)
			if len(hits) >= maxResults {
				return filepath.SkipAll
			}
			return nil
		}); err != nil && !errors.Is(err, filepath.SkipAll) {
			return nil, err
		}
	}
	return serverGoCodeContextNormalizeHitRefsV0(hits), nil
}

func serverGoCodeContextFileHitsV0(
	root string,
	path string,
	query orquestacontext.CodeContextQueryV0,
	limit int,
) ([]orquestacontext.CodeContextHitV0, error) {
	if limit <= 0 {
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > int64(serverRGCodeContextOutputLimitV0(query)) {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer file.Close()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	rel = filepath.ToSlash(filepath.Clean(rel))
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)
	hits := make([]orquestacontext.CodeContextHitV0, 0, limit)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if !serverCodeContextSmartCaseMatchV0(line, query.Query) {
			continue
		}
		hits = append(hits, orquestacontext.CodeContextHitV0{
			HitRef:  "code-context-go-hit-" + strconv.Itoa(len(hits)+1),
			Kind:    "text_match",
			Path:    rel,
			Line:    lineNo,
			Summary: "coincidencia literal servida por fallback Go central",
			Snippet: serverGoCodeContextSnippetV0(line),
			Score:   1,
		})
		if len(hits) >= limit {
			break
		}
	}
	return hits, nil
}

func serverGoCodeContextNormalizeHitRefsV0(
	hits []orquestacontext.CodeContextHitV0,
) []orquestacontext.CodeContextHitV0 {
	for idx := range hits {
		hits[idx].HitRef = "code-context-go-hit-" + strconv.Itoa(idx+1)
	}
	return hits
}

func serverCodeContextSmartCaseMatchV0(line string, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	if query == strings.ToLower(query) {
		return strings.Contains(strings.ToLower(line), query)
	}
	return strings.Contains(line, query)
}

func serverGoCodeContextSnippetV0(line string) string {
	line = strings.TrimSpace(line)
	if len(line) <= maxServerGoCodeContextSnippetBytesV0 {
		return line
	}
	return strings.TrimSpace(line[:maxServerGoCodeContextSnippetBytesV0])
}

func serverGoCodeContextSkipDirV0(name string) bool {
	switch name {
	case ".git", ".orquesta-runtime", "node_modules", "runtime", "go-cache", "go-build":
		return true
	default:
		return false
	}
}

func serverCodeContextPathWithinRootV0(root string, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
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
	return serverRGCodeContextResultWithEvidenceV0(
		query,
		hits,
		[]string{"evidence-ref-code-context-rg-central-v0"},
	)
}

func serverRGCodeContextResultWithEvidenceV0(
	query orquestacontext.CodeContextQueryV0,
	hits []orquestacontext.CodeContextHitV0,
	evidence []string,
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
		EvidenceRefs:  evidence,
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
