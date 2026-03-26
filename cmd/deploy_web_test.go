package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testMuxDeployWeb() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/deploy", webHandlerDeploy)
	registerAPIRoutes(mux)
	return mux
}

func TestWebDeployPlanYExecuteDryRunPorAPI(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/deploy?lang=en", nil)
	testMuxDeployWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deploy status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, token := range []string{"<html lang=\"en\">", "Remote deploy"} {
		if !strings.Contains(body, token) {
			t.Fatalf("vista deploy incompleta, falta %q:\n%s", token, body)
		}
	}

	form := url.Values{
		"proyecto":        {"demo"},
		"servicio":        {"web"},
		"ssh_host":        {"srv.example.com"},
		"ssh_user":        {"deploy"},
		"ssh_port":        {"22"},
		"remote_dir":      {"/srv/demo"},
		"image":           {"registry/demo-web"},
		"tag":             {"2026.03.25"},
		"rollback_tag":    {"2026.03.24"},
		"strategy":        {"docker_save"},
		"compose_file":    {"docker-compose.yml"},
		"env_file":        {".env"},
		"dockerfile":      {"Dockerfile"},
		"build_context":   {"."},
		"healthcheck_cmd": {"true"},
		"build":           {"on"},
		"dry_run":         {"on"},
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/deploy?lang=en", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxDeployWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deploy plan status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	for _, token := range []string{"Plan generated", "prepare_remote_dir", "srv.example.com"} {
		if !strings.Contains(body, token) {
			t.Fatalf("plan incompleto, falta %q:\n%s", token, body)
		}
	}

	form.Set("accion", "ejecutar")
	form.Set("auto_rollback", "on")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/deploy?lang=en", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testMuxDeployWeb().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deploy execute status=%d body=%s", rec.Code, rec.Body.String())
	}
	body = rec.Body.String()
	for _, token := range []string{"Dry-run executed", "docker build", "docker save"} {
		if !strings.Contains(body, token) {
			t.Fatalf("resultado deploy incompleto, falta %q:\n%s", token, body)
		}
	}
}
