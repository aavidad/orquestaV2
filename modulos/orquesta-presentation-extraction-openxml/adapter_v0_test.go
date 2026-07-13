package orquestapresentationextractionopenxml

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	presentation "orquesta/modulos/orquesta-presentation-extraction"
)

func fixturePPTXV0(t *testing.T, external bool) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	files := []struct{ name, value string }{{"ppt/presentation.xml", `<p:presentation xmlns:p="` + presentationMLNamespaceV0 + `" xmlns:r="` + officeRelationshipNamespaceV0 + `"><p:sldSz cx="9144000" cy="5143500"/><p:sldIdLst><p:sldId id="257" r:id="rId2"/><p:sldId id="256" r:id="rId1"/></p:sldIdLst></p:presentation>`}, {"ppt/_rels/presentation.xml.rels", `<Relationships xmlns="` + packageRelationshipNamespaceV0 + `">` + func() string {
		if external {
			return `<Relationship Id="rId1" Type="` + presentationSlideRelationshipTypeV0 + `" Target="https://bad" TargetMode="External"/>`
		}
		return `<Relationship Id="rId1" Type="` + presentationSlideRelationshipTypeV0 + `" Target="slides/slide1.xml"/>`
	}() + `<Relationship Id="rId2" Type="` + presentationSlideRelationshipTypeV0 + `" Target="slides/slide2.xml"/></Relationships>`}, {"ppt/slides/slide1.xml", `<p:sld xmlns:p="` + presentationMLNamespaceV0 + `" xmlns:a="` + drawingMLNamespaceV0 + `"><a:t>uno</a:t></p:sld>`}, {"ppt/slides/slide2.xml", `<p:sld xmlns:p="` + presentationMLNamespaceV0 + `" xmlns:a="` + drawingMLNamespaceV0 + `"><a:t>dos</a:t></p:sld>`}}
	for _, file := range files {
		w, err := z.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(file.value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func adapterFixtureV0(t *testing.T) (*AdapterV0, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "fixture.pptx"), fixturePPTXV0(t, false), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := NewAdapterV0(ConfigV0{RootDir: root, Catalog: []CatalogEntryV0{{PresentationRef: "presentation:fixture", Path: "fixture.pptx", SourceRef: "source:fixture"}}})
	if err != nil {
		t.Fatal(err)
	}
	return a, root
}
func TestAdapterV0ExtractsOrderedTwoSlideFixtureV0(t *testing.T) {
	a, _ := adapterFixtureV0(t)
	result, err := presentation.ExtractPresentationV0(context.Background(), presentation.PresentationExtractionRequestV0{PresentationRef: "presentation:fixture", ConfigurationRef: "config:test", ConfigurationHash: "sha256:test"}, presentation.PresentationExtractionPortsV0{Source: a, Projector: a})
	if err != nil {
		t.Fatal(err)
	}
	if result.Document.PageCount != 2 || result.Document.Pages[0].Blocks[0].Spans[0].TextRaw != "dos" || result.Receipt.SnapshotRef == "" {
		t.Fatalf("unexpected projection: %#v", result.Document)
	}
}
func TestAdapterV0RejectsNestedSymlinkAndExternalRelationshipV0(t *testing.T) {
	a, root := adapterFixtureV0(t)
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../fixture.pptx", filepath.Join(root, "nested", "file.pptx")); err == nil {
		a.catalog["presentation:fixture"] = CatalogEntryV0{PresentationRef: "presentation:fixture", Path: "nested/file.pptx", SourceRef: "source:fixture"}
		if _, err := a.ResolvePresentationV0(context.Background(), "presentation:fixture"); err == nil {
			t.Fatal("symlink accepted")
		}
	}
	if err := os.WriteFile(filepath.Join(root, "external.pptx"), fixturePPTXV0(t, true), 0600); err != nil {
		t.Fatal(err)
	}
	a.catalog["presentation:external"] = CatalogEntryV0{PresentationRef: "presentation:external", Path: "external.pptx", SourceRef: "source:external"}
	source, err := a.ResolvePresentationV0(context.Background(), "presentation:external")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ProjectPresentationDocumentV0(context.Background(), source); err == nil {
		t.Fatal("external relationship accepted")
	}
}
func TestAdapterV0RejectsSnapshotChangeV0(t *testing.T) {
	a, root := adapterFixtureV0(t)
	source, err := a.ResolvePresentationV0(context.Background(), "presentation:fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fixture.pptx"), fixturePPTXV0(t, true), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ProjectPresentationDocumentV0(context.Background(), source); err == nil {
		t.Fatal("changed source accepted")
	}
}

func TestAdapterV0RejectsFalseNamespacesAndUnsafeSlideRelationshipsV0(t *testing.T) {
	base := securityPPTXEntriesV0("valid")
	for _, test := range []struct {
		name   string
		mutate func([]securityPPTXEntryV0)
	}{
		{"presentation_namespace", func(entries []securityPPTXEntryV0) {
			entries[0].value = strings.Replace(entries[0].value, presentationMLNamespaceV0, "urn:forged:presentation", 1)
		}},
		{"relationship_namespace", func(entries []securityPPTXEntryV0) {
			entries[1].value = strings.Replace(entries[1].value, packageRelationshipNamespaceV0, "urn:forged:relationships", 1)
		}},
		{"slide_text_namespace", func(entries []securityPPTXEntryV0) {
			entries[2].value = strings.Replace(entries[2].value, drawingMLNamespaceV0, "urn:forged:drawing", 1)
		}},
		{"relationship_attribute_namespace", func(entries []securityPPTXEntryV0) {
			entries[0].value = strings.Replace(entries[0].value, officeRelationshipNamespaceV0, "urn:forged:relationship", 1)
		}},
		{"non_slide_relationship", func(entries []securityPPTXEntryV0) {
			entries[1].value = strings.Replace(entries[1].value, presentationSlideRelationshipTypeV0, officeRelationshipNamespaceV0+"/theme", 1)
		}},
		{"slide_target_escape", func(entries []securityPPTXEntryV0) {
			entries[1].value = strings.Replace(entries[1].value, "slides/slide1.xml", "../../outside.xml", 1)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			entries := append([]securityPPTXEntryV0(nil), base...)
			test.mutate(entries)
			archive, _ := securityPPTXArchiveV0(t, entries)
			if err := extractRejectedPPTXV0(t, archive, ConfigV0{}); err == nil {
				t.Fatal("unsafe OOXML accepted")
			}
		})
	}
}

func TestAdapterV0RejectsZipTraversalDuplicateAndEntryLimitV0(t *testing.T) {
	for _, test := range []struct {
		name string
		make func(*testing.T) []byte
	}{
		{"traversal", func(t *testing.T) []byte { return archiveWithEntriesV0(t, []string{"../slide.xml"}) }},
		{"duplicate", func(t *testing.T) []byte {
			return archiveWithEntriesV0(t, []string{"ppt/presentation.xml", "ppt/presentation.xml"})
		}},
		{"entries", func(t *testing.T) []byte { return archiveWithEntriesV0(t, []string{"one", "two"}) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "bad.pptx"), test.make(t), 0600); err != nil {
				t.Fatal(err)
			}
			config := ConfigV0{RootDir: root, Catalog: []CatalogEntryV0{{PresentationRef: "presentation:bad", Path: "bad.pptx", SourceRef: "source:bad"}}}
			if test.name == "entries" {
				config.MaxZipEntries = 1
			}
			a, err := NewAdapterV0(config)
			if err != nil {
				t.Fatal(err)
			}
			source, err := a.ResolvePresentationV0(context.Background(), "presentation:bad")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := a.ProjectPresentationDocumentV0(context.Background(), source); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}

func archiveWithEntriesV0(t *testing.T, names []string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range names {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
