package orquestapresentationextractionopenxml

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	document "orquesta/modulos/orquesta-document-extraction"
	presentation "orquesta/modulos/orquesta-presentation-extraction"
)

const (
	presentationMLNamespaceV0           = "http://schemas.openxmlformats.org/presentationml/2006/main"
	drawingMLNamespaceV0                = "http://schemas.openxmlformats.org/drawingml/2006/main"
	officeRelationshipNamespaceV0       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	packageRelationshipNamespaceV0      = "http://schemas.openxmlformats.org/package/2006/relationships"
	presentationSlideRelationshipTypeV0 = officeRelationshipNamespaceV0 + "/slide"
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
		if f.FileInfo().IsDir() {
			if !safeRelativePathV0(strings.TrimSuffix(f.Name, "/")) {
				return document.DocumentV0{}, ErrUnsafeArchiveV0
			}
			continue
		}
		if !safeZipNameV0(f.Name) {
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
			if strings.HasSuffix(f.Name, ".rels") && validateRelationshipsV0(content) != nil {
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
		rel, ok := rels[id]
		if !ok || rel.Type != presentationSlideRelationshipTypeV0 {
			return document.DocumentV0{}, ErrUnsafeArchiveV0
		}
		slideName, ok := resolvePartV0("ppt/presentation.xml", rel.Target)
		if !ok || !isPresentationSlidePartV0(slideName) {
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
		case xml.Directive:
			return ErrUnsafeArchiveV0
		}
	}
}

type relationshipV0 struct {
	ID         string
	Target     string
	TargetMode string
	Type       string
}

func validateRelationshipsV0(b []byte) error {
	_, err := relationshipsV0(b)
	return err
}

func relationshipsV0(b []byte) (map[string]relationshipV0, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	out := map[string]relationshipV0{}
	seenRoot := false
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if !seenRoot {
			if start.Name.Space != packageRelationshipNamespaceV0 || start.Name.Local != "Relationships" {
				return nil, ErrUnsafeArchiveV0
			}
			seenRoot = true
			continue
		}
		if start.Name.Space != packageRelationshipNamespaceV0 || start.Name.Local != "Relationship" {
			return nil, ErrUnsafeArchiveV0
		}
		var rel relationshipV0
		for _, attr := range start.Attr {
			if attr.Name.Space != "" {
				return nil, ErrUnsafeArchiveV0
			}
			switch attr.Name.Local {
			case "Id":
				rel.ID = attr.Value
			case "Target":
				rel.Target = attr.Value
			case "TargetMode":
				rel.TargetMode = attr.Value
			case "Type":
				rel.Type = attr.Value
			}
		}
		if rel.ID == "" || rel.Target == "" || rel.Type == "" || strings.EqualFold(rel.TargetMode, "external") {
			return nil, ErrUnsafeArchiveV0
		}
		if _, ok := out[rel.ID]; ok {
			return nil, ErrUnsafeArchiveV0
		}
		out[rel.ID] = rel
	}
	if !seenRoot {
		return nil, ErrUnsafeArchiveV0
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

func isPresentationSlidePartV0(name string) bool {
	return strings.HasPrefix(name, "ppt/slides/") && strings.HasSuffix(name, ".xml")
}

func slideIDsV0(b []byte) ([]string, float64, float64, error) {
	d := xml.NewDecoder(bytes.NewReader(b))
	ids := []string{}
	width, height := 0.0, 0.0
	seenRoot := false
	seenSize := false
	numericIDs := map[string]struct{}{}
	relationshipIDs := map[string]struct{}{}
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
		if !seenRoot {
			if start.Name.Space != presentationMLNamespaceV0 || start.Name.Local != "presentation" {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			seenRoot = true
			continue
		}
		switch start.Name.Local {
		case "sldId":
			if start.Name.Space != presentationMLNamespaceV0 {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			var numericID, relationshipID string
			for _, a := range start.Attr {
				switch {
				case a.Name.Space == "" && a.Name.Local == "id":
					numericID = a.Value
				case a.Name.Space == officeRelationshipNamespaceV0 && a.Name.Local == "id":
					relationshipID = a.Value
				case a.Name.Local == "id":
					return nil, 0, 0, ErrUnsafeArchiveV0
				}
			}
			if numericID == "" || relationshipID == "" {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			if _, err := strconv.ParseUint(numericID, 10, 32); err != nil {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			if _, exists := numericIDs[numericID]; exists {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			if _, exists := relationshipIDs[relationshipID]; exists {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			numericIDs[numericID] = struct{}{}
			relationshipIDs[relationshipID] = struct{}{}
			ids = append(ids, relationshipID)
		case "sldSz":
			if start.Name.Space != presentationMLNamespaceV0 {
				return nil, 0, 0, ErrUnsafeArchiveV0
			}
			seenSize = true
			for _, a := range start.Attr {
				// PresentationML permits non-numeric attributes such as
				// type="screen4x3". Only the dimensions are numeric inputs.
				switch {
				case a.Name.Space == "" && a.Name.Local == "cx":
					n, err := strconv.ParseUint(a.Value, 10, 32)
					if err != nil {
						return nil, 0, 0, ErrUnsafeArchiveV0
					}
					width = float64(n)
				case a.Name.Space == "" && a.Name.Local == "cy":
					n, err := strconv.ParseUint(a.Value, 10, 32)
					if err != nil {
						return nil, 0, 0, ErrUnsafeArchiveV0
					}
					height = float64(n)
				}
			}
		}
	}
	if !seenRoot || !seenSize || width <= 0 || height <= 0 {
		return nil, 0, 0, ErrUnsafeArchiveV0
	}
	return ids, width, height, nil
}
func slideTextV0(b []byte, depth int) (string, error) {
	if err := validateXMLV0(b, depth); err != nil {
		return "", err
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	parts := []string{}
	seenRoot := false
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		start, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		if !seenRoot {
			if start.Name.Space != presentationMLNamespaceV0 || start.Name.Local != "sld" {
				return "", ErrUnsafeArchiveV0
			}
			seenRoot = true
			continue
		}
		if start.Name.Local != "t" {
			continue
		}
		if start.Name.Space != drawingMLNamespaceV0 {
			return "", ErrUnsafeArchiveV0
		}
		var value string
		if err := d.DecodeElement(&value, &start); err != nil {
			return "", err
		}
		if value != "" {
			parts = append(parts, value)
		}
	}
	if !seenRoot {
		return "", ErrUnsafeArchiveV0
	}
	return strings.Join(parts, "\n"), nil
}
