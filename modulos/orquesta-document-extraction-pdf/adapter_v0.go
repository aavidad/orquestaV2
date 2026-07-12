package orquestadocumentextractionpdf

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	extraction "orquesta/modulos/orquesta-document-extraction"
)

const (
	AdapterRefV0     = "document-extraction-pdf"
	AdapterVersionV0 = "v0"
	MediaKindV0      = "application/pdf"

	// pdftotext (poppler) es la fuente de texto y de anclas espaciales. El
	// nucleo exige pagina y bbox por evidencia, asi que solo sirve el modo
	// -bbox-layout: los modos de solo texto no acreditan.
	pdfToTextBinaryV0 = "pdftotext"
)

var (
	ErrDocumentRefVacioV0      = errors.New("document_extraction_pdf_document_ref_vacio")
	ErrDocumentoNoEncontradoV0 = errors.New("document_extraction_pdf_documento_no_encontrado")
	ErrExtractorNoDisponibleV0 = errors.New("document_extraction_pdf_extractor_no_disponible")
	ErrExtraccionFallidaV0     = errors.New("document_extraction_pdf_extraccion_fallida")
	ErrDocumentoSinPaginasV0   = errors.New("document_extraction_pdf_documento_sin_paginas")
)

type AdapterV0 struct {
	identity extraction.DocumentAdapterIdentityV0
	binary   string
}

var _ extraction.DocumentSourcePortV0 = (*AdapterV0)(nil)
var _ extraction.DocumentNormalizerPortV0 = (*AdapterV0)(nil)
var _ extraction.DocumentParserPortV0 = (*AdapterV0)(nil)

func NewAdapterV0() *AdapterV0 {
	return &AdapterV0{
		identity: extraction.DocumentAdapterIdentityV0{
			AdapterRef: AdapterRefV0,
			Version:    AdapterVersionV0,
		},
		binary: pdfToTextBinaryV0,
	}
}

func (adapter *AdapterV0) AdapterIdentityV0() extraction.DocumentAdapterIdentityV0 {
	return adapter.identity
}

func (adapter *AdapterV0) ResolveDocumentV0(
	_ context.Context,
	documentRef string,
) (extraction.DocumentSourceMaterialV0, error) {
	ref := strings.TrimSpace(documentRef)
	if ref == "" {
		return extraction.DocumentSourceMaterialV0{}, ErrDocumentRefVacioV0
	}
	bytes, err := os.ReadFile(ref)
	if err != nil {
		return extraction.DocumentSourceMaterialV0{}, fmt.Errorf("%w: %s", ErrDocumentoNoEncontradoV0, ref)
	}
	return extraction.DocumentSourceMaterialV0{
		DocumentRef: ref,
		SourceRef:   filepath.Base(ref),
		ContentHash: hashV0(bytes),
		MediaKind:   MediaKindV0,
		Bytes:       bytes,
	}, nil
}

func (adapter *AdapterV0) NormalizeDocumentV0(
	_ context.Context,
	source extraction.DocumentSourceMaterialV0,
	policy extraction.DocumentExtractionPolicyV0,
) (extraction.DocumentNormalizedMaterialV0, error) {
	if len(source.Bytes) == 0 {
		return extraction.DocumentNormalizedMaterialV0{}, ErrDocumentoNoEncontradoV0
	}
	return extraction.DocumentNormalizedMaterialV0{
		Source: source,
		Transforms: []extraction.DocumentTransformV0{{
			TransformRef: AdapterRefV0 + ":normalize",
			Kind:         "passthrough",
			Version:      AdapterVersionV0,
			ConfigHash:   policyHashV0(policy),
			InputHash:    source.ContentHash,
			OutputHash:   source.ContentHash,
		}},
	}, nil
}

func (adapter *AdapterV0) ParseDocumentV0(
	ctx context.Context,
	material extraction.DocumentNormalizedMaterialV0,
) (extraction.DocumentV0, error) {
	source := material.Source
	if strings.TrimSpace(source.DocumentRef) == "" {
		return extraction.DocumentV0{}, ErrDocumentRefVacioV0
	}
	layout, err := adapter.runPDFToTextV0(ctx, source.DocumentRef)
	if err != nil {
		return extraction.DocumentV0{}, err
	}
	if len(layout.Doc.Pages) == 0 {
		return extraction.DocumentV0{}, ErrDocumentoSinPaginasV0
	}
	provenance := extraction.DocumentProvenanceV0{
		SourceRefs:     []string{source.SourceRef},
		AdapterRef:     AdapterRefV0,
		AdapterVersion: AdapterVersionV0,
		TransformRefs:  transformRefsV0(material.Transforms),
	}
	pages := make([]extraction.DocumentPageV0, 0, len(layout.Doc.Pages))
	for pageIndex, page := range layout.Doc.Pages {
		pages = append(pages, extraction.DocumentPageV0{
			PageRef: fmt.Sprintf("page-%d", pageIndex+1),
			Index:   pageIndex,
			Width:   page.Width,
			Height:  page.Height,
			Blocks:  blocksForPageV0(pageIndex, page, provenance),
		})
	}
	return extraction.DocumentV0{
		IRVersion:   extraction.DocumentExtractionIRSchemaVersionV0,
		DocumentRef: source.DocumentRef,
		SourceRef:   source.SourceRef,
		ContentHash: source.ContentHash,
		MediaKind:   MediaKindV0,
		PageCount:   len(pages),
		Pages:       pages,
	}, nil
}

// runPDFToTextV0 invoca poppler en modo -bbox-layout, que es el unico que
// devuelve coordenadas por palabra. Sin coordenadas el nucleo rechaza la
// evidencia, asi que un fallo aqui es error tipado y nunca texto vacio.
func (adapter *AdapterV0) runPDFToTextV0(
	ctx context.Context,
	documentRef string,
) (pdfToTextLayoutV0, error) {
	binary, err := exec.LookPath(adapter.binary)
	if err != nil {
		return pdfToTextLayoutV0{}, fmt.Errorf("%w: %s", ErrExtractorNoDisponibleV0, adapter.binary)
	}
	command := exec.CommandContext(ctx, binary, "-bbox-layout", documentRef, "-")
	output, err := command.Output()
	if err != nil {
		return pdfToTextLayoutV0{}, fmt.Errorf("%w: %v", ErrExtraccionFallidaV0, err)
	}
	var layout pdfToTextLayoutV0
	if err := xml.Unmarshal(output, &layout); err != nil {
		return pdfToTextLayoutV0{}, fmt.Errorf("%w: %v", ErrExtraccionFallidaV0, err)
	}
	return layout, nil
}

func blocksForPageV0(
	pageIndex int,
	page pdfToTextPageV0,
	provenance extraction.DocumentProvenanceV0,
) []extraction.DocumentBlockV0 {
	blocks := make([]extraction.DocumentBlockV0, 0)
	readingOrder := 0
	for _, flow := range page.Flows {
		for _, block := range flow.Blocks {
			spans := spansForBlockV0(pageIndex, readingOrder, block, provenance)
			if len(spans) == 0 {
				continue
			}
			blocks = append(blocks, extraction.DocumentBlockV0{
				BlockRef:     fmt.Sprintf("page-%d-block-%d", pageIndex+1, readingOrder),
				Kind:         "text",
				ReadingOrder: readingOrder,
				BoundingBox:  boundingBoxV0(block.XMin, block.YMin, block.XMax, block.YMax),
				Provenance:   provenance,
				Spans:        spans,
			})
			readingOrder++
		}
	}
	return blocks
}

func spansForBlockV0(
	pageIndex int,
	blockOrder int,
	block pdfToTextBlockV0,
	provenance extraction.DocumentProvenanceV0,
) []extraction.DocumentSpanV0 {
	spans := make([]extraction.DocumentSpanV0, 0, len(block.Lines))
	for lineIndex, line := range block.Lines {
		words := make([]string, 0, len(line.Words))
		for _, word := range line.Words {
			if text := strings.TrimSpace(word.Text); text != "" {
				words = append(words, text)
			}
		}
		if len(words) == 0 {
			continue
		}
		spans = append(spans, extraction.DocumentSpanV0{
			SpanRef:     fmt.Sprintf("page-%d-block-%d-line-%d", pageIndex+1, blockOrder, lineIndex),
			TextRaw:     strings.Join(words, " "),
			BoundingBox: boundingBoxV0(line.XMin, line.YMin, line.XMax, line.YMax),
			Provenance:  provenance,
		})
	}
	return spans
}

func boundingBoxV0(xMin, yMin, xMax, yMax float64) *extraction.DocumentBoundingBoxV0 {
	return &extraction.DocumentBoundingBoxV0{
		X:      xMin,
		Y:      yMin,
		Width:  xMax - xMin,
		Height: yMax - yMin,
	}
}

func transformRefsV0(transforms []extraction.DocumentTransformV0) []string {
	refs := make([]string, 0, len(transforms))
	for _, transform := range transforms {
		refs = append(refs, transform.TransformRef)
	}
	return refs
}

func hashV0(bytes []byte) string {
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func policyHashV0(policy extraction.DocumentExtractionPolicyV0) string {
	return hashV0([]byte(fmt.Sprintf("%v", policy)))
}

type pdfToTextLayoutV0 struct {
	Doc pdfToTextDocV0 `xml:"body>doc"`
}

type pdfToTextDocV0 struct {
	Pages []pdfToTextPageV0 `xml:"page"`
}

type pdfToTextPageV0 struct {
	Width  float64           `xml:"width,attr"`
	Height float64           `xml:"height,attr"`
	Flows  []pdfToTextFlowV0 `xml:"flow"`
}

type pdfToTextFlowV0 struct {
	Blocks []pdfToTextBlockV0 `xml:"block"`
}

type pdfToTextBlockV0 struct {
	XMin  float64           `xml:"xMin,attr"`
	YMin  float64           `xml:"yMin,attr"`
	XMax  float64           `xml:"xMax,attr"`
	YMax  float64           `xml:"yMax,attr"`
	Lines []pdfToTextLineV0 `xml:"line"`
}

type pdfToTextLineV0 struct {
	XMin  float64           `xml:"xMin,attr"`
	YMin  float64           `xml:"yMin,attr"`
	XMax  float64           `xml:"xMax,attr"`
	YMax  float64           `xml:"yMax,attr"`
	Words []pdfToTextWordV0 `xml:"word"`
}

type pdfToTextWordV0 struct {
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
	Text string  `xml:",chardata"`
}
