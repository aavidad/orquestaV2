package orquestaappcodexstack

import "testing"

func TestCodexAreaV0ClasificaImplementacionComoProgramacion(t *testing.T) {
	if got := codexAreaV0("implementacion", "task-ref-stack-agenda-001"); got != "programacion" {
		t.Fatalf("area=%s", got)
	}
}

func TestCodexAreaV0ClasificaDominioComoProgramacion(t *testing.T) {
	if got := codexAreaV0("dominio", "task-ref-domain-work-001"); got != "programacion" {
		t.Fatalf("area=%s", got)
	}
}

func TestCodexProfileForAreaV0AplicaPermisosEspecificosDelDirector(t *testing.T) {
	config := CodexRuntimeConfigV0{
		Sandbox:                "workspace-write",
		ApprovalPolicy:         "never",
		DirectorSandbox:        "danger-full-access",
		DirectorApprovalPolicy: "on-request",
	}

	director := codexProfileForAreaV0(config, "/tmp/runtime-director", "director")
	worker := codexProfileForAreaV0(config, "/tmp/runtime-worker", "programacion")

	if director.Sandbox != "danger-full-access" || director.ApprovalPolicy != "on-request" {
		t.Fatalf("director profile=%+v", director)
	}
	if worker.Sandbox != "workspace-write" || worker.ApprovalPolicy != "never" {
		t.Fatalf("worker profile=%+v", worker)
	}
}
