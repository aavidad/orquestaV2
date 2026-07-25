package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/i18n"
)

type codedWorkerTestError struct {
	message string
	cause   string
}

func (err *codedWorkerTestError) Error() string     { return err.message }
func (err *codedWorkerTestError) CauseCode() string { return err.cause }

func TestWorkerErrorLogAttrsIncludeOnlySafeTypedCause(t *testing.T) {
	attributes := workerErrorLogAttrs(&codedWorkerTestError{
		message: "application.effect_unknown_applied",
		cause:   "test_attestor.snapshot_limit_exceeded",
	})
	if len(attributes) != 2 ||
		attributes[0].Key != "code" || attributes[0].Value.String() != "application.effect_unknown_applied" ||
		attributes[1].Key != "cause_code" ||
		attributes[1].Value.String() != "test_attestor.snapshot_limit_exceeded" {
		t.Fatalf("typed attributes=%+v", attributes)
	}
	for _, err := range []error{
		&codedWorkerTestError{message: "application.effect_unknown_applied", cause: "secret\nvalue"},
		errors.New("application.effect_unknown_applied"),
	} {
		attributes = workerErrorLogAttrs(err)
		if len(attributes) != 1 || attributes[0].Key != "code" {
			t.Fatalf("unsafe attributes=%+v", attributes)
		}
	}
}

func TestMain(testMain *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "__orquesta_internal_codex_supervisor_v1" {
		os.Exit(run(os.Args[1:], io.Discard, io.Discard))
	}
	os.Exit(testMain.Run())
}

func TestVersionAndInvalidCommandDoNotStartRuntime(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 || strings.TrimSpace(stdout.String()) != version {
		t.Fatalf("version: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "solicitud") {
		t.Fatalf("invalid: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestPrivateCodexSupervisorDispatchFailsClosedWithoutDescriptors(t *testing.T) {
	command := exec.Command(os.Args[0], "__orquesta_internal_codex_supervisor_v1")
	command.Env = []string{}
	err := command.Run()
	exitError, ok := err.(*exec.ExitError)
	if !ok || exitError.ExitCode() != 125 {
		exitCode := -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
		t.Fatalf("descriptorless private dispatch error=%v exit=%d, want 125", err, exitCode)
	}
}

func TestCommandUsesExplicitConnectionCredentialAndGeneratedCLIPath(t *testing.T) {
	const token = "explicit-test-token"
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		if request.URL.Path != "/api/v1/commands/orquesta.system.status" ||
			request.Header.Get("Authorization") != "Bearer "+token {
			t.Fatalf("request path=%q auth=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		var input struct {
			Version    string          `json:"version"`
			RequestRef string          `json:"request_ref"`
			ProjectRef string          `json:"project_ref"`
			Payload    json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil ||
			input.Version != "1" || input.RequestRef != "request:cli" ||
			input.ProjectRef != "project:cli" || string(input.Payload) != "{}" {
			t.Fatalf("input=%+v err=%v", input, err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"command_id":"orquesta.system.status","command_version":"1","request_ref":"request:cli","data":{"goals":0,"running_goals":0,"pending_actions":0,"quarantined_actions":0},"audit_ref":"command-audit:cli"}`))
	}))
	t.Cleanup(server.Close)
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"command", "--url", server.URL, "--credential-file", credentialPath,
		"--max-credential-bytes", "128", "--max-response-bytes", "4096", "--timeout", "2s",
		"--request-ref", "request:cli",
		"--project-ref", "project:cli", "--payload", "{}", "--", "system", "status",
	}, &stdout, &stderr)
	if code != 0 || !called || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"audit_ref":"command-audit:cli"`) {
		t.Fatalf("code=%d called=%v stdout=%q stderr=%q", code, called, stdout.String(), stderr.String())
	}
}

func TestCommandRejectsNonPrivateCredentialBeforeNetwork(t *testing.T) {
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"command", "--url", "http://127.0.0.1:1", "--credential-file", credentialPath,
		"--max-credential-bytes", "128", "--max-response-bytes", "4096", "--timeout", "2s",
		"--request-ref", "request:cli",
		"--project-ref", "project:cli", "--payload", "{}", "--", "system", "status",
	}, &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "cli.credential_invalid") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestCommandRejectsRedirectWithoutForwardingCredential(t *testing.T) {
	const token = "redirect-secret"
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	var targetCalled bool
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalled = true
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Fatalf("initial authorization=%q", request.Header.Get("Authorization"))
		}
		http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirect.Close)
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"command", "--url", redirect.URL, "--credential-file", credentialPath,
		"--max-credential-bytes", "128", "--max-response-bytes", "4096", "--timeout", "2s",
		"--request-ref", "request:redirect", "--project-ref", "project:cli", "--payload", "{}",
		"--", "system", "status",
	}, &stdout, &stderr)
	if code != 1 || targetCalled || stdout.Len() != 0 {
		t.Fatalf("code=%d target=%v stdout=%q stderr=%q", code, targetCalled, stdout.String(), stderr.String())
	}
}

func TestCommandTimeoutIsExplicitAndEnforced(t *testing.T) {
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte("timeout-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(server.Close)
	var stdout, stderr bytes.Buffer
	started := time.Now()
	code := run([]string{
		"command", "--url", server.URL, "--credential-file", credentialPath,
		"--max-credential-bytes", "128", "--max-response-bytes", "4096", "--timeout", "20ms",
		"--request-ref", "request:timeout", "--project-ref", "project:cli", "--payload", "{}",
		"--", "system", "status",
	}, &stdout, &stderr)
	close(release)
	if code != 1 || time.Since(started) > time.Second || stdout.Len() != 0 {
		t.Fatalf("code=%d elapsed=%s stdout=%q stderr=%q", code, time.Since(started), stdout.String(), stderr.String())
	}
}

func TestCommandRejectsInsecureRemoteURL(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"command", "--url", "http://example.com", "--credential-file", "/unused",
		"--max-credential-bytes", "128", "--max-response-bytes", "4096", "--timeout", "2s",
		"--request-ref", "request:remote", "--project-ref", "project:cli", "--payload", "{}",
		"--", "system", "status",
	}, &stdout, &stderr)
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	want, err := catalog.Text(i18n.DefaultLocale, "error.cli.url_insecure")
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 || !strings.Contains(stderr.String(), want) ||
		!strings.Contains(stderr.String(), "code=cli.url_insecure") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
