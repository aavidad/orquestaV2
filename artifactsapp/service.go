package artifactsapp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Scope string

const (
	ScopeTask    Scope = "task"
	ScopeAgent   Scope = "agent"
	ScopeProject Scope = "project"
)

type Kind string

const (
	KindPatch      Kind = "patch"
	KindTests      Kind = "tests"
	KindTranscript Kind = "transcript"
)

type Artifact struct {
	ID             string    `json:"artifact_id"`
	Scope          Scope     `json:"scope"`
	Kind           Kind      `json:"kind"`
	Version        int       `json:"version"`
	TaskID         *int64    `json:"task_id,omitempty"`
	Agent          string    `json:"agent,omitempty"`
	Project        string    `json:"project,omitempty"`
	ContentType    string    `json:"content_type"`
	Path           string    `json:"path,omitempty"`
	BlobRef        string    `json:"blob_ref,omitempty"`
	ChecksumSHA256 string    `json:"checksum_sha256,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	MetadataJSON   string    `json:"metadata_json,omitempty"`
}

type RegisterInput struct {
	Scope          Scope
	Kind           Kind
	TaskID         *int64
	Agent          string
	Project        string
	ContentType    string
	Path           string
	BlobRef        string
	ChecksumSHA256 string
	CreatedAt      time.Time
	MetadataJSON   string
}

type ListFilter struct {
	Scope   Scope
	Kind    Kind
	TaskID  *int64
	Agent   string
	Project string
}

type Store interface {
	SaveArtifact(artifact Artifact) error
	ListArtifacts(filter ListFilter) ([]Artifact, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   time.Now,
	}
}

func NewServiceWithClock(store Store, now func() time.Time) *Service {
	s := NewService(store)
	if now != nil {
		s.now = now
	}
	return s
}

func (s *Service) Register(input RegisterInput) (*Artifact, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("store de artifacts obligatorio")
	}
	artifact := Artifact{
		Scope:          normalizeScope(input.Scope),
		Kind:           normalizeKind(input.Kind),
		TaskID:         positiveTaskID(input.TaskID),
		Agent:          strings.TrimSpace(input.Agent),
		Project:        strings.TrimSpace(input.Project),
		ContentType:    strings.TrimSpace(input.ContentType),
		Path:           cleanArtifactPath(input.Path),
		BlobRef:        strings.TrimSpace(input.BlobRef),
		ChecksumSHA256: strings.TrimSpace(input.ChecksumSHA256),
		CreatedAt:      input.CreatedAt,
		MetadataJSON:   strings.TrimSpace(input.MetadataJSON),
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = s.now().UTC()
	} else {
		artifact.CreatedAt = artifact.CreatedAt.UTC()
	}
	if artifact.ContentType == "" {
		artifact.ContentType = defaultContentType(artifact.Kind)
	}
	if err := validateArtifact(artifact); err != nil {
		return nil, err
	}

	existing, err := s.store.ListArtifacts(identityFilter(artifact))
	if err != nil {
		return nil, err
	}
	artifact.Version = nextVersion(existing)
	artifact.ID = artifactID(artifact)
	if err := s.store.SaveArtifact(artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}

func (s *Service) List(filter ListFilter) ([]Artifact, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("store de artifacts obligatorio")
	}
	normalized := ListFilter{
		Scope:   normalizeScope(filter.Scope),
		Kind:    normalizeKind(filter.Kind),
		TaskID:  positiveTaskID(filter.TaskID),
		Agent:   strings.TrimSpace(filter.Agent),
		Project: strings.TrimSpace(filter.Project),
	}
	items, err := s.store.ListArtifacts(normalized)
	if err != nil {
		return nil, err
	}
	sortArtifacts(items)
	return items, nil
}

func validateArtifact(artifact Artifact) error {
	if artifact.Scope == "" {
		return fmt.Errorf("scope obligatorio")
	}
	switch artifact.Scope {
	case ScopeTask:
		if artifact.TaskID == nil {
			return fmt.Errorf("task_id obligatorio para scope task")
		}
	case ScopeAgent:
		if artifact.Agent == "" {
			return fmt.Errorf("agent obligatorio para scope agent")
		}
	case ScopeProject:
		if artifact.Project == "" {
			return fmt.Errorf("project obligatorio para scope project")
		}
	default:
		return fmt.Errorf("scope no soportado: %s", artifact.Scope)
	}
	switch artifact.Kind {
	case KindPatch, KindTests, KindTranscript:
	default:
		return fmt.Errorf("kind no soportado: %s", artifact.Kind)
	}
	if artifact.Path == "" && artifact.BlobRef == "" {
		return fmt.Errorf("path o blob_ref obligatorio")
	}
	return nil
}

func identityFilter(artifact Artifact) ListFilter {
	return ListFilter{
		Scope:   artifact.Scope,
		Kind:    artifact.Kind,
		TaskID:  artifact.TaskID,
		Agent:   artifact.Agent,
		Project: artifact.Project,
	}
}

func nextVersion(existing []Artifact) int {
	maxVersion := 0
	for _, artifact := range existing {
		if artifact.Version > maxVersion {
			maxVersion = artifact.Version
		}
	}
	return maxVersion + 1
}

func artifactID(artifact Artifact) string {
	parts := []string{
		string(artifact.Scope),
		string(artifact.Kind),
		fmt.Sprintf("%d", artifact.Version),
		fmt.Sprintf("%d", artifact.CreatedAt.UnixNano()),
		artifact.Agent,
		artifact.Project,
		artifact.Path,
		artifact.BlobRef,
	}
	if artifact.TaskID != nil {
		parts = append(parts, fmt.Sprintf("%d", *artifact.TaskID))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("artifact:%s:%s:v%d:%s", artifact.Scope, artifact.Kind, artifact.Version, hex.EncodeToString(sum[:])[:12])
}

func defaultContentType(kind Kind) string {
	switch kind {
	case KindPatch:
		return "text/x-diff"
	case KindTests, KindTranscript:
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func cleanArtifactPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func normalizeScope(scope Scope) Scope {
	return Scope(strings.ToLower(strings.TrimSpace(string(scope))))
}

func normalizeKind(kind Kind) Kind {
	return Kind(strings.ToLower(strings.TrimSpace(string(kind))))
}

func positiveTaskID(id *int64) *int64 {
	if id == nil || *id <= 0 {
		return nil
	}
	copyID := *id
	return &copyID
}

func sortArtifacts(items []Artifact) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].Version < items[j].Version
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}
