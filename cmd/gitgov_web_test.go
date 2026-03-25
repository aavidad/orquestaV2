package cmd

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/db"
)

func testMuxGitGovWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/git", webHandlerGitGov)
	mux.HandleFunc("/git/", webRouterGitGov)
	registerAPIRoutes(mux)
	return mux
}

func TestWebGitGovUsaAPIParaWorktreesYLocks(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	lock, err := db.CoordinationLockRepository().Create(&coordinacion.Lock{
		ProjectID:   &proyectoID,
		Agent:       "Codex1",
		ScopeType:   "project",
		ScopeKey:    "orquestador",
		Path:        filepath.Join(tmp, "orquestador"),
		Branch:      "main",
		Reason:      "prueba web git",
		LeaseToken:  "lease-1",
		State:       coordinacion.LockActive,
		HeartbeatAt: time.Now(),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("crear lock: %v", err)
	}

	worktree, err := db.CoordinationWorktreeRepository().Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		LockID:    &lock.ID,
		Agent:     "Codex1",
		Name:      "wt-codex1",
		Path:      filepath.Join(tmp, "orquestador-wt"),
		Branch:    "feature/server-first",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "prueba web git",
	})
	if err != nil {
		t.Fatalf("crear worktree: %v", err)
	}

	form := strings.NewReader("proyecto_slug=orquestador&source_branch=feature/server-first&target_branch=main&requested_by=Codex1&estado=pendiente&commit_origen=abc123&commit_merge=def456&notas=merge+web")
	req := httptest.NewRequest(http.MethodPost, "/git/merges", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	testMuxGitGovWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("alta merge status=%d body=%s", rec.Code, rec.Body.String())
	}

	merges, err := db.ListarGitMerges(&proyectoID, "pendiente")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("merges esperados=1 got=%d", len(merges))
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/git?estado=activa&agente=Codex1&proyecto=orquestador&merge_estado=pendiente&lang=en", nil)
	testMuxGitGovWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("gitgov status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"orquestador", "Codex1", "wt-codex1", "feature/server-first", "abc123", "def456", "merge web"} {
		if !strings.Contains(body, token) {
			t.Fatalf("panel gitgov sin %q:\n%s", token, body)
		}
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/git/worktrees/"+itoa(worktree.ID), nil)
	testMuxGitGovWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle worktree status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "feature/server-first") {
		t.Fatalf("detalle worktree incompleto:\n%s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/git/locks/"+itoa(lock.ID), nil)
	testMuxGitGovWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detalle lock status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "lease-1") {
		t.Fatalf("detalle lock incompleto:\n%s", rec.Body.String())
	}
}
