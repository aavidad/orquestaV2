package orquestacore

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

var (
	_ PersistenceRepositoryPortV0  = fakePersistenceRepositoryPortV0{}
	_ OrquestaEventPublisherPortV0 = fakeOrquestaEventPublisherPortV0{}
	_ RuntimeLauncherPortV0        = fakeRuntimeLauncherPortV0{}
	_ GovernanceCatalogPortV0      = fakeGovernanceCatalogPortV0{}
)

func TestNewGuardarProyectoBorradorPersistenceRequestV0(t *testing.T) {
	cmd, accepted := acceptedProyectoV0(t)

	request := NewGuardarProyectoBorradorPersistenceRequestV0(cmd, accepted)

	if request.SchemaVersion != PersistenceRepositoryContractVersionV0 || request.Contrato != PersistenceRepositoryContractVersionV0 {
		t.Fatalf("persistence contract mismatch: %+v", request)
	}
	if request.Operacion != PersistenceRepositoryOperationGuardarProyectoBorradorV0 {
		t.Fatalf("operacion=%q", request.Operacion)
	}
	if request.IdempotencyKey != cmd.IdempotencyKey {
		t.Fatalf("idempotency_key=%q, want %q", request.IdempotencyKey, cmd.IdempotencyKey)
	}
	if request.PayloadVersion != ProyectoPlanBorradorPayloadVersionV0 {
		t.Fatalf("payload_version=%q", request.PayloadVersion)
	}
	if request.Payload.ProyectoIDPropuesto != accepted.ProyectoPlanBorrador.ProyectoIDPropuesto {
		t.Fatalf("payload project id mismatch")
	}
	if request.Payload.Estado != EstadoProyectoPlanBorradorV0 {
		t.Fatalf("payload should remain draft: %+v", request.Payload)
	}
}

func TestNewPublishOrquestaEventRequestsV0MapsDomainEvents(t *testing.T) {
	cmd, accepted := acceptedProyectoV0(t)

	requests := NewPublishOrquestaEventRequestsV0(cmd, accepted, true)

	if len(requests) != len(accepted.EventosDominio) {
		t.Fatalf("requests=%d, want %d", len(requests), len(accepted.EventosDominio))
	}
	request := requests[0]
	if !request.DryRun {
		t.Fatalf("dry_run should be true")
	}
	if request.IdempotencyKey == "" || request.IdempotencyKey == cmd.IdempotencyKey {
		t.Fatalf("event idempotency key should be derived, got %q", request.IdempotencyKey)
	}
	if request.Event.SchemaVersion != OrquestaEventContractVersionV0 {
		t.Fatalf("event schema=%q", request.Event.SchemaVersion)
	}
	if request.Event.EventID == "" {
		t.Fatalf("event_id required")
	}
	if request.Event.SourceArea != OrquestaEventSourceAreaCoreV0 {
		t.Fatalf("source_area=%q", request.Event.SourceArea)
	}
	if request.Event.EventType != OrquestaEventTypeProyectoRegistradoEnBorradorV0 {
		t.Fatalf("event_type=%q", request.Event.EventType)
	}
	if request.Event.Severity != OrquestaEventSeverityInfoV0 || request.Event.Outcome != OrquestaEventOutcomeAcceptedV0 {
		t.Fatalf("event severity/outcome mismatch: %+v", request.Event)
	}
	if request.Event.Correlation.CorrelationID != cmd.CorrelationID || request.Event.Correlation.RequestID != cmd.RequestID {
		t.Fatalf("correlation mismatch: %+v", request.Event.Correlation)
	}
	if request.Event.Subject.Kind != OrquestaEventSubjectKindProjectV0 ||
		request.Event.Subject.ID != accepted.ProyectoPlanBorrador.ProyectoIDPropuesto ||
		request.Event.Subject.Version != ProyectoPlanBorradorPayloadVersionV0 {
		t.Fatalf("subject mismatch: %+v", request.Event.Subject)
	}
	if request.Event.Producer.Module != OrquestaEventProducerModuleCoreV0 || request.Event.Producer.Port != OrquestaEventProducerPortCoreV0 {
		t.Fatalf("producer mismatch: %+v", request.Event.Producer)
	}
	if request.Event.Privacy.ContainsSecret || request.Event.Privacy.ContainsTranscript {
		t.Fatalf("privacy flags should be false: %+v", request.Event.Privacy)
	}
	if got := request.Event.Payload["microtareas"]; got != len(accepted.ProyectoPlanBorrador.Microtareas) {
		t.Fatalf("microtareas payload=%v", got)
	}

	accepted.EventosDominio[0].Payload["microtareas"] = 99
	if got := request.Event.Payload["microtareas"]; got == 99 {
		t.Fatalf("event payload should be copied")
	}
}

func TestOrquestaEventV0SerializesWithGlobalShapeAndNoForbiddenPayloadFields(t *testing.T) {
	cmd, accepted := acceptedProyectoV0(t)
	request := NewPublishOrquestaEventRequestsV0(cmd, accepted, false)[0]

	payload, err := json.Marshal(request.Event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	for _, required := range []string{
		"schema_version",
		"event_id",
		"event_type",
		"source_area",
		"occurred_at",
		"severity",
		"outcome",
		"correlation",
		"subject",
		"producer",
		"privacy",
		"summary",
		"payload",
	} {
		if _, ok := object[required]; !ok {
			t.Fatalf("missing required event field %q in %s", required, string(payload))
		}
	}
	for _, forbidden := range []string{"correlation_id", "references"} {
		if _, ok := object[forbidden]; ok {
			t.Fatalf("unexpected top-level field %q in %s", forbidden, string(payload))
		}
	}
	assertNoForbiddenPayloadKeysV0(t, request.Event.Payload)

	serialized := strings.ToLower(string(payload))
	for _, forbidden := range []string{"token", "password", "api_key", "oauth", "prompt", "completion", "raw_text", "full_text", "sql", "dsn", "connection", "table", "provider", "model_name", "home_path"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized event contains forbidden marker %q: %s", forbidden, string(payload))
		}
	}
}

func TestOutputPortDTOsSerializeWithoutAdapterCommands(t *testing.T) {
	cmd, accepted := acceptedProyectoV0(t)
	persistenceRequest := NewGuardarProyectoBorradorPersistenceRequestV0(cmd, accepted)
	eventRequests := NewPublishOrquestaEventRequestsV0(cmd, accepted, false)

	payload, err := json.Marshal(struct {
		Persistence GuardarProyectoBorradorPersistenceRequestV0 `json:"persistence"`
		Events      []PublishOrquestaEventRequestV0             `json:"events"`
	}{
		Persistence: persistenceRequest,
		Events:      eventRequests,
	})
	if err != nil {
		t.Fatalf("marshal output port DTOs: %v", err)
	}

	serialized := string(payload)
	for _, forbidden := range []string{"dsn", "sql", "tmux", "docker", "token"} {
		if strings.Contains(strings.ToLower(serialized), forbidden) {
			t.Fatalf("serialized DTO contains forbidden adapter detail %q: %s", forbidden, serialized)
		}
	}
}

func TestNewGovernanceCatalogRequestV0CompactsScope(t *testing.T) {
	request := NewGovernanceCatalogRequestV0(" req-1 ", " corr-1 ", GovernanceCatalogScopeV0{
		Modulo: " orquesta-core ",
		Rol:    " core ",
		Fase:   " revision ",
		Tags:   []string{" microtarea ", "", "microtarea", " runtime "},
	})

	if request.SchemaVersion != GovernanceCatalogContractVersionV0 {
		t.Fatalf("schema_version=%q", request.SchemaVersion)
	}
	if request.RequestID != "req-1" || request.CorrelationID != "corr-1" {
		t.Fatalf("request metadata not trimmed: %+v", request)
	}
	if request.Scope.Modulo != "orquesta-core" || request.Scope.Fase != "revision" {
		t.Fatalf("scope not trimmed: %+v", request.Scope)
	}
	if len(request.Scope.Tags) != 2 || request.Scope.Tags[0] != "microtarea" || request.Scope.Tags[1] != "runtime" {
		t.Fatalf("tags not compacted: %+v", request.Scope.Tags)
	}
}

func acceptedProyectoV0(t *testing.T) (RegistrarProyectoDesdeAppSpecCommandV0, RegistroProyectoAceptadoV0) {
	t.Helper()
	cmd := validRegistrarProyectoCommandV0(t)
	accepted, err := RegistrarProyectoDesdeAppSpecV0(cmd)
	if err != nil {
		t.Fatalf("registrar proyecto: %v", err)
	}
	return cmd, accepted
}

type fakePersistenceRepositoryPortV0 struct{}

func (fakePersistenceRepositoryPortV0) GuardarProyectoBorradorV0(context.Context, GuardarProyectoBorradorPersistenceRequestV0) (GuardarProyectoBorradorPersistenceAcceptedV0, error) {
	return GuardarProyectoBorradorPersistenceAcceptedV0{}, nil
}

type fakeOrquestaEventPublisherPortV0 struct{}

func (fakeOrquestaEventPublisherPortV0) PublicarOrquestaEventV0(context.Context, PublishOrquestaEventRequestV0) (PublishOrquestaEventAcceptedV0, error) {
	return PublishOrquestaEventAcceptedV0{}, nil
}

type fakeRuntimeLauncherPortV0 struct{}

func (fakeRuntimeLauncherPortV0) LanzarRuntimeV0(context.Context, RuntimeLaunchRequestCoreV0) (RuntimeLaunchAcceptedCoreV0, error) {
	return RuntimeLaunchAcceptedCoreV0{}, nil
}

type fakeGovernanceCatalogPortV0 struct{}

func (fakeGovernanceCatalogPortV0) ConsultarGovernanceCatalogV0(context.Context, GovernanceCatalogRequestV0) (GovernanceCatalogSnapshotV0, error) {
	return GovernanceCatalogSnapshotV0{}, nil
}

func assertNoForbiddenPayloadKeysV0(t *testing.T, values map[string]any) {
	t.Helper()
	for key, value := range values {
		lower := strings.ToLower(key)
		for _, forbidden := range []string{"secret", "token", "password", "credential", "api_key", "oauth", "transcript", "prompt", "completion", "raw_text", "full_text", "sql", "dsn", "connection", "table", "provider", "model_name", "home_path"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("forbidden payload key %q", key)
			}
		}
		if nested, ok := value.(map[string]any); ok {
			assertNoForbiddenPayloadKeysV0(t, nested)
		}
	}
}
