package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/i18n"
)

func TestCommandRejectsSymlinkAndOversizedCredentials(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		path    string
		maximum int64
	}{
		{name: "symlink", path: link, maximum: 128},
		{name: "oversized", path: target, maximum: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := readCommandCredential(test.path, test.maximum); err == nil {
				t.Fatal("unsafe credential accepted")
			}
		})
	}
}

func TestCommandCredentialTransportRejectsCrossOrigin(t *testing.T) {
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte("origin-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	credential, err := readCommandCredential(credentialPath, 128)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	transport := commandCredentialTransport{
		credential: credential,
		origin:     "https://trusted.example",
		base: commandCLIRoundTripFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, nil
		}),
	}
	request, err := http.NewRequest(http.MethodPost, "https://other.example/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(request); err == nil || called {
		t.Fatalf("cross-origin request accepted err=%v called=%v", err, called)
	}
}

type commandCLIRoundTripFunc func(*http.Request) (*http.Response, error)

func (function commandCLIRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestCommandHelpUsesLocaleAndSpanishFallback(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"es", "en", "gl"} {
		t.Run(locale, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := RunCommand([]string{"--locale", locale, "--help"}, catalog, &stdout, &stderr)
			want, err := catalog.Text(locale, commandUsageKey)
			if err != nil {
				t.Fatal(err)
			}
			if code != 0 || stderr.Len() != 0 || strings.TrimSpace(stdout.String()) != want {
				t.Fatalf("code=%d stdout=%q stderr=%q want=%q", code, stdout.String(), stderr.String(), want)
			}
		})
	}
}

func TestCommandRejectsInvalidLocaleWithLocalizedDiagnosticAndStableCode(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := RunCommand([]string{"--locale", "not_a_locale!"}, catalog, &stdout, &stderr)
	want, err := catalog.Text(i18n.DefaultLocale, "error.cli.locale_invalid")
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), want) ||
		!strings.Contains(stderr.String(), "code=cli.locale_invalid") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestCommandRejectsMissingArgumentsWithLocalizedDiagnosticAndStableCode(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := RunCommand(nil, catalog, &stdout, &stderr)
	want, err := catalog.Text(i18n.DefaultLocale, "error.cli.command_arguments_invalid")
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), want) ||
		!strings.Contains(stderr.String(), "code=cli.command_arguments_invalid") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestCommandRejectsRedirectWithLocalizedDiagnosticAndStableCode(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte("redirect-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalled = true
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer redirect-secret" {
			t.Fatalf("initial authorization=%q", request.Header.Get("Authorization"))
		}
		http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirect.Close)
	var stdout, stderr bytes.Buffer
	code := RunCommand([]string{
		"--locale", "en",
		"--url", redirect.URL,
		"--credential-file", credentialPath,
		"--max-credential-bytes", "128",
		"--max-response-bytes", "4096",
		"--timeout", "2s",
		"--request-ref", "request:redirect",
		"--project-ref", "project:v21",
		"--payload", "{}",
		"--", "system", "status",
	}, catalog, &stdout, &stderr)
	want, err := catalog.Text("en", "error.cli.redirect_rejected")
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || targetCalled || stdout.Len() != 0 || strings.TrimSpace(stderr.String()) != want+" code=cli.redirect_rejected" {
		t.Fatalf("code=%d target=%v stdout=%q stderr=%q want=%q", code, targetCalled, stdout.String(), stderr.String(), want)
	}
}

func TestCommandLocaleDoesNotChangeMachineResult(t *testing.T) {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	credentialPath := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(credentialPath, []byte("locale-test-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"command_id":"orquesta.system.status","command_version":"1","request_ref":"request:i18n-cli","data":{"digest":"sha256:locale-invariant","ref":"goal:locale-invariant"},"audit_ref":"command-audit:locale-invariant"}`))
	}))
	t.Cleanup(server.Close)
	var reference string
	for _, locale := range []string{"es", "en", "gl"} {
		var stdout, stderr bytes.Buffer
		code := RunCommand([]string{
			"--locale", locale,
			"--url", server.URL,
			"--credential-file", credentialPath,
			"--max-credential-bytes", "128",
			"--max-response-bytes", "4096",
			"--timeout", "2s",
			"--request-ref", "request:i18n-cli",
			"--project-ref", "project:v21",
			"--payload", "{}",
			"--", "system", "status",
		}, catalog, &stdout, &stderr)
		if code != 0 || stderr.Len() != 0 || !json.Valid(stdout.Bytes()) {
			t.Fatalf("locale=%s code=%d stdout=%q stderr=%q", locale, code, stdout.String(), stderr.String())
		}
		if reference == "" {
			reference = stdout.String()
		} else if stdout.String() != reference {
			t.Fatalf("locale=%s output=%q reference=%q", locale, stdout.String(), reference)
		}
	}
}
