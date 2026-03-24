package rpclocal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServerHealthHandler(t *testing.T) {
	mux := NewMux(&Server{
		State: State{
			Addr:      "127.0.0.1:17899",
			PID:       100,
			StartedAt: time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC),
			Version:   "dev",
		},
	})

	req := httptest.NewRequest(http.MethodGet, HealthPath, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"pid":100`) {
		t.Fatalf("body inesperado: %s", body)
	}
}

func TestServerExecHandler(t *testing.T) {
	mux := NewMux(&Server{
		State: State{Addr: "127.0.0.1:17899", Token: "secret-token"},
		Executor: func(ctx context.Context, req *ExecRequest) (*ExecResponse, error) {
			if len(req.Args) != 1 || req.Args[0] != "status" {
				t.Fatalf("args inesperados: %+v", req.Args)
			}
			return &ExecResponse{Stdout: "ok\n", ExitCode: 0}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, ExecPath, strings.NewReader(`{"args":["status"]}`))
	req.Header.Set(HeaderAuthToken, "secret-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"exit_code":0`) {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestServerExecHandlerRejectsMissingToken(t *testing.T) {
	mux := NewMux(&Server{
		State: State{Addr: "127.0.0.1:17899", Token: "secret-token"},
		Executor: func(ctx context.Context, req *ExecRequest) (*ExecResponse, error) {
			t.Fatalf("no deberia ejecutar con token ausente")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, ExecPath, strings.NewReader(`{"args":["status"]}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
}
