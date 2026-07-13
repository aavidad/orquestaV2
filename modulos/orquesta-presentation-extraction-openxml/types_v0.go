// Package orquestapresentationextractionopenxml is the opt-in local PPTX
// adapter.  It deliberately keeps filesystem paths private: callers identify a
// presentation only through a catalogued opaque ref.
package orquestapresentationextractionopenxml

import "errors"

const (
	AdapterRefV0     = "presentation-extraction-openxml"
	AdapterVersionV0 = "v0"

	DefaultMaxArchiveBytesV0  int64 = 32 << 20
	DefaultMaxZipEntriesV0          = 256
	DefaultMaxXMLBytesV0      int64 = 4 << 20
	DefaultMaxExpandedBytesV0 int64 = 64 << 20
	DefaultMaxSlidesV0              = 200
	DefaultMaxTextBytesV0     int64 = 4 << 20
	DefaultMaxXMLDepthV0            = 64
)

var (
	ErrRootRequiredV0         = errors.New("presentation_openxml_root_required")
	ErrCatalogInvalidV0       = errors.New("presentation_openxml_catalog_invalid")
	ErrPresentationNotFoundV0 = errors.New("presentation_openxml_presentation_not_found")
	ErrUnsafePathV0           = errors.New("presentation_openxml_unsafe_path")
	ErrUnsafeSymlinkV0        = errors.New("presentation_openxml_unsafe_symlink")
	ErrUnsafeArchiveV0        = errors.New("presentation_openxml_unsafe_archive")
	ErrLimitExceededV0        = errors.New("presentation_openxml_limit_exceeded")
	ErrSnapshotChangedV0      = errors.New("presentation_openxml_snapshot_changed")
	ErrUnsupportedFormatV0    = errors.New("presentation_openxml_unsupported_format")
)

// CatalogEntryV0 maps a stable opaque ref to a relative file beneath RootDir.
// Path is configuration data, never accepted from a presentation request.
type CatalogEntryV0 struct {
	PresentationRef string
	Path            string
	SourceRef       string
}

type ConfigV0 struct {
	RootDir string
	Catalog []CatalogEntryV0

	MaxArchiveBytes  int64
	MaxZipEntries    int
	MaxXMLBytes      int64
	MaxExpandedBytes int64
	MaxSlides        int
	MaxTextBytes     int64
	MaxXMLDepth      int
}

type AdapterV0 struct {
	rootDir string
	catalog map[string]CatalogEntryV0
	limits  limitsV0
}

type limitsV0 struct {
	archive, xml, expanded, text int64
	entries, slides, depth       int
}
