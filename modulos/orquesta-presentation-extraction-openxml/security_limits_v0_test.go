package orquestapresentationextractionopenxml

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	presentation "orquesta/modulos/orquesta-presentation-extraction"
)

type securityPPTXEntryV0 struct {
	name  string
	value string
}

func TestAdapterV0EnforcesSecurityLimitsThroughExtractionV0(t *testing.T) {
	base := securityPPTXEntriesV0("within")
	baseXMLLimit := maxXMLSizeV0(base)
	deepXML := "<root>" + strings.Repeat("<node>", 4) + "value" + strings.Repeat("</node>", 4) + "</root>"

	for _, test := range []struct {
		name    string
		entries []securityPPTXEntryV0
		config  func([]securityPPTXEntryV0, []byte, int64) ConfigV0
		want    error
	}{
		{
			name:    "individual_xml_bytes",
			entries: append(base, securityPPTXEntryV0{name: "ppt/theme/theme1.xml", value: "<root>" + strings.Repeat("x", baseXMLLimit+1) + "</root>"}),
			config: func(_ []securityPPTXEntryV0, _ []byte, _ int64) ConfigV0 {
				return ConfigV0{MaxXMLBytes: int64(baseXMLLimit)}
			},
			want: ErrUnsafeArchiveV0,
		},
		{
			name: "expanded_bytes",
			entries: append(base,
				securityPPTXEntryV0{name: "ppt/media/first.bin", value: strings.Repeat("a", 128)},
				securityPPTXEntryV0{name: "ppt/media/second.bin", value: strings.Repeat("b", 128)}),
			config: func(_ []securityPPTXEntryV0, _ []byte, expanded int64) ConfigV0 {
				return ConfigV0{MaxExpandedBytes: expanded - 1}
			},
			want: ErrLimitExceededV0,
		},
		{
			name:    "xml_depth",
			entries: append(base, securityPPTXEntryV0{name: "ppt/theme/theme1.xml", value: deepXML}),
			config: func(_ []securityPPTXEntryV0, _ []byte, _ int64) ConfigV0 {
				return ConfigV0{MaxXMLBytes: int64(len(deepXML)), MaxXMLDepth: 3}
			},
			want: ErrUnsafeArchiveV0,
		},
		{
			name:    "slides",
			entries: securityPPTXEntriesV0("first", "second"),
			config: func(_ []securityPPTXEntryV0, _ []byte, _ int64) ConfigV0 {
				return ConfigV0{MaxSlides: 1}
			},
			want: ErrUnsafeArchiveV0,
		},
		{
			name:    "total_text_bytes",
			entries: securityPPTXEntriesV0("exceeds"),
			config: func(_ []securityPPTXEntryV0, _ []byte, _ int64) ConfigV0 {
				return ConfigV0{MaxTextBytes: 3}
			},
			want: ErrLimitExceededV0,
		},
		{
			name:    "archive_bytes",
			entries: base,
			config: func(_ []securityPPTXEntryV0, archive []byte, _ int64) ConfigV0 {
				return ConfigV0{MaxArchiveBytes: int64(len(archive) - 1)}
			},
			want: ErrLimitExceededV0,
		},
		{
			name:    "zip_parent_path",
			entries: append(base, securityPPTXEntryV0{name: "../outside.xml", value: "<root/>"}),
			config: func(_ []securityPPTXEntryV0, _ []byte, _ int64) ConfigV0 {
				return ConfigV0{}
			},
			want: ErrUnsafeArchiveV0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			archive, expanded := securityPPTXArchiveV0(t, test.entries)
			config := test.config(test.entries, archive, expanded)
			if err := extractRejectedPPTXV0(t, archive, config); !errors.Is(err, test.want) {
				t.Fatalf("ExtractPresentationV0 error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestAdapterV0AcceptsEverySecurityLimitAtItsBoundaryV0(t *testing.T) {
	entries := securityPPTXEntriesV0("within")
	archive, expanded := securityPPTXArchiveV0(t, entries)
	if err := extractRejectedPPTXV0(t, archive, ConfigV0{
		MaxArchiveBytes:  int64(len(archive)),
		MaxZipEntries:    len(entries),
		MaxXMLBytes:      int64(maxXMLSizeV0(entries)),
		MaxExpandedBytes: expanded,
		MaxSlides:        1,
		MaxTextBytes:     int64(len("within")),
		MaxXMLDepth:      3,
	}); err != nil {
		t.Fatalf("boundary archive rejected: %v", err)
	}
}

func securityPPTXEntriesV0(texts ...string) []securityPPTXEntryV0 {
	ids := make([]string, 0, len(texts))
	rels := make([]string, 0, len(texts))
	entries := []securityPPTXEntryV0{}
	for index, text := range texts {
		id := index + 1
		ids = append(ids, `<p:sldId id="`+string(rune('0'+id))+`" r:id="rId`+string(rune('0'+id))+`"/>`)
		rels = append(rels, `<Relationship Id="rId`+string(rune('0'+id))+`" Type="`+presentationSlideRelationshipTypeV0+`" Target="slides/slide`+string(rune('0'+id))+`.xml"/>`)
		entries = append(entries, securityPPTXEntryV0{name: "ppt/slides/slide" + string(rune('0'+id)) + ".xml", value: `<p:sld xmlns:p="` + presentationMLNamespaceV0 + `" xmlns:a="` + drawingMLNamespaceV0 + `"><a:t>` + text + `</a:t></p:sld>`})
	}
	return append([]securityPPTXEntryV0{
		{name: "ppt/presentation.xml", value: `<p:presentation xmlns:p="` + presentationMLNamespaceV0 + `" xmlns:r="` + officeRelationshipNamespaceV0 + `"><p:sldSz cx="1" cy="1"/><p:sldIdLst>` + strings.Join(ids, "") + `</p:sldIdLst></p:presentation>`},
		{name: "ppt/_rels/presentation.xml.rels", value: `<Relationships xmlns="` + packageRelationshipNamespaceV0 + `">` + strings.Join(rels, "") + `</Relationships>`},
	}, entries...)
}

func securityPPTXArchiveV0(t *testing.T, entries []securityPPTXEntryV0) ([]byte, int64) {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	var expanded int64
	for _, entry := range entries {
		file, err := writer.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(entry.value)); err != nil {
			t.Fatal(err)
		}
		expanded += int64(len(entry.value))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes(), expanded
}

func maxXMLSizeV0(entries []securityPPTXEntryV0) int {
	maximum := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.name, ".xml") || strings.HasSuffix(entry.name, ".rels") {
			maximum = max(maximum, len(entry.value))
		}
	}
	return maximum
}

func extractRejectedPPTXV0(t *testing.T, archive []byte, limits ConfigV0) error {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "security.pptx"), archive, 0600); err != nil {
		t.Fatal(err)
	}
	limits.RootDir = root
	limits.Catalog = []CatalogEntryV0{{PresentationRef: "presentation:security", Path: "security.pptx", SourceRef: "source:security"}}
	adapter, err := NewAdapterV0(limits)
	if err != nil {
		t.Fatal(err)
	}
	_, err = presentation.ExtractPresentationV0(context.Background(), presentation.PresentationExtractionRequestV0{
		PresentationRef: "presentation:security", ConfigurationRef: "config:security", ConfigurationHash: "sha256:security",
	}, presentation.PresentationExtractionPortsV0{Source: adapter, Projector: adapter})
	return err
}
