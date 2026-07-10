package orquestadocumentextractionjson_test

import (
	"context"
	"encoding/json"
	"testing"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
	orquestadocumentextractionjson "orquesta/modulos/orquesta-document-extraction-json"
)

func TestJSONDocumentExporterV0PreservesRawAndNormalizedValues(t *testing.T) {
	exporter := orquestadocumentextractionjson.JSONDocumentExporterV0{Identity: orquestadocumentextraction.DocumentAdapterIdentityV0{AdapterRef: "json", Version: "1"}, DestinationRef: "destination:json"}
	artifact, err := exporter.ExportDocumentFieldsV0(context.Background(), orquestadocumentextraction.DocumentExportRequestV0{DocumentRef: "document:1", SchemaRef: "schema:1", OutputLocale: "es-ES", Fields: []orquestadocumentextraction.DocumentFieldCandidateV0{{FieldRef: "field:1", ValueRaw: "01/02/2026", ValueNormalized: &orquestadocumentextraction.DocumentNormalizedValueV0{Value: "2026-02-01", NormalizationLocale: "es-ES", RuleRef: "rule:date", ValidatorVersion: "1"}, ValueState: orquestadocumentextraction.DocumentValueStatePresentV0}}})
	if err != nil || artifact.ContentHash == "" {
		t.Fatalf("export = %#v, %v", artifact, err)
	}
	var payload struct {
		Fields []orquestadocumentextraction.DocumentFieldCandidateV0 `json:"fields"`
	}
	if err := json.Unmarshal(artifact.Bytes, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Fields) != 1 || payload.Fields[0].ValueRaw != "01/02/2026" || payload.Fields[0].ValueNormalized.Value != "2026-02-01" {
		t.Fatalf("payload = %#v", payload)
	}
}
