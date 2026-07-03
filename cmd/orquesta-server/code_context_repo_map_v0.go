package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	orquestacontext "orquesta/modulos/orquesta-context"
)

func (provider serverRGCodeContextProviderV0) queryRepoMapCodeContextV0(
	ctx context.Context,
	root string,
	command string,
	query orquestacontext.CodeContextQueryV0,
) (orquestacontext.CodeContextResultV0, error) {
	if _, err := exec.LookPath(command); err != nil {
		hits, fallbackErr := serverGoRepoMapContextHitsV0(root, query)
		if fallbackErr != nil {
			return orquestacontext.CodeContextResultV0{}, fallbackErr
		}
		return serverRGCodeContextResultWithEvidenceV0(
			query,
			hits,
			[]string{"evidence-ref-code-context-go-repo-map-fallback-v0"},
		), nil
	}
	cmd := exec.CommandContext(ctx, command, serverRGRepoMapContextArgsV0(query)...)
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
	return serverRGCodeContextResultWithEvidenceV0(
		query,
		serverRGRepoMapContextHitsV0(stdout.String(), query),
		[]string{"evidence-ref-code-context-rg-repo-map-central-v0"},
	), nil
}

func serverRGRepoMapContextArgsV0(query orquestacontext.CodeContextQueryV0) []string {
	args := []string{
		"--line-number",
		"--no-heading",
		"--color", "never",
		"--smart-case",
		"--glob", "*.go",
		"-e", `^\s*(func|type)\s+`,
		"--",
	}
	scopes := serverRGCodeContextScopesV0(query.Scope)
	if len(scopes) == 0 {
		scopes = []string{"cmd", "modulos"}
	}
	args = append(args, scopes...)
	return args
}

func serverGoRepoMapContextHitsV0(root string, query orquestacontext.CodeContextQueryV0) ([]orquestacontext.CodeContextHitV0, error) {
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
		scopes = []string{"cmd", "modulos"}
	}
	tokens := serverRepoMapQueryTokensV0(query.Query)
	matches := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	fallback := make([]orquestacontext.CodeContextHitV0, 0, maxResults)
	for _, scope := range scopes {
		if len(matches) >= maxResults {
			break
		}
		target := filepath.Join(root, filepath.Clean(scope))
		if !serverCodeContextPathWithinRootV0(root, target) {
			continue
		}
		err := filepath.WalkDir(target, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || entry == nil {
				return nil
			}
			if len(matches) >= maxResults {
				return filepath.SkipAll
			}
			if entry.IsDir() {
				if serverGoCodeContextSkipDirV0(entry.Name()) && path != target {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(entry.Name(), ".go") || !serverCodeContextPathWithinRootV0(root, path) {
				return nil
			}
			fileHits, err := serverGoRepoMapContextFileHitsV0(root, path, query)
			if err != nil {
				return nil
			}
			for _, hit := range fileHits {
				if len(fallback) < maxResults {
					fallback = append(fallback, hit)
				}
				if serverRepoMapHitMatchesTokensV0(hit, tokens) {
					matches = append(matches, hit)
					if len(matches) >= maxResults {
						return filepath.SkipAll
					}
				}
			}
			return nil
		})
		if err != nil && !errors.Is(err, filepath.SkipAll) {
			return nil, err
		}
	}
	if len(matches) > 0 {
		return serverRepoMapNormalizeHitRefsV0(matches), nil
	}
	return serverRepoMapNormalizeHitRefsV0(fallback), nil
}

func serverGoRepoMapContextFileHitsV0(
	root string,
	path string,
	query orquestacontext.CodeContextQueryV0,
) ([]orquestacontext.CodeContextHitV0, error) {
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
	hits := make([]orquestacontext.CodeContextHitV0, 0)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		kind, symbol, summary, ok := parseServerRepoMapDeclarationV0(line)
		if !ok {
			continue
		}
		hits = append(hits, orquestacontext.CodeContextHitV0{
			Kind:    kind,
			Path:    rel,
			Line:    lineNo,
			Symbol:  symbol,
			Summary: summary,
			Snippet: serverGoCodeContextSnippetV0(line),
			Score:   1,
		})
	}
	return hits, nil
}
