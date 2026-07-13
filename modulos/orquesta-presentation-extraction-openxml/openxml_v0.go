package orquestapresentationextractionopenxml

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	document "orquesta/modulos/orquesta-document-extraction"
	presentation "orquesta/modulos/orquesta-presentation-extraction"
)

func (a *AdapterV0) projectPPTXV0(data []byte, source presentation.PresentationSourceMaterialV0) (document.DocumentV0, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return document.DocumentV0{}, ErrUnsafeArchiveV0
	}
	if len(zr.File) > a.limits.entries {
		return document.DocumentV0{}, ErrLimitExceededV0
	}
	files := make(map[string]*zip.File, len(zr.File))
	var expanded int64
	for _, f := range zr.File {
		if !safeZipNameV0(f.Name) || f.FileInfo().IsDir() {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		if f.UncompressedSize64 > uint64(a.limits.expanded-expanded) {
			return document.DocumentV0{}, ErrLimitExceededV0
		}
		if _, duplicate := files[f.Name]; duplicate {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		files[f.Name] = f
		content, err := readZipFileV0(f, a.limits.expanded-expanded)
		if err != nil {
			return document.DocumentV0{}, err
		}
		expanded += int64(len(content))
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			if int64(len(content)) > a.limits.xml || validateXMLV0(content, a.limits.depth) != nil {
				return document.DocumentV0{}, ErrUnsafeArchiveV0
			}
			if strings.HasSuffix(f.Name, ".rels") && hasExternalRelationshipV0(content) {
				return document.DocumentV0{}, ErrUnsafeArchiveV0
			}
		}
	}
	presentationXML, ok := files["ppt/presentation.xml"]
	if !ok {
		return document.DocumentV0{}, ErrUnsafeArchiveV0
	}
	presentationBytes, err := readZipFileV0(presentationXML, a.limits.xml)
	if err != nil {
		return document.DocumentV0{}, err
	}
	ids, width, height, err := slideIDsV0(presentationBytes)
	if err != nil || len(ids) == 0 || len(ids) > a.limits.slides {
		return document.DocumentV0{}, ErrUnsafeArchiveV0
	}
	relsFile, ok := files["ppt/_rels/presentation.xml.rels"]
	if !ok {
		return document.DocumentV0{}, ErrUnsafeArchiveV0
	}
	relsBytes, err := readZipFileV0(relsFile, a.limits.xml)
	if err != nil {
		return document.DocumentV0{}, err
	}
	rels, err := relationshipsV0(relsBytes)
	if err != nil {
		return document.DocumentV0{}, ErrUnsafeArchiveV0
	}
	pages := make([]document.DocumentPageV0, 0, len(ids))
	var totalText int64
	for index, id := range ids {
		target, ok := rels[id]
		if !ok {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		slideName, ok := resolvePartV0("ppt/presentation.xml", target)
		if !ok {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		slide, ok := files[slideName]
		if !ok {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		raw, err := readZipFileV0(slide, a.limits.xml)
		if err != nil {
			return document.DocumentV0{}, err
		}
		text, err := slideTextV0(raw, a.limits.depth)
		if err != nil {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		totalText += int64(len(text))
		if totalText > a.limits.text {
			return document.DocumentV0{}, ErrLimitExceededV0
		}
		provenance := document.DocumentProvenanceV0{SourceRefs: []string{source.SourceRef}, AdapterRef: AdapterRefV0, AdapterVersion: AdapterVersionV0}
		page := document.DocumentPageV0{PageRef: fmt.Sprintf("slide-%d", index+1), Index: index, Width: width, Height: height}
		if text != "" {
			page.Blocks = []document.DocumentBlockV0{{BlockRef: fmt.Sprintf("slide-%d-text-0", index+1), Kind: "text", ReadingOrder: 0, Provenance: provenance, Spans: []document.DocumentSpanV0{{SpanRef: fmt.Sprintf("slide-%d-text-0-span-0", index+1), TextRaw: text, Provenance: provenance}}}}
		}
		pages = append(pages, page)
	}
	return document.DocumentV0{IRVersion: document.DocumentExtractionIRSchemaVersionV0, DocumentRef: source.PresentationRef, SourceRef: source.SourceRef, ContentHash: source.ContentHash, MediaKind: "presentation", DetectedKind: "application/vnd.openxmlformats-officedocument.presentationml.presentation", PageCount: len(pages), Pages: pages}, nil
}

func safeZipNameV0(name string) bool {
	return safeRelativePathV0(name) && !strings.HasSuffix(name, "/")
}
func readZipFileV0(f *zip.File, maximum int64) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, ErrUnsafeArchiveV0
	}
	defer r.Close()
	b, err := io.ReadAll(io.LimitReader(r, maximum+1))
	if err != nil || int64(len(b)) > maximum {
		return nil, ErrLimitExceededV0
	}
	return b, nil
}
func validateXMLV0(b []byte, maxDepth int) error {
	d := xml.NewDecoder(bytes.NewReader(b))
	depth := 0
	for {
		token, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch token.(type) {
		case xml.StartElement:
			depth++
			if depth > maxDepth {
				return ErrLimitExceededV0
			}
		case xml.EndElement:
			depth--
		}
	}
}

type relationshipsDocumentV0 struct {
	Relationships []relationshipV0 `xml:"Relationship"`
}

type relationshipV0 struct {
	ID         string `xml:"Id,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr"`
}

func hasExternalRelationshipV0(b []byte) bool {
	var relationships relationshipsDocumentV0
	if xml.Unmarshal(b, &relationships) != nil {
		return false
	}
	for _, relationship := range relationships.Relationships {
		if strings.EqualFold(relationship.TargetMode, "external") {
			return true
		}
	}
	return false
}

func relationshipsV0(b []byte) (map[string]string, error) {
	var relationships relationshipsDocumentV0
	if err := xml.Unmarshal(b, &relationships); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, rel := range relationships.Relationships {
		if rel.ID == "" || rel.Target == "" || strings.EqualFold(rel.TargetMode, "external") {
			return nil, ErrUnsafeArchiveV0
		}
		if _, ok := out[rel.ID]; ok {
			return nil, ErrUnsafeArchiveV0
		}
		out[rel.ID] = rel.Target
	}
	return out, nil
}
func resolvePartV0(base, target string) (string, bool) {
	if strings.Contains(target, "\\") || strings.HasPrefix(target, "/") {
		return "", false
	}
	result := path.Clean(path.Join(path.Dir(base), target))
	return result, safeZipNameV0(result)
}
func slideIDsV0(b []byte) ([]string, float64, float64, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	ids := []string{}
	width, height := 0.0, 0.0
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, 0, err
		}
		start, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "sldId":
			for _, a := range start.Attr {
				if a.Name.Local == "id" {
					ids = append(ids, a.Value)
				}
			}
		case "sldSz":
			for _, a := range start.Attr {
				n, _ := strconv.ParseFloat(a.Value, 64)
				if a.Name.Local == "cx" {
					width = n
				}
				if a.Name.Local == "cy" {
					height = n
				}
			}
		}
	}
	return ids, width, height, nil
}
func slideTextV0(b []byte, depth int) (string, error) {
	if err := validateXMLV0(b, depth); err != nil {
		return "", err
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	parts := []string{}
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		start, ok := t.(xml.StartElement)
		if !ok || start.Name.Local != "t" {
			continue
		}
		var value string
		if err := d.DecodeElement(&value, &start); err != nil {
			return "", err
		}
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n"), nil
}

// Kept deterministic for callers that need reproducible diagnostics.
func sortedZipNamesV0(files map[string]*zip.File) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
