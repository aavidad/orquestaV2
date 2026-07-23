package bootstrap

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
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
