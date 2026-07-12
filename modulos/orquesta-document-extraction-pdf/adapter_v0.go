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

const (
	DefaultMaxDocumentBytesV0 = 64 << 20
	DefaultMaxPageCountV0     = 500
)

var (
	ErrDocumentRefVacioV0         = errors.New("document_extraction_pdf_document_ref_vacio")
	ErrDocumentoNoEncontradoV0    = errors.New("document_extraction_pdf_documento_no_encontrado")
	ErrExtractorNoDisponibleV0    = errors.New("document_extraction_pdf_extractor_no_disponible")
	ErrExtraccionFallidaV0        = errors.New("document_extraction_pdf_extraccion_fallida")
	ErrDocumentoSinPaginasV0      = errors.New("document_extraction_pdf_documento_sin_paginas")
	ErrRaizNoConfiguradaV0        = errors.New("document_extraction_pdf_raiz_no_configurada")
	ErrDocumentRefFueraDeRaizV0   = errors.New("document_extraction_pdf_document_ref_fuera_de_raiz")
	ErrDocumentoDemasiadoGrandeV0 = errors.New("document_extraction_pdf_documento_demasiado_grande")
	ErrDemasiadasPaginasV0        = errors.New("document_extraction_pdf_demasiadas_paginas")
)

// ConfigV0 confina el adaptador a una raiz de ingesta. El document_ref que
// llega por la tool es SIEMPRE relativo a esa raiz: aceptar rutas absolutas del
// host seria inservible dentro del contenedor y abriria lectura arbitraria de
// ficheros.
type ConfigV0 struct {
	RootDir  string
	MaxBytes int64
	MaxPages int
}

type AdapterV0 struct {
	identity extraction.DocumentAdapterIdentityV0
	binary   string
	root     string
	maxBytes int64
	maxPages int
}

var _ extraction.DocumentSourcePortV0 = (*AdapterV0)(nil)
var _ extraction.DocumentNormalizerPortV0 = (*AdapterV0)(nil)
var _ extraction.DocumentParserPortV0 = (*AdapterV0)(nil)

func NewAdapterV0(config ConfigV0) (*AdapterV0, error) {
	root := strings.TrimSpace(config.RootDir)
	if root == "" {
		return nil, ErrRaizNoConfiguradaV0
	}
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRaizNoConfiguradaV0, err)
	}
	if resolved, err := filepath.EvalSymlinks(resolvedRoot); err == nil {
		resolvedRoot = resolved
	}
	maxBytes := config.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxDocumentBytesV0
	}
	maxPages := config.MaxPages
	if maxPages <= 0 {
		maxPages = DefaultMaxPageCountV0
	}
	return &AdapterV0{
		identity: extraction.DocumentAdapterIdentityV0{
			AdapterRef: AdapterRefV0,
			Version:    AdapterVersionV0,
		},
		binary:   pdfToTextBinaryV0,
		root:     resolvedRoot,
		maxBytes: maxBytes,
		maxPages: maxPages,
	}, nil
}

// resolveWithinRootV0 traduce un document_ref catalogado a una ruta real dentro
// de la raiz. Rechaza rutas absolutas, escapes por ".." y enlaces que salgan de
// la raiz: el escape se comprueba despues de resolver symlinks, no antes.
func (adapter *AdapterV0) resolveWithinRootV0(documentRef string) (string, error) {
	ref := strings.TrimSpace(documentRef)
	if ref == "" {
		return "", ErrDocumentRefVacioV0
	}
	if filepath.IsAbs(ref) {
		return "", fmt.Errorf("%w: %s", ErrDocumentRefFueraDeRaizV0, ref)
	}
	candidate := filepath.Join(adapter.root, filepath.Clean("/"+ref))
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		candidate = resolved
	}
	relative, err := filepath.Rel(adapter.root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", ErrDocumentRefFueraDeRaizV0, ref)
	}
	return candidate, nil
}

func (adapter *AdapterV0) AdapterIdentityV0() extraction.DocumentAdapterIdentityV0 {
	return adapter.identity
}

func (adapter *AdapterV0) ResolveDocumentV0(
	_ context.Context,
	documentRef string,
) (extraction.DocumentSourceMaterialV0, error) {
	path, err := adapter.resolveWithinRootV0(documentRef)
	if err != nil {
		return extraction.DocumentSourceMaterialV0{}, err
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return extraction.DocumentSourceMaterialV0{}, fmt.Errorf("%w: %s", ErrDocumentoNoEncontradoV0, documentRef)
	}
	if info.Size() > adapter.maxBytes {
		return extraction.DocumentSourceMaterialV0{}, fmt.Errorf(
			"%w: %d bytes, limite %d", ErrDocumentoDemasiadoGrandeV0, info.Size(), adapter.maxBytes,
		)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return extraction.DocumentSourceMaterialV0{}, fmt.Errorf("%w: %s", ErrDocumentoNoEncontradoV0, documentRef)
	}
	return extraction.DocumentSourceMaterialV0{
		DocumentRef: strings.TrimSpace(documentRef),
		SourceRef:   filepath.Base(path),
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
	path, err := adapter.resolveWithinRootV0(source.DocumentRef)
	if err != nil {
		return extraction.DocumentV0{}, err
	}
	layout, err := adapter.runPDFToTextV0(ctx, path)
	if err != nil {
		return extraction.DocumentV0{}, err
	}
	if len(layout.Doc.Pages) == 0 {
		return extraction.DocumentV0{}, ErrDocumentoSinPaginasV0
	}
	if len(layout.Doc.Pages) > adapter.maxPages {
		return extraction.DocumentV0{}, fmt.Errorf(
			"%w: %d paginas, limite %d", ErrDemasiadasPaginasV0, len(layout.Doc.Pages), adapter.maxPages,
		)
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
