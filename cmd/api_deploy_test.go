package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"orquesta/deployapp"
)

func TestAPIDeployDockerRemotePlanAndExecuteDryRun(t *testing.T) {
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	payload := apiDeployDockerRemoteRequest{
		Spec: structToSpec(),
	}
	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/deploy/docker-remoto/plan", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan status=%d body=%s", rec.Code, rec.Body.String())
	}

	execPayload := apiDeployDockerRemoteRequest{
		Spec:         structToSpec(),
		DryRun:       true,
		AutoRollback: true,
	}
	execBody, _ := json.Marshal(execPayload)
	recExec := httptest.NewRecorder()
	reqExec := httptest.NewRequest(http.MethodPost, "/api/deploy/docker-remoto/ejecutar", bytes.NewReader(execBody))
	mux.ServeHTTP(recExec, reqExec)
	if recExec.Code != http.StatusOK {
		t.Fatalf("execute status=%d body=%s", recExec.Code, recExec.Body.String())
	}
	var resp apiDeployDockerRemoteExecuteResponse
	if err := json.Unmarshal(recExec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Plan == nil || resp.Result == nil {
		t.Fatalf("response incompleta: %+v", resp)
	}
	if len(resp.Result.Executed) == 0 {
		t.Fatalf("executed vacío: %+v", resp.Result)
	}
}

func structToSpec() deployapp.DockerRemoteSpec {
	return deployapp.DockerRemoteSpec{
		ProyectoSlug: "demo",
		Servicio:     "web",
		SSHHost:      "srv.example.com",
		SSHUser:      "deploy",
		Image:        "registry/demo-web",
		Tag:          "2026.03.25",
		RollbackTag:  "2026.03.24",
	}
}
