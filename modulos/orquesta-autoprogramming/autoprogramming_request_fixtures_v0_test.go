package orquestaautoprogramming

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type autoprogrammingRequestFixtureV0 struct {
	SchemaVersion          string                   `json:"schema_version"`
	FixtureRef             string                   `json:"fixture_ref"`
	SourceSurface          string                   `json:"source_surface"`
	Transport              string                   `json:"transport"`
	Endpoint               string                   `json:"endpoint,omitempty"`
	ToolName               string                   `json:"tool_name,omitempty"`
	ResourceURI            string                   `json:"resource_uri,omitempty"`
	RequestID              string                   `json:"request_id"`
	CorrelationID          string                   `json:"correlation_id"`
	RequestedBy            string                   `json:"requested_by"`
	PriorityScore          int                      `json:"priority_score"`
	AutoprogrammingRequest AutoprogrammingRequestV0 `json:"autoprogramming_request"`
}

func TestAutoprogrammingRequestV0FixturesRealesWebYMCPValidanContrato(t *testing.T) {
	for _, path := range []string{
		"docs/fixtures/autoprogramming_request_v0/web_prepare_run_real.json",
		"docs/fixtures/autoprogramming_request_v0/mcp_prepare_run_real.json",
	} {
		t.Run(path, func(t *testing.T) {
			fixture := readAutoprogrammingRequestFixtureV0(t, path)
			assertAutoprogrammingRequestFixtureEnvelopeV0(t, fixture)

			source := autoprogrammingFixtureSourceV0(fixture)
			sourceValidation := ValidateAutoprogrammingRequestSourceV0(source)
			if !sourceValidation.Accepted {
				t.Fatalf("source invalida: issues=%+v", sourceValidation.Issues)
			}
			validation := ValidateAutoprogrammingRequestV0(fixture.AutoprogrammingRequest)
			if !validation.Accepted {
				t.Fatalf("fixture invalida: issues=%+v", validation.Issues)
			}
			work := BuildAutoprogrammingProgrammableWorkV0(fixture.AutoprogrammingRequest)
			if !work.Accepted {
				t.Fatalf("work no aceptado: issues=%+v", work.Issues)
			}
			if work.Work.WorktreeRef != fixture.AutoprogrammingRequest.WorktreeRef ||
				work.Work.BranchRef != fixture.AutoprogrammingRequest.BranchRef ||
				len(work.Work.Tasks) != 1 {
				t.Fatalf("refs/work no preservados: work=%+v fixture=%+v", work.Work, fixture)
			}
			assertAutoprogrammingFixtureContextRefV0(
				t,
				work.Work.Tasks[0].ContextRefs,
				"source_surface:"+fixture.SourceSurface,
			)
			for _, ref := range sourceValidation.SourceRefs {
				assertAutoprogrammingFixtureContextRefV0(t, work.Work.Tasks[0].ContextRefs, ref)
			}
			assertAutoprogrammingFixtureContextRefV0(
				t,
				work.Work.Tasks[0].ContextRefs,
				"source_task_ref:task-ref-self-improvement-ab390583089c",
			)
		})
	}
}

func readAutoprogrammingRequestFixtureV0(t *testing.T, path string) autoprogrammingRequestFixtureV0 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	var fixture autoprogrammingRequestFixtureV0
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}
	return fixture
}

func autoprogrammingFixtureSourceV0(fixture autoprogrammingRequestFixtureV0) AutoprogrammingRequestSourceV0 {
	return AutoprogrammingRequestSourceV0{
		SchemaVersion:          AutoprogrammingRequestSourceSchemaVersionV0,
		SourceRef:              fixture.FixtureRef,
		SourceSurface:          fixture.SourceSurface,
		Transport:              fixture.Transport,
		Endpoint:               fixture.Endpoint,
		ToolName:               fixture.ToolName,
		ResourceURI:            fixture.ResourceURI,
		RequestID:              fixture.RequestID,
		CorrelationID:          fixture.CorrelationID,
		RequestedBy:            fixture.RequestedBy,
		PriorityScore:          fixture.PriorityScore,
		AutoprogrammingRequest: fixture.AutoprogrammingRequest,
	}
}

func assertAutoprogrammingRequestFixtureEnvelopeV0(
	t *testing.T,
	fixture autoprogrammingRequestFixtureV0,
) {
	t.Helper()
	if fixture.SchemaVersion != "autoprogramming_request_fixture.v0" ||
		fixture.FixtureRef == "" ||
		fixture.RequestID == "" ||
		fixture.CorrelationID == "" ||
		fixture.RequestedBy == "" ||
		fixture.PriorityScore <= 0 {
		t.Fatalf("fixture envelope incompleto: %+v", fixture)
	}
	if fixture.SourceSurface == "web" && fixture.Endpoint != "/api/v0/autoprogramming/prepare-run" {
		t.Fatalf("endpoint web inesperado: %+v", fixture)
	}
	if fixture.SourceSurface == "mcp" &&
		(fixture.ToolName != "orquesta.autoprogramming.prepare_run.v0" ||
			fixture.ResourceURI == "") {
		t.Fatalf("tool mcp inesperado: %+v", fixture)
	}
}

func assertAutoprogrammingFixtureContextRefV0(t *testing.T, refs []string, want string) {
	t.Helper()
	for _, ref := range refs {
		if strings.TrimSpace(ref) == want {
			return
		}
	}
	t.Fatalf("context_ref %q no encontrado en %v", want, refs)
}
