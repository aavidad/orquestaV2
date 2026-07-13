package orquestapresentationextractionopenxml

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	document "orquesta/modulos/orquesta-document-extraction"
	presentation "orquesta/modulos/orquesta-presentation-extraction"
)

var _ presentation.PresentationSourcePortV0 = (*AdapterV0)(nil)
var _ presentation.PresentationDocumentProjectorPortV0 = (*AdapterV0)(nil)

func NewAdapterV0(config ConfigV0) (*AdapterV0, error) {
	if strings.TrimSpace(config.RootDir) == "" {
		return nil, ErrRootRequiredV0
	}
	limits := defaultLimitsV0(config)
	if limits.archive <= 0 || limits.entries <= 0 || limits.xml <= 0 || limits.expanded <= 0 || limits.slides <= 0 || limits.text <= 0 || limits.depth <= 0 {
		return nil, ErrCatalogInvalidV0
	}
	a := &AdapterV0{rootDir: config.RootDir, catalog: make(map[string]CatalogEntryV0), limits: limits}
	for _, entry := range config.Catalog {
		entry.PresentationRef, entry.Path, entry.SourceRef = strings.TrimSpace(entry.PresentationRef), strings.TrimSpace(entry.Path), strings.TrimSpace(entry.SourceRef)
		if entry.PresentationRef == "" || entry.SourceRef == "" || !safeRelativePathV0(entry.Path) || !strings.HasSuffix(strings.ToLower(entry.Path), ".pptx") {
			return nil, ErrCatalogInvalidV0
		}
		if _, exists := a.catalog[entry.PresentationRef]; exists {
			return nil, ErrCatalogInvalidV0
		}
		a.catalog[entry.PresentationRef] = entry
	}
	return a, nil
}

func defaultLimitsV0(c ConfigV0) limitsV0 {
	l := limitsV0{archive: c.MaxArchiveBytes, entries: c.MaxZipEntries, xml: c.MaxXMLBytes, expanded: c.MaxExpandedBytes, slides: c.MaxSlides, text: c.MaxTextBytes, depth: c.MaxXMLDepth}
	if l.archive == 0 {
		l.archive = DefaultMaxArchiveBytesV0
	}
	if l.entries == 0 {
		l.entries = DefaultMaxZipEntriesV0
	}
	if l.xml == 0 {
		l.xml = DefaultMaxXMLBytesV0
	}
	if l.expanded == 0 {
		l.expanded = DefaultMaxExpandedBytesV0
	}
	if l.slides == 0 {
		l.slides = DefaultMaxSlidesV0
	}
	if l.text == 0 {
		l.text = DefaultMaxTextBytesV0
	}
	if l.depth == 0 {
		l.depth = DefaultMaxXMLDepthV0
	}
	return l
}

func (a *AdapterV0) AdapterIdentityV0() presentation.PresentationAdapterIdentityV0 {
	return presentation.PresentationAdapterIdentityV0{AdapterRef: AdapterRefV0, Version: AdapterVersionV0}
}

func (a *AdapterV0) ResolvePresentationV0(_ context.Context, presentationRef string) (presentation.PresentationSourceMaterialV0, error) {
	entry, ok := a.catalog[strings.TrimSpace(presentationRef)]
	if !ok {
		return presentation.PresentationSourceMaterialV0{}, ErrPresentationNotFoundV0
	}
	data, err := a.readCataloguedV0(entry)
	if err != nil {
		return presentation.PresentationSourceMaterialV0{}, err
	}
	hash := hashV0(data)
	return presentation.PresentationSourceMaterialV0{PresentationRef: entry.PresentationRef, SourceRef: entry.SourceRef, Format: presentation.PresentationFormatPPTXV0, ContentHash: hash, SnapshotRef: "snapshot:openxml:" + hash}, nil
}

// ProjectPresentationDocumentV0 reopens the catalogued source. Source bytes
// are intentionally never cached; hash and snapshot bind projection to Resolve.
func (a *AdapterV0) ProjectPresentationDocumentV0(_ context.Context, source presentation.PresentationSourceMaterialV0) (document.DocumentV0, error) {
	entry, ok := a.catalog[source.PresentationRef]
	if !ok || source.Format != presentation.PresentationFormatPPTXV0 || entry.SourceRef != source.SourceRef {
		return document.DocumentV0{}, ErrUnsupportedFormatV0
	}
	data, err := a.readCataloguedV0(entry)
	if err != nil {
		return document.DocumentV0{}, err
	}
	hash := hashV0(data)
	if hash != source.ContentHash || source.SnapshotRef != "snapshot:openxml:"+hash {
		return document.DocumentV0{}, ErrSnapshotChangedV0
	}
	return a.projectPPTXV0(data, source)
}

func (a *AdapterV0) readCataloguedV0(entry CatalogEntryV0) ([]byte, error) {
	root, err := os.OpenRoot(a.rootDir)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRootRequiredV0, err)
	}
	defer root.Close()
	if err := lstatComponentsV0(root, entry.Path); err != nil {
		return nil, err
	}
	file, err := root.Open(entry.Path)
	if err != nil {
		return nil, ErrPresentationNotFoundV0
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, ErrPresentationNotFoundV0
	}
	if info.Size() > a.limits.archive {
		return nil, ErrLimitExceededV0
	}
	data, err := io.ReadAll(io.LimitReader(file, a.limits.archive+1))
	if err != nil || int64(len(data)) > a.limits.archive {
		return nil, ErrLimitExceededV0
	}
	return data, nil
}

func lstatComponentsV0(root *os.Root, name string) error {
	// Root.Lstat only accepts paths below root. Check every cumulative component,
	// rather than a final path, so a nested symlink is never followed.
	parts := strings.Split(name, "/")
	for i := range parts {
		info, err := root.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil {
			return ErrPresentationNotFoundV0
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafeSymlinkV0
		}
	}
	return nil
}

func safeRelativePathV0(name string) bool {
	return name != "" && !strings.Contains(name, "\\") && !strings.HasPrefix(name, "/") && path.Clean(name) == name && name != "." && !strings.HasPrefix(name, "../") && !strings.Contains(name, "//")
}
func hashV0(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
