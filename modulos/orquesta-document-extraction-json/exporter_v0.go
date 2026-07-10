package orquestadocumentextractionjson

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
)

type JSONDocumentExporterV0 struct {
	Identity       orquestadocumentextraction.DocumentAdapterIdentityV0
	DestinationRef string
}

func (exporter JSONDocumentExporterV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return exporter.Identity
}

func (exporter JSONDocumentExporterV0) ExportDocumentFieldsV0(_ context.Context, request orquestadocumentextraction.DocumentExportRequestV0) (orquestadocumentextraction.DocumentExportArtifactV0, error) {
	if exporter.DestinationRef == "" {
		return orquestadocumentextraction.DocumentExportArtifactV0{}, fmt.Errorf("document_json_destination_ref_required")
	}
	payload, err := json.Marshal(struct {
		DocumentRef  string                                                `json:"document_ref"`
		SchemaRef    string                                                `json:"schema_ref"`
		OutputLocale string                                                `json:"output_locale"`
		Fields       []orquestadocumentextraction.DocumentFieldCandidateV0 `json:"fields"`
	}{request.DocumentRef, request.SchemaRef, request.OutputLocale, request.Fields})
	if err != nil {
		return orquestadocumentextraction.DocumentExportArtifactV0{}, err
	}
	hash := sha256.Sum256(payload)
	return orquestadocumentextraction.DocumentExportArtifactV0{ArtifactRef: "document-export:json:" + request.DocumentRef, DestinationRef: exporter.DestinationRef, Format: "json", ContentHash: hex.EncodeToString(hash[:]), Bytes: payload}, nil
}
