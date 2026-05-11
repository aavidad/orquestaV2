package orquestapersistence

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDecodeGuardarProyectoBorradorMaterialV0FixtureValido(t *testing.T) {
	material, issues := DecodeGuardarProyectoBorradorMaterialV0(readFixtureV0(t, "guardar_proyecto_borrador_valido.json"))
	if len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if material.Request.IdempotencyKey != material.Response.IdempotencyKey {
		t.Fatalf("idempotency_key mismatch: request=%q response=%q", material.Request.IdempotencyKey, material.Response.IdempotencyKey)
	}
	if material.Request.PayloadVersion != ProyectoPlanBorradorPayloadVersionV0 {
		t.Fatalf("payload_version=%q", material.Request.PayloadVersion)
	}
}

func TestDecodeGuardarProyectoBorradorMaterialV0RechazaConectorDirectoEnRequest(t *testing.T) {
	_, issues := DecodeGuardarProyectoBorradorMaterialV0(readFixtureV0(t, "conector_directo_invalido.json"))
	if !HasPersistenceRepositoryIssueV0(issues, ErrPersistenceConnectorNoSoportadoV0, "request.connector_ref") {
		t.Fatalf("expected request.connector_ref issue, got %+v", issues)
	}
}

func TestDecodeGuardarProyectoBorradorMaterialV0RechazaQueryEnContratoMaterial(t *testing.T) {
	_, issues := DecodeGuardarProyectoBorradorMaterialV0(readFixtureV0(t, "regla_negocio_en_query_invalido.json"))
	if !HasPersistenceRepositoryIssueV0(issues, ErrReglaNegocioEnAdaptadorV0, "query") {
		t.Fatalf("expected query issue, got %+v", issues)
	}
}

func TestValidateGuardarProyectoBorradorRequestV0RequiereIdempotencyKey(t *testing.T) {
	request := validRequestV0(t)
	request.IdempotencyKey = "  "

	issues := ValidateGuardarProyectoBorradorRequestV0(request)
	if !HasPersistenceRepositoryIssueV0(issues, ErrPayloadInvalidoV0, "idempotency_key") {
		t.Fatalf("expected idempotency_key issue, got %+v", issues)
	}
}

func TestValidateGuardarProyectoBorradorRequestV0RequierePayloadVersionSoportada(t *testing.T) {
	request := validRequestV0(t)
	request.PayloadVersion = "ProyectoPlanBorradorV1"

	issues := ValidateGuardarProyectoBorradorRequestV0(request)
	if !HasPersistenceRepositoryIssueV0(issues, ErrPayloadInvalidoV0, "payload_version") {
		t.Fatalf("expected payload_version issue, got %+v", issues)
	}
}

func TestDecodeGuardarProyectoBorradorRequestV0RechazaConectorYQuery(t *testing.T) {
	rawRequest := map[string]any{}
	data, err := json.Marshal(validRequestV0(t))
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if err := json.Unmarshal(data, &rawRequest); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	rawRequest["connector_ref"] = "persistence-connector-ref-forbidden-001"
	rawRequest["query"] = map[string]any{"tipo": "filtro_negocio"}
	data, err = json.Marshal(rawRequest)
	if err != nil {
		t.Fatalf("marshal raw request: %v", err)
	}

	_, issues := DecodeGuardarProyectoBorradorRequestV0(data)
	if !HasPersistenceRepositoryIssueV0(issues, ErrPersistenceConnectorNoSoportadoV0, "connector_ref") {
		t.Fatalf("expected connector_ref issue, got %+v", issues)
	}
	if !HasPersistenceRepositoryIssueV0(issues, ErrReglaNegocioEnAdaptadorV0, "query") {
		t.Fatalf("expected query issue, got %+v", issues)
	}
}

func TestDecodeGuardarProyectoBorradorMaterialV0RechazaDetallesConcretosEnRequestYMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]any)
		code   string
		field  string
	}{
		{
			name: "request database",
			mutate: func(material map[string]any) {
				requestObjectV0(t, material)["database"] = "forbidden"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "request.database",
		},
		{
			name: "request db",
			mutate: func(material map[string]any) {
				requestObjectV0(t, material)["db"] = "forbidden"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "request.db",
		},
		{
			name: "request driver",
			mutate: func(material map[string]any) {
				requestObjectV0(t, material)["driver"] = "forbidden"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "request.driver",
		},
		{
			name: "request connector ref",
			mutate: func(material map[string]any) {
				requestObjectV0(t, material)["connector_ref"] = "forbidden"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "request.connector_ref",
		},
		{
			name: "request engine name",
			mutate: func(material map[string]any) {
				requestObjectV0(t, material)["idempotency_key"] = "idem_postgres_001"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "request.idempotency_key",
		},
		{
			name: "metadata driver",
			mutate: func(material map[string]any) {
				adapterMetadataObjectV0(t, material)["driver"] = "forbidden"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "adapter_metadata.driver",
		},
		{
			name: "metadata connector ref",
			mutate: func(material map[string]any) {
				adapterMetadataObjectV0(t, material)["connector_ref"] = "forbidden"
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "adapter_metadata.connector_ref",
		},
		{
			name: "metadata engine name",
			mutate: func(material map[string]any) {
				adapterMetadataObjectV0(t, material)["connector_profile_refs"] = []any{"sqlite_connector_profile_ref_001"}
			},
			code:  ErrPersistenceConnectorNoSoportadoV0,
			field: "adapter_metadata.connector_profile_refs.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material := validMaterialMapV0(t)
			tt.mutate(material)
			data, err := json.Marshal(material)
			if err != nil {
				t.Fatalf("marshal material: %v", err)
			}

			_, issues := DecodeGuardarProyectoBorradorMaterialV0(data)
			if !HasPersistenceRepositoryIssueV0(issues, tt.code, tt.field) {
				t.Fatalf("expected %s at %s, got %+v", tt.code, tt.field, issues)
			}
		})
	}
}

func validRequestV0(t *testing.T) GuardarProyectoBorradorRequestV0 {
	t.Helper()
	material, issues := DecodeGuardarProyectoBorradorMaterialV0(readFixtureV0(t, "guardar_proyecto_borrador_valido.json"))
	if len(issues) != 0 {
		t.Fatalf("fixture invalido: %+v", issues)
	}
	return material.Request
}

func validMaterialMapV0(t *testing.T) map[string]any {
	t.Helper()
	var material map[string]any
	if err := json.Unmarshal(readFixtureV0(t, "guardar_proyecto_borrador_valido.json"), &material); err != nil {
		t.Fatalf("unmarshal valid material: %v", err)
	}
	return material
}

func requestObjectV0(t *testing.T, material map[string]any) map[string]any {
	t.Helper()
	return objectFieldV0(t, material, "request")
}

func adapterMetadataObjectV0(t *testing.T, material map[string]any) map[string]any {
	t.Helper()
	return objectFieldV0(t, material, "adapter_metadata")
}

func objectFieldV0(t *testing.T, object map[string]any, field string) map[string]any {
	t.Helper()
	child, ok := object[field].(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object", field)
	}
	return child
}

func readFixtureV0(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("docs/fixtures/persistence_repository_v0/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
