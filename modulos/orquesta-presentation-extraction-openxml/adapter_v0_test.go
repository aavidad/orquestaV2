package orquestapresentationextractionopenxml

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	presentation "orquesta/modulos/orquesta-presentation-extraction"
)

func fixturePPTXV0(t *testing.T, external bool) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	files := []struct{ name, value string }{{"ppt/presentation.xml", `<p:presentation xmlns:p="p" xmlns:r="r"><p:sldSz cx="9144000" cy="5143500"/><p:sldIdLst><p:sldId r:id="rId2"/><p:sldId r:id="rId1"/></p:sldIdLst></p:presentation>`}, {"ppt/_rels/presentation.xml.rels", `<Relationships>` + func() string {
		if external {
			return `<Relationship Id="rId1" Target="https://bad" TargetMode="External"/>`
		}
		return `<Relationship Id="rId1" Target="slides/slide1.xml"/>`
	}() + `<Relationship Id="rId2" Target="slides/slide2.xml"/></Relationships>`}, {"ppt/slides/slide1.xml", `<p:sld xmlns:p="p" xmlns:a="a"><a:t>uno</a:t></p:sld>`}, {"ppt/slides/slide2.xml", `<p:sld xmlns:p="p" xmlns:a="a"><a:t>dos</a:t></p:sld>`}}
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

func TestTwoSlideFixtureIsStableV0(t *testing.T) {
	fixture := fixturePPTXV0(t, false)
	sum := sha256.Sum256(fixture)
	if got := hex.EncodeToString(sum[:]); got != "0194a5abf17c5de801b413effbda0488c32891da0d427926ce3e08fcfe41006a" {
		t.Fatalf("two-slide fixture hash = %s", got)
	}
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
