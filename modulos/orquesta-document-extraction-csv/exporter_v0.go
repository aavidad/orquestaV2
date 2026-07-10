package orquestadocumentextractioncsv

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"strings"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
)

type CSVDocumentExporterV0 struct {
	Identity       orquestadocumentextraction.DocumentAdapterIdentityV0
	DestinationRef string
}

func (exporter CSVDocumentExporterV0) AdapterIdentityV0() orquestadocumentextraction.DocumentAdapterIdentityV0 {
	return exporter.Identity
}

func (exporter CSVDocumentExporterV0) ExportDocumentFieldsV0(_ context.Context, request orquestadocumentextraction.DocumentExportRequestV0) (orquestadocumentextraction.DocumentExportArtifactV0, error) {
	if exporter.DestinationRef == "" {
		return orquestadocumentextraction.DocumentExportArtifactV0{}, fmt.Errorf("document_csv_destination_ref_required")
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write([]string{"field_ref", "occurrence_ref", "value_raw", "value_normalized", "value_state", "evidence_refs"}); err != nil {
		return orquestadocumentextraction.DocumentExportArtifactV0{}, err
	}
	for _, field := range request.Fields {
		normalized := ""
		if field.ValueNormalized != nil {
			normalized = field.ValueNormalized.Value
		}
		if err := writer.Write([]string{field.FieldRef, field.OccurrenceRef, field.ValueRaw, normalized, string(field.ValueState), strings.Join(field.EvidenceRefs, "|")}); err != nil {
			return orquestadocumentextraction.DocumentExportArtifactV0{}, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return orquestadocumentextraction.DocumentExportArtifactV0{}, err
	}
	payload := output.Bytes()
	hash := sha256.Sum256(payload)
	return orquestadocumentextraction.DocumentExportArtifactV0{ArtifactRef: "document-export:csv:" + request.DocumentRef, DestinationRef: exporter.DestinationRef, Format: "csv", ContentHash: hex.EncodeToString(hash[:]), Bytes: payload}, nil
}
