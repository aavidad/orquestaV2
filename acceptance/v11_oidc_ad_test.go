package acceptance_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const v11FixturePath = "acceptance/fixtures/v11_oidc_ad.json"
const v11TrustedBaseGitCommitOID = "3108caa7e7f3f0a7b693e3027ef7cb3e56f462d9"
const v11ProductDeltaBaseGitCommitOID = "3108caa7e7f3f0a7b693e3027ef7cb3e56f462d9"
const v11ProductDeltaSealedGitCommitOID = "0000000000000000000000000000000000000000"

type v11Fixture struct {
	SchemaVersion                  int         `json:"schema_version"`
	ReceiptSchemaVersion           int         `json:"receipt_schema_version"`
	ContractID                     string      `json:"contract_id"`
	TrustedBaseGitCommitOID        string      `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string      `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string      `json:"product_delta_sealed_git_commit_oid"`
	Command                        string      `json:"command"`
	ExecutionArgv                  []string    `json:"execution_argv"`
	OutputPath                     string      `json:"output_path"`
	ReceiptPath                    string      `json:"receipt_path"`
	CandidateSubjects              []string    `json:"candidate_subjects"`
	OwnedCapabilityIDs             []string    `json:"owned_capability_ids"`
	DeferredSurfaces               []string    `json:"deferred_surfaces"`
	Scenario                       v11Scenario `json:"scenario"`
	Assertions                     []string    `json:"assertions"`
}

type v11Scenario struct {
	BaseTime              string   `json:"base_time"`
	ProviderConfigKey     string   `json:"provider_config_key"`
	Methods               []string `json:"methods"`
	OIDCIssuer            string   `json:"oidc_issuer"`
	OIDCAudience          string   `json:"oidc_audience"`
	OIDCSubject           string   `json:"oidc_subject"`
	PrincipalRefAlgorithm string   `json:"principal_ref_algorithm"`
	ExpectedPrincipalRef  string   `json:"expected_principal_ref"`
	ExpectedActorRef      string   `json:"expected_actor_ref"`
	PKCEMethod            string   `json:"pkce_method"`
	State                 string   `json:"state"`
	Nonce                 string   `json:"nonce"`
	JWKSKids              []string `json:"jwks_kids"`
	ExternalGroups        []string `json:"external_groups"`
	RequiredGroups        []string `json:"required_groups"`
	DexConnector          string   `json:"dex_connector"`
	DirectoryTransport    string   `json:"directory_transport"`
	TokenSentinel         string   `json:"token_sentinel"`
	CanonicalConfigKeys   []string `json:"canonical_config_keys"`
}

type v11Registry struct {
	Keys []v11RegistryKey `json:"keys"`
}

type v11RegistryKey struct {
	Key           string   `json:"key"`
	Type          string   `json:"type"`
	Default       any      `json:"default"`
	Sensitive     bool     `json:"sensitive"`
	AllowedValues []string `json:"allowed_values"`
}

func TestAcceptanceV11OIDCAD(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v11Fixture](t, filepath.Join(repositoryRoot, filepath.FromSlash(v11FixturePath)))
	v11AssertFixtureHeader(t, repositoryRoot, fixture)

	t.Run("one_identity_provider_port_and_two_adapters", func(t *testing.T) {
		v11AssertProviderShape(t, repositoryRoot)
	})
	t.Run("stable_subject_identity_and_no_group_rbac", func(t *testing.T) {
		v11AssertStableIdentity(t, repositoryRoot, fixture)
	})
	t.Run("oidc_validation_client_flow_and_jwks_rotation", func(t *testing.T) {
		v11AssertOIDCShape(t, repositoryRoot)
	})
	t.Run("canonical_config_has_no_oidc_or_directory_secret", func(t *testing.T) {
		v11AssertCanonicalConfig(t, repositoryRoot, fixture)
	})
	t.Run("classic_ad_is_external_dex_ldaps_bridge", func(t *testing.T) {
		v11AssertExternalADBridge(t, repositoryRoot, fixture)
	})
}

func v11AssertFixtureHeader(t *testing.T, repositoryRoot string, fixture v11Fixture) {
	t.Helper()
	wantDeferred := []string{
		"automatic_external_group_to_rbac_mapping",
		"direct_in_process_ldap_adapter",
		"kerberos",
		"saml",
		"server_side_oidc_login_session",
	}
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.ContractID != "AC-V11-OIDC-AD" ||
		fixture.TrustedBaseGitCommitOID != v11TrustedBaseGitCommitOID ||
		fixture.ProductDeltaBaseGitCommitOID != v11ProductDeltaBaseGitCommitOID ||
		fixture.ProductDeltaSealedGitCommitOID != v11ProductDeltaSealedGitCommitOID ||
		fixture.OutputPath != "product/evidence/v11_oidc_ad.output.txt" ||
		fixture.ReceiptPath != "product/evidence/v11_oidc_ad.json" ||
		len(fixture.OwnedCapabilityIDs) != 0 ||
		!reflect.DeepEqual(fixture.DeferredSurfaces, wantDeferred) || len(fixture.Assertions) != 16 {
		t.Fatalf("invalid V11 fixture header: %+v", fixture)
	}
	if _, err := time.Parse(time.RFC3339Nano, fixture.Scenario.BaseTime); err != nil {
		t.Fatalf("invalid V11 base time: %v", err)
	}
	if fixture.Command != "sh -c '"+v11ValidationShellBody()+"'" ||
		!reflect.DeepEqual(fixture.ExecutionArgv, []string{"sh", "-c", v11ValidationShellBody()}) {
		t.Fatalf("invalid V11 command/argv: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	if err := evidenceValidateCandidateSubjects(fixture.CandidateSubjects, fixture.ReceiptPath, fixture.OutputPath); err != nil {
		t.Fatal(err)
	}
	for _, relative := range fixture.CandidateSubjects {
		if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(relative))); err != nil {
			t.Errorf("candidate subject %q is not readable: %v", relative, err)
		}
	}
}

func v11ValidationShellBody() string {
	return "go test -mod=vendor -count=1 . ./acceptance -run \"^(TestProductRoadmapIsExhaustiveAndCausal|TestProductRoadmapV11ScopeAndExecutableContract|TestV11OwnsNoCapabilityIDs|TestV11AcceptanceCommandRunsIdentityConsumers|TestRebuildArchitecture|TestTraceabilityRebuildBugLessons|TestAcceptanceV11OIDCAD|TestV11CandidateSubjectsCoverCommittedDelta)$\"" +
		" && go test -mod=vendor -count=1 ./internal/identity ./internal/config ./internal/adapters/auth/localtoken ./internal/adapters/auth/oidc ./internal/bootstrap ./cmd/orquesta"
}

func v11AssertProviderShape(t *testing.T, repositoryRoot string) {
	t.Helper()
	production := v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal"), false)
	if count := strings.Count(production, "type IdentityProvider interface"); count != 1 {
		t.Errorf("V11_RED IdentityProvider interface count = %d, want exactly one", count)
	}
	for _, forbidden := range []string{"type PrincipalSource interface", "type IdentityMapper interface", "type LDAPProvider interface", "type ActiveDirectoryProvider interface"} {
		if strings.Contains(production, forbidden) {
			t.Errorf("V11_RED duplicate identity seam present: %s", forbidden)
		}
	}
	identitySource := v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "identity"), false)
	for _, required := range []string{"IdentityProvider", "Authenticate", "AuthenticationMethod"} {
		if !strings.Contains(identitySource, required) {
			t.Errorf("V11_RED neutral identity contract lacks %q", required)
		}
	}
	for _, adapter := range []string{"localtoken", "oidc"} {
		source := v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "adapters", "auth", adapter), true)
		for _, required := range []string{"IdentityProvider", "AuthenticationMethod"} {
			if !strings.Contains(source, required) {
				t.Errorf("V11_RED %s adapter does not implement shared %s", adapter, required)
			}
		}
	}
}

func v11AssertStableIdentity(t *testing.T, repositoryRoot string, fixture v11Fixture) {
	t.Helper()
	payload := strings.Join([]string{"oidc", fixture.Scenario.OIDCIssuer, fixture.Scenario.OIDCSubject}, "\x00")
	digest := sha256.Sum256([]byte(payload))
	wantDigest := hex.EncodeToString(digest[:])
	if fixture.Scenario.ExpectedPrincipalRef != "principal:sha256:"+wantDigest ||
		fixture.Scenario.ExpectedActorRef != "actor:sha256:"+wantDigest {
		t.Fatalf("fixture stable identity is not derived from method issuer subject: %+v", fixture.Scenario)
	}
	identitySource := v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "identity"), false)
	for _, required := range []string{"StablePrincipal", "sha256", "issuer", "subject"} {
		if !strings.Contains(strings.ToLower(identitySource), strings.ToLower(required)) {
			t.Errorf("V11_RED stable identity mapping lacks %q", required)
		}
	}
	oidcSource := v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "adapters", "auth", "oidc"), true)
	for _, forbidden := range []string{"RoleProject", "GrantMembership", "MembershipRole", "project_admins", "group_to_role"} {
		if strings.Contains(oidcSource, forbidden) {
			t.Errorf("V11_RED OIDC groups can mutate RBAC through %q", forbidden)
		}
	}
}

func v11AssertOIDCShape(t *testing.T, repositoryRoot string) {
	t.Helper()
	oidcSource := strings.ToLower(v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal", "adapters", "auth", "oidc"), true))
	for _, required := range []string{
		"issuer", "audience", "subject", "expiry", "notbefore", "issuedat", "clockskew",
		"jwks", "kid", "signature", "algorithm", "unknownkid", "refresh",
	} {
		if !strings.Contains(oidcSource, required) {
			t.Errorf("V11_RED OIDC verifier lacks %q contract", required)
		}
	}
	for _, forbidden := range []string{"clientsecret", "refreshtoken", "authorizationcode", "pkceverifier", "sessioncookie"} {
		if strings.Contains(oidcSource, forbidden) {
			t.Errorf("V11_RED resource server owns client/session concern %q", forbidden)
		}
	}
}

func v11AssertCanonicalConfig(t *testing.T, repositoryRoot string, fixture v11Fixture) {
	t.Helper()
	content := v11ReadFile(t, filepath.Join(repositoryRoot, "config", "registry.json"))
	var registry v11Registry
	if err := json.Unmarshal([]byte(content), &registry); err != nil {
		t.Fatalf("decode config registry: %v", err)
	}
	definitions := make(map[string]v11RegistryKey, len(registry.Keys))
	for _, definition := range registry.Keys {
		definitions[definition.Key] = definition
	}
	for _, key := range fixture.Scenario.CanonicalConfigKeys {
		definition, found := definitions[key]
		if !found {
			t.Errorf("V11_RED canonical config key missing: %s", key)
			continue
		}
		if definition.Sensitive {
			t.Errorf("V11_RED public OIDC verifier setting marked secret: %s", key)
		}
	}
	provider := definitions[fixture.Scenario.ProviderConfigKey]
	if provider.Type != "string" || provider.Default != "local_token" ||
		!reflect.DeepEqual(provider.AllowedValues, fixture.Scenario.Methods) {
		t.Errorf("V11_RED identity.provider must select exactly local_token or oidc: %+v", provider)
	}
	for key := range definitions {
		lower := strings.ToLower(key)
		for _, forbidden := range []string{"client_secret", "password", "cookie", "session", "ldap.bind"} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("V11_RED forbidden identity secret/session config %q", key)
			}
		}
	}
	for _, path := range []string{
		filepath.Join(repositoryRoot, "internal", "adapters", "state", "sqlite"),
		filepath.Join(repositoryRoot, "internal", "adapters", "config", "effectivefile"),
	} {
		if strings.Contains(v11ReadProductionGo(t, path, false), fixture.Scenario.TokenSentinel) {
			t.Errorf("V11_RED token sentinel reached durable product path %s", path)
		}
	}
}

func v11AssertExternalADBridge(t *testing.T, repositoryRoot string, fixture v11Fixture) {
	t.Helper()
	if fixture.Scenario.DexConnector != "ldap" || fixture.Scenario.DirectoryTransport != "ldaps" {
		t.Fatalf("classic AD bridge must be Dex LDAP over LDAPS: %+v", fixture.Scenario)
	}
	goMod := strings.ToLower(v11ReadFile(t, filepath.Join(repositoryRoot, "go.mod")))
	if strings.Contains(goMod, "go-ldap") {
		t.Error("V11_RED production module embeds LDAP client; classic AD must stay behind external Dex OIDC bridge")
	}
	production := strings.ToLower(v11ReadProductionGo(t, filepath.Join(repositoryRoot, "internal"), false))
	for _, imported := range []string{"github.com/go-ldap", "gopkg.in/ldap", "ldap.dial", "ldap.bind"} {
		if strings.Contains(production, imported) {
			t.Errorf("V11_RED production embeds directory operation %q", imported)
		}
	}
	for _, forbiddenPath := range []string{
		filepath.Join(repositoryRoot, "internal", "adapters", "auth", "ldap"),
		filepath.Join(repositoryRoot, "internal", "adapters", "auth", "activedirectory"),
	} {
		if _, err := os.Stat(forbiddenPath); err == nil {
			t.Errorf("V11_RED direct directory adapter exists at %s", forbiddenPath)
		} else if !os.IsNotExist(err) {
			t.Errorf("inspect forbidden directory adapter %s: %v", forbiddenPath, err)
		}
	}
}

func v11ReadProductionGo(t *testing.T, directory string, missingIsRed bool) string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		if missingIsRed && os.IsNotExist(err) {
			t.Errorf("V11_RED production package missing: %s", directory)
			return ""
		}
		t.Fatalf("read production tree %s: %v", directory, err)
	}
	var files []string
	for _, entry := range entries {
		path := filepath.Join(directory, entry.Name())
		if entry.IsDir() {
			files = append(files, path+string(filepath.Separator))
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			files = append(files, path)
		}
	}
	sort.Strings(files)
	var source strings.Builder
	for len(files) > 0 {
		path := files[0]
		files = files[1:]
		if strings.HasSuffix(path, string(filepath.Separator)) {
			nested := v11ReadProductionGo(t, strings.TrimSuffix(path, string(filepath.Separator)), false)
			source.WriteString(nested)
			continue
		}
		source.WriteString(v11ReadFile(t, path))
		source.WriteByte('\n')
	}
	return source.String()
}

func v11ReadFile(t *testing.T, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	return string(content)
}
