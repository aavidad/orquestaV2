package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"

	"orquesta/internal/config"
	"orquesta/internal/identity"
)

func TestV11BootstrapSelectsRealOIDCProviderWithoutLocalProvisioning(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer *httptest.Server
	issuer = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"issuer": issuer.URL, "jwks_uri": issuer.URL + "/keys",
				"authorization_endpoint": issuer.URL + "/authorize", "token_endpoint": issuer.URL + "/token",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/keys":
			_ = json.NewEncoder(writer).Encode(map[string]any{"keys": []jose.JSONWebKey{{
				Key: &key.PublicKey, KeyID: "bootstrap-v11", Algorithm: "RS256", Use: "sig",
			}}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer issuer.Close()

	tokenPath := filepath.Join(t.TempDir(), "must-not-exist", "local.token")
	snapshot, err := config.Resolve(config.ResolveOptions{TOML: []byte(fmt.Sprintf(`
[identity]
provider = "oidc"
local_token_path = %q

[identity.oidc]
issuer = %q
audience = "orquesta-bootstrap-v11"
required_groups = ["orquesta-users"]
clock_skew = "30s"
`, tokenPath, issuer.URL))})
	if err != nil {
		t.Fatalf("resolve OIDC config: %v", err)
	}
	composition, err := composeIdentityRuntime(context.Background(), snapshot, issuer.Client())
	if err != nil {
		t.Fatalf("compose OIDC identity: %v", err)
	}
	if composition.provider.AuthenticationMethod() != "oidc" || composition.provisionLocal ||
		composition.localPrincipal.Ref.String() != "" || composition.localHierarchy.ProjectRef().String() != "" {
		t.Fatalf("OIDC composition leaked local authority: %+v", composition)
	}
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Fatalf("OIDC composition touched local token path: %v", err)
	}

	now := time.Now().UTC()
	claims, err := json.Marshal(map[string]any{
		"iss": issuer.URL, "aud": "orquesta-bootstrap-v11", "sub": "alice",
		"iat": now.Add(-time.Minute).Unix(), "nbf": now.Add(-time.Minute).Unix(),
		"exp": now.Add(time.Minute).Unix(), "groups": []string{"orquesta-users"},
	})
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader(jose.HeaderKey("kid"), "bootstrap-v11"),
	)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(claims)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	credential, err := identity.NewCredential([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	principal, err := composition.provider.Authenticate(context.Background(), credential)
	if err != nil {
		t.Fatalf("authenticate through selected provider: %v", err)
	}
	want, err := identity.StablePrincipal("oidc", issuer.URL, "alice", identity.PrincipalKindHuman)
	if err != nil || principal != want {
		t.Fatalf("selected provider principal=%+v want=%+v err=%v", principal, want, err)
	}
}
