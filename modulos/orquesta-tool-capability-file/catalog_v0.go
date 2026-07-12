package orquestatoolcapabilityfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

const (
	toolCapabilityCatalogSchemaV0   = "tool_capability_file_catalog.v0"
	toolCapabilityCatalogFileNameV0 = "capability-catalog-v0.json"
	maxToolCapabilityCatalogFileV0  = 1 << 20
)

type ToolCapabilityFileCatalogV0 struct {
	root        string
	catalogPath string
}

func NewToolCapabilityFileCatalogV0(root string) (*ToolCapabilityFileCatalogV0, error) {
	if !filepath.IsAbs(root) {
		return nil, errors.New("tool capability catalog root must be absolute")
	}
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create tool capability catalog root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect tool capability catalog root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("tool capability catalog root must not traverse symlinks")
	}
	if !info.IsDir() {
		return nil, errors.New("tool capability catalog root must be a real directory")
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("secure tool capability catalog root: %w", err)
	}
	info, err = os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("reinspect tool capability catalog root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("tool capability catalog root must be a real directory")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("tool capability catalog root must not be group/world accessible")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || filepath.Clean(resolved) != root {
		return nil, errors.New("tool capability catalog root must not traverse symlinks")
	}
	return &ToolCapabilityFileCatalogV0{
		root:        root,
		catalogPath: filepath.Join(root, toolCapabilityCatalogFileNameV0),
	}, nil
}

func (catalog *ToolCapabilityFileCatalogV0) ListCapabilityManifestsV0(
	ctx context.Context,
	filter toolcapability.CapabilityManifestFilterV0,
) ([]toolcapability.CapabilityManifestV0, error) {
	if catalog == nil {
		return nil, errors.New("tool capability catalog is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	manifests, err := catalog.loadManifestsV0()
	if err != nil {
		return nil, err
	}
	filtered := make([]toolcapability.CapabilityManifestV0, 0, len(manifests))
	for _, manifest := range manifests {
		if capabilityManifestMatchesFilterV0(manifest, filter) {
			filtered = append(filtered, manifest)
		}
	}
	sortCapabilityManifestsV0(filtered)
	return filtered, nil
}

func (catalog *ToolCapabilityFileCatalogV0) loadManifestsV0() ([]toolcapability.CapabilityManifestV0, error) {
	fd, err := syscall.Open(catalog.catalogPath, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open tool capability catalog: %w", err)
	}
	file := os.NewFile(uintptr(fd), catalog.catalogPath)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > maxToolCapabilityCatalogFileV0 {
		return nil, errors.New("tool capability catalog must be a private bounded regular file")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxToolCapabilityCatalogFileV0))
	decoder.DisallowUnknownFields()
	var envelope toolCapabilityFileCatalogEnvelopeV0
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode tool capability catalog: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("tool capability catalog contains trailing data")
	}
	if envelope.SchemaVersion != toolCapabilityCatalogSchemaV0 {
		return nil, errors.New("invalid tool capability catalog schema")
	}
	for _, manifest := range envelope.Manifests {
		if issues := toolcapability.ValidateCapabilityManifestV0(manifest); len(issues) > 0 {
			return nil, fmt.Errorf("invalid tool capability manifest: %+v", issues)
		}
	}
	manifests := append([]toolcapability.CapabilityManifestV0(nil), envelope.Manifests...)
	sortCapabilityManifestsV0(manifests)
	return manifests, nil
}

type toolCapabilityFileCatalogEnvelopeV0 struct {
	SchemaVersion string                                `json:"schema_version"`
	Manifests     []toolcapability.CapabilityManifestV0 `json:"manifests"`
}

func capabilityManifestMatchesFilterV0(
	manifest toolcapability.CapabilityManifestV0,
	filter toolcapability.CapabilityManifestFilterV0,
) bool {
	if filter.CapabilityRef != "" && manifest.CapabilityRef != filter.CapabilityRef {
		return false
	}
	if filter.ToolRef != "" && manifest.ToolRef != filter.ToolRef {
		return false
	}
	if filter.Locale != "" && !stringSliceContainsToolCapabilityFileV0(manifest.Locales, filter.Locale) {
		return false
	}
	return true
}

func sortCapabilityManifestsV0(manifests []toolcapability.CapabilityManifestV0) {
	sort.SliceStable(manifests, func(left, right int) bool {
		a := manifests[left]
		b := manifests[right]
		if a.CapabilityRef != b.CapabilityRef {
			return a.CapabilityRef < b.CapabilityRef
		}
		if a.ToolRef != b.ToolRef {
			return a.ToolRef < b.ToolRef
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.ManifestRef < b.ManifestRef
	})
}

func stringSliceContainsToolCapabilityFileV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
