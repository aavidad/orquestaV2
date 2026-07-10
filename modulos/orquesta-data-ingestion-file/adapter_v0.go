// Package orquestadataingestionfile provides an opt-in local-file adapter for
// the neutral data ingestion ports. Paths never leave this package: callers
// register opaque source refs in a catalogue at construction time.
package orquestadataingestionfile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ingestion "orquesta/modulos/orquesta-data-ingestion"
)

const (
	defaultMaxBytesV0 int64 = 64 << 20
	defaultMaxRowsV0  int64 = 1_000_000
)

// SourceRegistrationV0 is composition-owned configuration. RelativePath is
// always relative to the explicitly configured AllowedRoot.
type SourceRegistrationV0 struct {
	DatasetRef   string
	RelativePath string
	SourceKind   ingestion.DataSourceKindV0
}

// ConfigurationV0 supplies all local filesystem authority for the adapter.
// Source refs and dataset refs are opaque identifiers owned by the caller.
type ConfigurationV0 struct {
	AllowedRoot string
	Sources     map[string]SourceRegistrationV0
	MaxBytes    int64
	MaxRows     int64
	AdapterRef  string
	Version     string
}

// AdapterV0 implements ingestion.DataSourcePortV0 and
// ingestion.DataProfilerPortV0.
type AdapterV0 struct {
	allowedRoot string
	sources     map[string]SourceRegistrationV0
	maxBytes    int64
	maxRows     int64
	identity    ingestion.DataAdapterIdentityV0
}

var _ ingestion.DataSourcePortV0 = (*AdapterV0)(nil)
var _ ingestion.DataProfilerPortV0 = (*AdapterV0)(nil)

// NewAdapterV0 validates the catalogue before it can be used. It accepts only
// CSV and JSON sources and rejects symlinks in registered paths.
func NewAdapterV0(configuration ConfigurationV0) (*AdapterV0, error) {
	root, err := absoluteDirectoryV0(configuration.AllowedRoot)
	if err != nil {
		return nil, err
	}
	if len(configuration.Sources) == 0 {
		return nil, fmt.Errorf("data_file_source_catalogue_required")
	}
	adapter := &AdapterV0{
		allowedRoot: root,
		sources:     make(map[string]SourceRegistrationV0, len(configuration.Sources)),
		maxBytes:    positiveOrDefaultV0(configuration.MaxBytes, defaultMaxBytesV0),
		maxRows:     positiveOrDefaultV0(configuration.MaxRows, defaultMaxRowsV0),
		identity: ingestion.DataAdapterIdentityV0{
			AdapterRef: nonBlankOrDefaultV0(configuration.AdapterRef, "adapter:data-ingestion-file"),
			Version:    nonBlankOrDefaultV0(configuration.Version, "v0"),
		},
	}
	for sourceRef, registration := range configuration.Sources {
		if strings.TrimSpace(sourceRef) == "" || strings.TrimSpace(registration.DatasetRef) == "" || !supportedKindV0(registration.SourceKind) {
			return nil, fmt.Errorf("data_file_source_catalogue_invalid")
		}
		if _, err := adapter.pathForRegistrationV0(registration); err != nil {
			return nil, fmt.Errorf("data_file_source_catalogue_invalid: %w", err)
		}
		adapter.sources[sourceRef] = registration
	}
	return adapter, nil
}

func (adapter *AdapterV0) AdapterIdentityV0() ingestion.DataAdapterIdentityV0 {
	return adapter.identity
}

// ResolveDataSourceV0 opens and hashes the registered regular file without
// exposing its location in the returned material.
func (adapter *AdapterV0) ResolveDataSourceV0(ctx context.Context, sourceRef string) (ingestion.DataSourceMaterialV0, error) {
	registration, path, err := adapter.sourcePathV0(sourceRef)
	if err != nil {
		return ingestion.DataSourceMaterialV0{}, err
	}
	hash, err := adapter.hashFileV0(ctx, path)
	if err != nil {
		return ingestion.DataSourceMaterialV0{}, err
	}
	return materialV0(sourceRef, registration, hash), nil
}

// ProfileDataSetV0 resolves the opaque ref again, verifies its current hash,
// then reads at most the configured rows and bytes.
func (adapter *AdapterV0) ProfileDataSetV0(ctx context.Context, source ingestion.DataSourceMaterialV0) (ingestion.DataDatasetProfileV0, error) {
	registration, path, err := adapter.sourcePathV0(source.SourceRef)
	if err != nil {
		return ingestion.DataDatasetProfileV0{}, err
	}
	hash, err := adapter.hashFileV0(ctx, path)
	if err != nil {
		return ingestion.DataDatasetProfileV0{}, err
	}
	expected := materialV0(source.SourceRef, registration, hash)
	if source != expected {
		return ingestion.DataDatasetProfileV0{}, fmt.Errorf("data_file_source_snapshot_mismatch")
	}
	columns, rows, err := adapter.profileFileV0(ctx, path, registration.SourceKind)
	if err != nil {
		return ingestion.DataDatasetProfileV0{}, err
	}
	return ingestion.DataDatasetProfileV0{
		DatasetRef: source.DatasetRef,
		ProfileRef: "profile:file:" + sha256HexV0(source.ContentHash+fmt.Sprintf(":%d:%d", adapter.maxBytes, adapter.maxRows)),
		Columns:    columns,
		RowCount:   rows,
	}, nil
}

func (adapter *AdapterV0) sourcePathV0(sourceRef string) (SourceRegistrationV0, string, error) {
	registration, found := adapter.sources[sourceRef]
	if !found {
		return SourceRegistrationV0{}, "", fmt.Errorf("data_file_source_not_found")
	}
	path, err := adapter.pathForRegistrationV0(registration)
	if err != nil {
		return SourceRegistrationV0{}, "", err
	}
	return registration, path, nil
}

func (adapter *AdapterV0) pathForRegistrationV0(registration SourceRegistrationV0) (string, error) {
	for _, part := range strings.Split(registration.RelativePath, string(filepath.Separator)) {
		if part == ".." {
			return "", fmt.Errorf("data_file_path_outside_allowed_root")
		}
	}
	relative := filepath.Clean(registration.RelativePath)
	if relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("data_file_path_outside_allowed_root")
	}
	path := filepath.Join(adapter.allowedRoot, relative)
	if err := rejectSymlinkPathV0(adapter.allowedRoot, relative); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("data_file_source_unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("data_file_source_not_regular")
	}
	return path, nil
}

func (adapter *AdapterV0) hashFileV0(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("data_file_source_unavailable: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 32*1024)
	var read int64
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := file.Read(buffer)
		if n > 0 {
			read += int64(n)
			if read > adapter.maxBytes {
				return "", fmt.Errorf("data_file_size_limit_exceeded")
			}
			_, _ = hash.Write(buffer[:n])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func materialV0(sourceRef string, registration SourceRegistrationV0, hash string) ingestion.DataSourceMaterialV0 {
	return ingestion.DataSourceMaterialV0{DatasetRef: registration.DatasetRef, SourceRef: sourceRef, SourceKind: registration.SourceKind, ContentHash: hash, SnapshotRef: "snapshot:file:" + hash, ProvenanceRef: "provenance:file:" + hash}
}

func absoluteDirectoryV0(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("data_file_allowed_root_required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("data_file_allowed_root_invalid")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("data_file_allowed_root_symlink_not_allowed")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("data_file_allowed_root_invalid")
	}
	return filepath.Clean(absolute), nil
}

func rejectSymlinkPathV0(root, relative string) error {
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("data_file_source_unavailable: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("data_file_symlink_not_allowed")
		}
	}
	return nil
}

func supportedKindV0(kind ingestion.DataSourceKindV0) bool {
	return kind == ingestion.DataSourceKindCSVV0 || kind == ingestion.DataSourceKindJSONV0
}

func positiveOrDefaultV0(value, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}

func nonBlankOrDefaultV0(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func sha256HexV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
