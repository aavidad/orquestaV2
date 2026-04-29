package db

import "testing"

func TestExtraerMetadataResumeSesionConservaHintsOperativosDePerfilEjecucion(t *testing.T) {
	meta := extraerMetadataResumeSesion(&Sesion{
		ResumePayloadJSON: `{"perfil_ejecucion":{"driver":"tmux_cli_session","transport":"tmux","perfil_operativo":"persistente","worktree_path":"/tmp/orquesta/.orquesta-worktrees/orq-codex1","tmux_session":"orq-codex1","tmux_pane_id":"%7","mailbox_delivery_mode":"session_resume","can_send_input":false}}`,
	})
	if got := stringFromMap(meta, "driver", ""); got != "tmux_cli_session" {
		t.Fatalf("driver inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "transport", ""); got != "tmux" {
		t.Fatalf("transport inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "perfil_operativo", ""); got != "persistente" {
		t.Fatalf("perfil_operativo inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "worktree_path", ""); got != "/tmp/orquesta/.orquesta-worktrees/orq-codex1" {
		t.Fatalf("worktree_path inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "tmux_session", ""); got != "orq-codex1" {
		t.Fatalf("tmux_session inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "tmux_pane_id", ""); got != "%7" {
		t.Fatalf("tmux_pane_id inesperado: %+v", meta)
	}
	if got, ok := meta["can_send_input"].(bool); !ok || got {
		t.Fatalf("can_send_input inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "execution_profile", ""); got != "persistente" {
		t.Fatalf("execution_profile inesperado: %+v", meta)
	}
}

func TestExtraerMetadataResumeSesionAceptaExecutionProfileCanonico(t *testing.T) {
	meta := extraerMetadataResumeSesion(&Sesion{
		ResumePayloadJSON: `{"perfil_ejecucion":{"driver":"tmux_cli_session","transport":"tmux","execution_profile":"qa-heavy","worktree_path":"/tmp/orquesta/.orquesta-worktrees/orq-codex2","tmux_session":"orq-codex2","tmux_pane_id":"%9"}}`,
	})
	if got := stringFromMap(meta, "execution_profile", ""); got != "qa-heavy" {
		t.Fatalf("execution_profile canonico inesperado: %+v", meta)
	}
	if got := stringFromMap(meta, "perfil_operativo", ""); got != "qa-heavy" {
		t.Fatalf("perfil_operativo espejado inesperado: %+v", meta)
	}
}
