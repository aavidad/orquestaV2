package main

import (
	"context"
	"fmt"

	"path/filepath"
	"strings"

	extraction "orquesta/modulos/orquesta-document-extraction"
	pdf "orquesta/modulos/orquesta-document-extraction-pdf"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	pptx "orquesta/modulos/orquesta-presentation-extraction-openxml"
)

// documentTextExtractExecutorV0 cablea la capacidad de extraccion documental
// sobre el adaptador real. El adaptador va confinado a la raiz de ingesta: el
// document_ref que llega por MCP es relativo a ella y nunca una ruta del host.
type documentTextExtractExecutorV0 struct {
	adapter  *pdf.AdapterV0
	policy   extraction.DocumentExtractionPolicyV0
	inboxDir string
}

var _ orquestamcp.MCPDocumentTextExtractExtractorPortV0 = documentTextExtractExecutorV0{}

func newDocumentTextExtractExecutorV0(rootDir string) (documentTextExtractExecutorV0, error) {
	adapter, err := pdf.NewAdapterV0(pdf.ConfigV0{RootDir: rootDir})
	if err != nil {
		return documentTextExtractExecutorV0{}, fmt.Errorf("document text extract: %w", err)
	}
	return documentTextExtractExecutorV0{
		adapter:  adapter,
		policy:   extraction.DefaultDocumentExtractionPolicyV0(),
		inboxDir: rootDir,
	}, nil
}

// Las presentaciones (.pptx) proyectan AL MISMO documento intermedio que los PDF,
// asi que no hacen falta tool ni superficie nuevas: se despacha por extension y el
// resto del camino -paginacion, anclas, presupuesto del transporte- es identico.
func (executor documentTextExtractExecutorV0) presentationDocumentV0(
	ctx context.Context,
	documentRef string,
) (extraction.DocumentV0, error) {
	ref := strings.TrimSpace(documentRef)
	adapter, err := pptx.NewAdapterV0(pptx.ConfigV0{
		RootDir: executor.inboxDir,
		Catalog: []pptx.CatalogEntryV0{{
			PresentationRef: ref,
			Path:            ref,
			SourceRef:       ref,
		}},
	})
	if err != nil {
		return extraction.DocumentV0{}, err
	}
	source, err := adapter.ResolvePresentationV0(ctx, ref)
	if err != nil {
		return extraction.DocumentV0{}, err
	}
	return adapter.ProjectPresentationDocumentV0(ctx, source)
}

func esPresentacionV0(documentRef string) bool {
	return strings.EqualFold(filepath.Ext(strings.TrimSpace(documentRef)), ".pptx")
}

func (executor documentTextExtractExecutorV0) ExtractDocumentTextV0(
	ctx context.Context,
	input orquestamcp.MCPDocumentTextExtractToolInputV0,
) (orquestamcp.MCPDocumentTextExtractToolResultV0, error) {
	var document extraction.DocumentV0
	if esPresentacionV0(input.DocumentRef) {
		proyectado, err := executor.presentationDocumentV0(ctx, input.DocumentRef)
		if err != nil {
			return orquestamcp.MCPDocumentTextExtractToolResultV0{}, err
		}
		document = proyectado
	} else {
		source, err := executor.adapter.ResolveDocumentV0(ctx, input.DocumentRef)
		if err != nil {
			return orquestamcp.MCPDocumentTextExtractToolResultV0{}, err
		}
		normalized, err := executor.adapter.NormalizeDocumentV0(ctx, source, executor.policy)
		if err != nil {
			return orquestamcp.MCPDocumentTextExtractToolResultV0{}, err
		}
		parsed, err := executor.adapter.ParseDocumentV0(ctx, normalized)
		if err != nil {
			return orquestamcp.MCPDocumentTextExtractToolResultV0{}, err
		}
		document = parsed
	}

	pageFrom, pageLimit := orquestamcp.NormalizeMCPDocumentTextExtractPagingV0(input)
	if pageFrom > document.PageCount {
		pageFrom = document.PageCount
	}
	pageTo := pageFrom + pageLimit - 1
	if pageTo > document.PageCount {
		pageTo = document.PageCount
	}

	pages := make([]orquestamcp.MCPDocumentTextExtractPageV0, 0, pageTo-pageFrom+1)
	spans := 0
	for index := pageFrom - 1; index >= 0 && index < pageTo && index < len(document.Pages); index++ {
		page := document.Pages[index]
		lines := make([]string, 0)
		for _, block := range page.Blocks {
			for _, span := range block.Spans {
				lines = append(lines, span.TextRaw)
				spans++
			}
		}
		pages = append(pages, orquestamcp.MCPDocumentTextExtractPageV0{
			PageRef: page.PageRef,
			Index:   page.Index,
			Width:   page.Width,
			Height:  page.Height,
			Lines:   lines,
		})
	}

	return orquestamcp.MCPDocumentTextExtractToolResultV0{
		DocumentRef:  document.DocumentRef,
		SourceRef:    document.SourceRef,
		ContentHash:  document.ContentHash,
		MediaKind:    document.MediaKind,
		AdapterRef:   executor.adapter.AdapterIdentityV0().AdapterRef,
		PageCount:    document.PageCount,
		PageFrom:     pageFrom,
		PageTo:       pageTo,
		SpanCount:    spans,
		HasMorePages: pageTo < document.PageCount,
		Pages:        pages,
	}, nil
}
