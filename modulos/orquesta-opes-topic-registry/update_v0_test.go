package orquestaopestopicregistry

import (
	"context"
	"errors"
	"reflect"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestTopicRegistryCLIArgsV0UpdateConEvidencias(t *testing.T) {
	args := TopicRegistryCLIArgsV0(TopicRegistryUpdateRequestV0{
		Action:       "update",
		CourseID:     "curso-a2",
		TopicID:      "tema-001",
		AgentID:      "agent-001",
		Status:       "en_progreso_orquesta",
		Summary:      "Resumen",
		Done:         "artifact-001",
		Pending:      "audio",
		EvidenceRefs: []string{"evidence-001", "evidence-001", "evidence-002"},
	})
	want := []string{
		"update",
		"--course-id", "curso-a2",
		"--topic-id", "tema-001",
		"--agent-id", "agent-001",
		"--status", "en_progreso_orquesta",
		"--summary", "Resumen",
		"--done", "artifact-001",
		"--pending", "audio",
		"--evidence-ref", "evidence-001",
		"--evidence-ref", "evidence-002",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args=%v want=%v", args, want)
	}
}

func TestApplyTopicRegistryUpdateV0EjecutaRunnerInyectado(t *testing.T) {
	runner := &fakeTopicRegistryRunnerV0{}
	result, err := ApplyTopicRegistryUpdateV0(context.Background(), TopicRegistryUpdateRequestV0{
		ToolPath: "/tmp/registro_trabajo_temas.py",
		Action:   TopicRegistryActionReleaseV0,
		CourseID: "curso-a2",
		TopicID:  "tema-001",
		Status:   "paquete_final_local_verificable",
		Force:    true,
	}, runner)
	if err != nil {
		t.Fatalf("ApplyTopicRegistryUpdateV0: %v", err)
	}
	if result.Status != TopicRegistryUpdateStatusAppliedV0 ||
		len(runner.invocations) != 1 ||
		runner.invocations[0].ToolPath != "/tmp/registro_trabajo_temas.py" ||
		runner.invocations[0].Args[0] != TopicRegistryActionReleaseV0 {
		t.Fatalf("result=%+v invocations=%+v", result, runner.invocations)
	}
	if !containsStringForTopicRegistryTestV0(runner.invocations[0].Args, "--force") {
		t.Fatalf("args=%+v", runner.invocations[0].Args)
	}
}

func TestApplyTopicRegistryUpdateV0InvalidoSinToolPath(t *testing.T) {
	result, err := ApplyTopicRegistryUpdateV0(context.Background(), TopicRegistryUpdateRequestV0{
		CourseID: "curso-a2",
		TopicID:  "tema-001",
	}, &fakeTopicRegistryRunnerV0{})
	if err != nil {
		t.Fatalf("ApplyTopicRegistryUpdateV0: %v", err)
	}
	if result.Status != TopicRegistryUpdateStatusInvalidV0 ||
		len(result.Issues) != 1 ||
		result.Issues[0].Code != ErrTopicRegistryToolPathRequiredV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestApplyTopicRegistryUpdateV0DevuelveFalloDeComando(t *testing.T) {
	runner := &fakeTopicRegistryRunnerV0{
		result: TopicRegistryCommandResultV0{ExitCode: 2},
		err:    errors.New("exit status 2"),
	}
	result, err := ApplyTopicRegistryUpdateV0(context.Background(), TopicRegistryUpdateRequestV0{
		ToolPath: "/tmp/registro_trabajo_temas.py",
		CourseID: "curso-a2",
		TopicID:  "tema-001",
	}, runner)
	if err == nil ||
		result.Status != TopicRegistryUpdateStatusFailedV0 ||
		len(result.Issues) != 1 ||
		result.Issues[0].Code != ErrTopicRegistryCommandFailedV0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTopicRegistryUpdateRequestFromDomainWorkJobV0(t *testing.T) {
	request := TopicRegistryUpdateRequestFromDomainWorkJobV0(orquestadomainwork.DomainWorkJobRequestV0{
		Objective:    "Actualizar registro",
		EvidenceRefs: []string{"evidence-job-001"},
		InputFields: []orquestadomainwork.DomainWorkFieldV0{
			{Name: "registry_action", Value: "release"},
			{Name: "course_id", Value: "curso-a2"},
			{Name: "topic_id", Value: "tema-001"},
			{Name: "proposed_status", Value: "paquete_final_local_verificable"},
			{Name: "source_summary", Value: "Paquete final aceptado."},
			{Name: "done_refs", Values: []string{"artifact-001", "receipt-001"}},
			{Name: "pending_refs", Values: []string{"audio-qa"}},
		},
	}, "/tmp/tool.py", "orquesta-agent")
	if request.ToolPath != "/tmp/tool.py" ||
		request.Action != TopicRegistryActionReleaseV0 ||
		request.CourseID != "curso-a2" ||
		request.TopicID != "tema-001" ||
		request.AgentID != "orquesta-agent" ||
		request.Status != "paquete_final_local_verificable" ||
		request.Summary != "Paquete final aceptado." ||
		request.Done != "artifact-001, receipt-001" ||
		request.Pending != "audio-qa" ||
		len(request.EvidenceRefs) != 1 ||
		request.EvidenceRefs[0] != "evidence-job-001" {
		t.Fatalf("request=%+v", request)
	}
}

type fakeTopicRegistryRunnerV0 struct {
	invocations []TopicRegistryCommandInvocationV0
	result      TopicRegistryCommandResultV0
	err         error
}

func (runner *fakeTopicRegistryRunnerV0) RunTopicRegistryCommandV0(
	_ context.Context,
	invocation TopicRegistryCommandInvocationV0,
) (TopicRegistryCommandResultV0, error) {
	runner.invocations = append(runner.invocations, invocation)
	return runner.result, runner.err
}

func containsStringForTopicRegistryTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
