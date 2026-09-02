package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"math"
	"os"
	"path/filepath"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/adapters/credentials/keysource"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/i18n"
)

const (
	continuationCredentialProvisionOutputSchema = 1
	continuationCredentialProvisionUsageKey     = "cli.credentials.provision_microvm_continuation_authority.usage"
)

var errContinuationCredentialPublicKeyDigestMismatch = errors.New(
	"continuation_credential_provision.public_key_digest_mismatch",
)

type continuationCredentialProvisionDependencies struct {
	loadSnapshot func(context.Context, string) (config.Snapshot, error)
	openStore    func(credentiallocal.Options) (credentialProvisionOwnedStore, error)
	newKey       func() (credentials.Ed25519PrivateKeySource, error)
	provision    func(
		context.Context,
		credentials.ProvisionStore,
		credentials.Ed25519PrivateKeySource,
		credentials.ProvisionEd25519SigningCredentialRequest,
	) (credentials.ProvisionEd25519SigningCredentialResult, error)
	effectiveUID func() int
}

func productionContinuationCredentialProvisionDependencies() continuationCredentialProvisionDependencies {
	return continuationCredentialProvisionDependencies{
		loadSnapshot: loadContinuationCredentialProvisionConfigSnapshot,
		openStore: func(options credentiallocal.Options) (credentialProvisionOwnedStore, error) {
			return credentiallocal.Open(options)
		},
		newKey: func() (credentials.Ed25519PrivateKeySource, error) {
			return keysource.New(keysource.Options{})
		},
		provision:    credentials.ProvisionEd25519SigningCredential,
		effectiveUID: os.Geteuid,
	}
}

// RunProvisionMicroVMContinuationAuthority composes a local, one-shot
// maintenance command. It can create or reconsult the configured signing
// credential, but it cannot rotate, revoke, or select trust metadata by flag.
func RunProvisionMicroVMContinuationAuthority(
	ctx context.Context,
	arguments []string,
	catalog *i18n.Catalog,
	stdout io.Writer,
	stderr io.Writer,
) int {
	return runProvisionMicroVMContinuationAuthority(
		ctx, arguments, catalog, stdout, stderr,
		productionContinuationCredentialProvisionDependencies(),
	)
}

func runProvisionMicroVMContinuationAuthority(
	ctx context.Context,
	arguments []string,
	catalog *i18n.Catalog,
	stdout io.Writer,
	stderr io.Writer,
	dependencies continuationCredentialProvisionDependencies,
) int {
	flags := flag.NewFlagSet("orquesta credentials provision-microvm-continuation-authority", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "")
	requestRef := flags.String("request-ref", "", "")
	parseErr := flags.Parse(arguments)
	locale := i18n.DefaultLocale

	if catalog == nil {
		_, _ = io.WriteString(stderr, "code=i18n_catalog_unavailable\n")
		return 1
	}
	if errors.Is(parseErr, flag.ErrHelp) {
		if usage, err := catalog.Text(locale, continuationCredentialProvisionUsageKey); err == nil {
			_, _ = io.WriteString(stdout, usage+"\n")
			return 0
		}
		_, _ = io.WriteString(stderr, "code=i18n_catalog_unavailable\n")
		return 1
	}
	if parseErr != nil || flags.NArg() != 0 ||
		!validContinuationCredentialProvisionConfigPath(*configPath) ||
		!validCredentialProvisionRequestBase(*requestRef) {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_arguments_invalid",
			"error.cli.credential_provision_arguments_invalid",
		)
		return 2
	}
	if ctx == nil || dependencies.loadSnapshot == nil || dependencies.openStore == nil ||
		dependencies.newKey == nil || dependencies.provision == nil || dependencies.effectiveUID == nil {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_configuration_invalid",
			"error.cli.credential_provision_configuration_invalid",
		)
		return 2
	}
	if err := ctx.Err(); err != nil {
		writeCredentialProvisionOperationError(stderr, catalog, locale, err)
		return 1
	}

	snapshot, err := dependencies.loadSnapshot(ctx, *configPath)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeCredentialProvisionOperationError(stderr, catalog, locale, err)
			return 1
		}
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_configuration_invalid",
			"error.cli.credential_provision_configuration_invalid",
		)
		return 2
	}
	locale = snapshot.APILocale()
	if _, err := catalog.Resolve(locale); err != nil {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, i18n.DefaultLocale,
			"cli.credential_provision_configuration_invalid",
			"error.cli.credential_provision_configuration_invalid",
		)
		return 2
	}

	uid := dependencies.effectiveUID()
	plan, err := buildContinuationCredentialProvisionPlan(snapshot, *requestRef, uid)
	if err != nil {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_configuration_invalid",
			"error.cli.credential_provision_configuration_invalid",
		)
		return 2
	}
	keySource, err := dependencies.newKey()
	if err != nil {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_configuration_invalid",
			"error.cli.credential_provision_configuration_invalid",
		)
		return 2
	}
	if err := ctx.Err(); err != nil {
		writeCredentialProvisionOperationError(stderr, catalog, locale, err)
		return 1
	}

	store, err := dependencies.openStore(credentiallocal.Options{
		Path: snapshot.CredentialsLocalPath(), OwnerUID: uid,
		MaxStoreBytes: snapshot.CredentialsLocalMaxDocumentBytes(),
	})
	if err != nil {
		writeCredentialProvisionOperationError(stderr, catalog, locale, err)
		return 1
	}
	result, operationErr := operateWithOwnedCredentialStore(store, func() (
		credentials.ProvisionEd25519SigningCredentialResult, error,
	) {
		return dependencies.provision(ctx, store, keySource, plan.request)
	})
	output, projectionErr := projectContinuationCredentialProvisionOutput(plan, result, operationErr)
	if projectionErr != nil {
		operationErr = projectionErr
	}
	if operationErr != nil {
		output.Status = "failed"
		output.Error = projectContinuationCredentialProvisionError(operationErr)
	}
	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		writeCLIDiagnostic(stderr, catalog, locale, cliDiagnostic{
			code: "cli.output_invalid", messageKey: "error.cli.output_invalid",
		})
		return 1
	}
	if operationErr != nil {
		writeContinuationCredentialProvisionOperationError(
			stderr, catalog, locale, operationErr,
		)
		return 1
	}
	return 0
}

type continuationCredentialProvisionPlan struct {
	keyID                   string
	keyEpoch                uint64
	trustRevision           uint64
	expectedPublicKeySHA256 string
	request                 credentials.ProvisionEd25519SigningCredentialRequest
}

func buildContinuationCredentialProvisionPlan(
	snapshot config.Snapshot,
	requestBase string,
	uid int,
) (continuationCredentialProvisionPlan, error) {
	if snapshot.RuntimeProvider() != "codex" || snapshot.RuntimeIsolation() != "microvm" ||
		snapshot.IdentityProvider() != "local_token" || uid < 0 || uint64(uid) > math.MaxUint32 {
		return continuationCredentialProvisionPlan{}, errors.New(
			"continuation_credential_provision.configuration_invalid",
		)
	}
	actor := snapshot.IdentityLocalActor()
	request := credentials.ProvisionEd25519SigningCredentialRequest{
		ActorRef: actor,
		OwnerRef: credentials.OwnerRef(actor),
		Signing: credentials.ProvisionCredentialRequest{
			RequestRef: requestBase + ":continuation-signing",
			CredentialRef: credentials.CredentialRef(
				snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthoritySigningCredentialRef(),
			),
			ScopeRefs:  []credentials.ScopeRef{credentials.ScopeRef(snapshot.ProjectDefault())},
			PurposeRef: agentmicrovm.ExpiredLaunchContinuationSigningCredentialPurposeV41,
		},
	}
	if err := credentials.ValidateProvisionEd25519SigningCredentialRequest(request); err != nil {
		return continuationCredentialProvisionPlan{}, errors.New(
			"continuation_credential_provision.configuration_invalid",
		)
	}
	return continuationCredentialProvisionPlan{
		keyID: snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityKeyID(),
		keyEpoch: uint64(
			snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityKeyEpoch(),
		),
		trustRevision: uint64(
			snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityTrustRevision(),
		),
		expectedPublicKeySHA256: snapshot.RuntimeMicroVMExpiredLaunchContinuationAuthorityPublicKeySHA256(),
		request:                 request,
	}, nil
}

type continuationCredentialProvisionOutput struct {
	SchemaVersion                    int                             `json:"schema_version"`
	Status                           string                          `json:"status"`
	CredentialState                  string                          `json:"credential_state,omitempty"`
	CredentialVersion                uint64                          `json:"credential_version,omitempty"`
	KeyID                            string                          `json:"key_id,omitempty"`
	KeyEpoch                         uint64                          `json:"key_epoch,omitempty"`
	TrustRevision                    uint64                          `json:"trust_revision,omitempty"`
	PublicKeyBase64                  string                          `json:"public_key_base64,omitempty"`
	PublicKeySHA256                  string                          `json:"public_key_sha256,omitempty"`
	ConfiguredPublicKeySHA256Matches *bool                           `json:"configured_public_key_sha256_matches,omitempty"`
	Error                            *credentialProvisionErrorOutput `json:"error,omitempty"`
}

func projectContinuationCredentialProvisionOutput(
	plan continuationCredentialProvisionPlan,
	result credentials.ProvisionEd25519SigningCredentialResult,
	operationErr error,
) (continuationCredentialProvisionOutput, error) {
	output := continuationCredentialProvisionOutput{
		SchemaVersion:     continuationCredentialProvisionOutputSchema,
		Status:            "complete",
		CredentialState:   string(result.Signing.State),
		CredentialVersion: uint64(result.Signing.RequestedAuthority.Version),
		KeyID:             plan.keyID, KeyEpoch: plan.keyEpoch, TrustRevision: plan.trustRevision,
	}
	if operationErr != nil {
		return output, operationErr
	}
	if result.SigningPublicKeyBase64 == "" {
		return output, credentials.NewError(credentials.ErrorInvalidRequest, "public_key")
	}
	publicKey, err := base64.StdEncoding.DecodeString(result.SigningPublicKeyBase64)
	if err != nil || len(publicKey) != ed25519.PublicKeySize ||
		base64.StdEncoding.EncodeToString(publicKey) != result.SigningPublicKeyBase64 {
		clear(publicKey)
		return output, credentials.NewError(credentials.ErrorInvalidRequest, "public_key")
	}
	digest := sha256.Sum256(publicKey)
	clear(publicKey)
	output.PublicKeyBase64 = result.SigningPublicKeyBase64
	output.PublicKeySHA256 = hex.EncodeToString(digest[:])
	if plan.expectedPublicKeySHA256 != "" {
		matches := plan.expectedPublicKeySHA256 == output.PublicKeySHA256
		output.ConfiguredPublicKeySHA256Matches = &matches
		if !matches {
			return output, errContinuationCredentialPublicKeyDigestMismatch
		}
	}
	return output, nil
}

func projectContinuationCredentialProvisionError(err error) *credentialProvisionErrorOutput {
	if errors.Is(err, errContinuationCredentialPublicKeyDigestMismatch) {
		return &credentialProvisionErrorOutput{
			Code: "cli.continuation_credential_public_key_digest_mismatch",
		}
	}
	return projectCredentialProvisionError(err)
}

func writeContinuationCredentialProvisionOperationError(
	writer io.Writer,
	catalog *i18n.Catalog,
	locale string,
	err error,
) {
	if errors.Is(err, errContinuationCredentialPublicKeyDigestMismatch) {
		writeCredentialProvisionDiagnostic(
			writer, catalog, locale,
			"cli.continuation_credential_public_key_digest_mismatch",
			"error.cli.continuation_credential_public_key_digest_mismatch",
		)
		return
	}
	writeCredentialProvisionOperationError(writer, catalog, locale, err)
}

func validContinuationCredentialProvisionConfigPath(value string) bool {
	return validCredentialProvisionConfigPath(value) && filepath.IsAbs(value) &&
		filepath.Clean(value) == value && filepath.Base(value) != string(filepath.Separator)
}
