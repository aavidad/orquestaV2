// Este fichero define el formato sellado y su resumen determinista.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
	"time"
)

const (
	manifestSchemaVersion = 1
	manifestAlgorithm     = "orquesta.legacy-source-manifest.v1"
	sourceSealAlgorithm   = "orquesta.legacy-source.v1"
	defaultGitTimeout     = 30 * time.Second
)

type options struct {
	roots        []string
	excluded     []string
	manifestPath string
	gitTimeout   time.Duration
}

type manifest struct {
	SchemaVersion  int            `json:"schema_version"`
	Algorithm      string         `json:"algorithm"`
	Roots          []rootRecord   `json:"roots"`
	ExcludedPaths  []string       `json:"excluded_paths"`
	Sources        []sourceRecord `json:"sources"`
	Summary        summary        `json:"summary"`
	ManifestSHA256 string         `json:"manifest_sha256"`
}

type rootRecord struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	ErrorCode string `json:"error_code,omitempty"`
}

type sourceRecord struct {
	Path                      string      `json:"path"`
	Kind                      string      `json:"kind"`
	Status                    string      `json:"status"`
	Reason                    string      `json:"reason,omitempty"`
	ErrorCode                 string      `json:"error_code,omitempty"`
	GitDirectory              string      `json:"git_directory,omitempty"`
	GitCommonDirectory        string      `json:"git_common_directory,omitempty"`
	ObjectFormat              string      `json:"object_format,omitempty"`
	HeadReference             string      `json:"head_reference,omitempty"`
	HeadObject                string      `json:"head_object,omitempty"`
	References                []reference `json:"references,omitempty"`
	ReferenceSHA256           string      `json:"reference_sha256,omitempty"`
	ReachableCommitCount      *int        `json:"reachable_commit_count,omitempty"`
	WorktreeState             string      `json:"worktree_state,omitempty"`
	WorktreeStatusSHA256      string      `json:"worktree_status_sha256,omitempty"`
	WorktreeChangeCount       int         `json:"worktree_change_count,omitempty"`
	RequiresPhysicalInventory bool        `json:"requires_physical_inventory,omitempty"`
	SizeBytes                 int64       `json:"size_bytes,omitempty"`
	ContentSHA256             string      `json:"content_sha256,omitempty"`
	SourceSHA256              string      `json:"source_sha256"`
}

type reference struct {
	Name         string `json:"name"`
	Object       string `json:"object"`
	ObjectType   string `json:"object_type,omitempty"`
	PeeledObject string `json:"peeled_object,omitempty"`
	Symbolic     string `json:"symbolic,omitempty"`
}

type summary struct {
	TotalSources             int `json:"total_sources"`
	IncludedSources          int `json:"included_sources"`
	ExcludedSources          int `json:"excluded_sources"`
	SourcesWithErrors        int `json:"sources_with_errors"`
	Repositories             int `json:"repositories"`
	Worktrees                int `json:"worktrees"`
	BareRepositories         int `json:"bare_repositories"`
	Bundles                  int `json:"bundles"`
	SymbolicLinksNotFollowed int `json:"symbolic_links_not_followed"`
	DirtyWorktrees           int `json:"dirty_worktrees"`
	UnassessedWorktrees      int `json:"unassessed_worktrees"`
	PhysicalInventoriesDue   int `json:"physical_inventories_due"`
	References               int `json:"references"`
	RootsWithErrors          int `json:"roots_with_errors"`
}

type collector struct {
	sources map[string]sourceRecord
}

func sealSource(source sourceRecord) sourceRecord {
	source.SourceSHA256 = ""
	source.SourceSHA256 = digestJSON(sourceSealAlgorithm, source)
	return source
}

func digestJSON(domain string, value any) string {
	content, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return digestBytes(domain, content)
}

func digestBytes(domain string, content []byte) string {
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, domain)
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (collected *collector) add(source sourceRecord) {
	previous, exists := collected.sources[source.Path]
	if !exists || sourcePriority(source) > sourcePriority(previous) ||
		(sourcePriority(source) == sourcePriority(previous) && source.SourceSHA256 < previous.SourceSHA256) {
		collected.sources[source.Path] = source
	}
}

func sourcePriority(source sourceRecord) int {
	switch source.Status {
	case "error":
		return 3
	case "excluded":
		return 2
	case "included":
		return 1
	default:
		return 0
	}
}

func (collected *collector) sorted() []sourceRecord {
	result := make([]sourceRecord, 0, len(collected.sources))
	for _, source := range collected.sources {
		result = append(result, source)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Path != result[right].Path {
			return result[left].Path < result[right].Path
		}
		return result[left].Kind < result[right].Kind
	})
	return result
}

func summarize(roots []rootRecord, sources []sourceRecord) summary {
	var result summary
	result.TotalSources = len(sources)
	for _, root := range roots {
		if root.Status == "error" {
			result.RootsWithErrors++
		}
	}
	for _, source := range sources {
		switch source.Status {
		case "included":
			result.IncludedSources++
		case "excluded":
			result.ExcludedSources++
		case "error":
			result.SourcesWithErrors++
		}
		switch source.Kind {
		case "git_repository":
			result.Repositories++
		case "git_worktree":
			result.Worktrees++
		case "git_bare_repository":
			result.BareRepositories++
		case "git_bundle":
			result.Bundles++
		case "symbolic_link":
			result.SymbolicLinksNotFollowed++
		}
		switch source.WorktreeState {
		case "dirty":
			result.DirtyWorktrees++
		case "dirty_state_unassessed":
			result.UnassessedWorktrees++
		}
		if source.RequiresPhysicalInventory {
			result.PhysicalInventoriesDue++
		}
		result.References += len(source.References)
	}
	return result
}

func marshalManifest(value manifest) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
