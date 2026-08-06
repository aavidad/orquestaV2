package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/adapters/credentials/filesource"
	"orquesta/internal/adapters/credentials/keysource"
	credentiallocal "orquesta/internal/adapters/credentials/local"
	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/i18n"
	"orquesta/internal/ports"
)

const (
	credentialProvisionOutputSchema = 1
	credentialProvisionUsageKey     = "cli.credentials.provision_codex_microvm.usage"
)

var (
	errCredentialProvisionClose = errors.New("credential_provision.close_failed")
	errCredentialProvisionPanic = errors.New("credential_provision.panicked")
)

type credentialProvisionOwnedStore interface {
	credentials.ProvisionStore
	Close() error
}

type credentialProvisionDependencies struct {
	loadSnapshot func(context.Context, string) (config.Snapshot, error)
	openStore    func(credentiallocal.Options) (credentialProvisionOwnedStore, error)
	newAuth      func(string, uint32, uint64) (credentials.MaterialSource, error)
	newKey       func() (credentials.Ed25519PrivateKeySource, error)
	provision    func(
		context.Context,
		credentials.ProvisionStore,
		credentials.MaterialSource,
		credentials.Ed25519PrivateKeySource,
		credentials.ProvisionCodexMicroVMCredentialsRequest,
	) (credentials.ProvisionCodexMicroVMCredentialsResult, error)
	effectiveUID func() int
}

func productionCredentialProvisionDependencies() credentialProvisionDependencies {
	return credentialProvisionDependencies{
		loadSnapshot: loadCredentialProvisionConfigSnapshot,
		openStore: func(options credentiallocal.Options) (credentialProvisionOwnedStore, error) {
			return credentiallocal.Open(options)
		},
		newAuth: func(path string, ownerUID uint32, maximum uint64) (credentials.MaterialSource, error) {
			return filesource.New(path, ownerUID, maximum)
		},
		newKey: func() (credentials.Ed25519PrivateKeySource, error) {
			return keysource.New(keysource.Options{})
		},
		provision:    credentials.ProvisionCodexMicroVMCredentials,
		effectiveUID: os.Geteuid,
	}
}

// RunProvisionCodexMicroVMCredentials composes the one-shot local maintenance
// command. The canonical snapshot remains the sole authority for identities,
// scopes, credential refs, key ID, placement and store limits.
func RunProvisionCodexMicroVMCredentials(
	ctx context.Context,
	arguments []string,
	catalog *i18n.Catalog,
	stdout io.Writer,
	stderr io.Writer,
) int {
	return runProvisionCodexMicroVMCredentials(
		ctx, arguments, catalog, stdout, stderr, productionCredentialProvisionDependencies(),
	)
}

func runProvisionCodexMicroVMCredentials(
	ctx context.Context,
	arguments []string,
	catalog *i18n.Catalog,
	stdout io.Writer,
	stderr io.Writer,
	dependencies credentialProvisionDependencies,
) int {
	flags := flag.NewFlagSet("orquesta credentials provision-codex-microvm", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "")
	authPath := flags.String("auth-json-path", "", "")
	requestRef := flags.String("request-ref", "", "")
	parseErr := flags.Parse(arguments)
	locale := i18n.DefaultLocale

	if catalog == nil {
		_, _ = io.WriteString(stderr, "code=i18n_catalog_unavailable\n")
		return 1
	}
	if errors.Is(parseErr, flag.ErrHelp) {
		if usage, err := catalog.Text(locale, credentialProvisionUsageKey); err == nil {
			_, _ = io.WriteString(stdout, usage+"\n")
			return 0
		}
		_, _ = io.WriteString(stderr, "code=i18n_catalog_unavailable\n")
		return 1
	}
	if parseErr != nil || flags.NArg() != 0 ||
		!validCredentialProvisionConfigPath(*configPath) ||
		!validCredentialProvisionAuthPath(*authPath) ||
		!validCredentialProvisionRequestBase(*requestRef) {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_arguments_invalid",
			"error.cli.credential_provision_arguments_invalid",
		)
		return 2
	}
	if ctx == nil || dependencies.loadSnapshot == nil || dependencies.openStore == nil ||
		dependencies.newAuth == nil || dependencies.newKey == nil || dependencies.provision == nil ||
		dependencies.effectiveUID == nil {
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
	plan, err := buildCredentialProvisionPlan(snapshot, *requestRef, uid)
	if err != nil {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_configuration_invalid",
			"error.cli.credential_provision_configuration_invalid",
		)
		return 2
	}
	authSource, err := dependencies.newAuth(*authPath, uint32(uid), plan.authMaximum)
	if err != nil {
		writeCredentialProvisionDiagnostic(
			stderr, catalog, locale,
			"cli.credential_provision_arguments_invalid",
			"error.cli.credential_provision_arguments_invalid",
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
	result, operationErr := provisionWithOwnedCredentialStore(
		ctx, store, authSource, keySource, plan.request, dependencies.provision,
	)
	if operationErr != nil {
		if hasCredentialProvisionProjection(result) {
			output := projectCredentialProvisionOutput(plan.placementRef, result, operationErr)
			if encodeErr := json.NewEncoder(stdout).Encode(output); encodeErr != nil {
				writeCLIDiagnostic(stderr, catalog, locale, cliDiagnostic{
					code: "cli.output_invalid", messageKey: "error.cli.output_invalid",
				})
				return 1
			}
		}
		writeCredentialProvisionOperationError(stderr, catalog, locale, operationErr)
		return 1
	}
	output := projectCredentialProvisionOutput(plan.placementRef, result, nil)
	if err := json.NewEncoder(stdout).Encode(output); err != nil {
		writeCLIDiagnostic(stderr, catalog, locale, cliDiagnostic{
			code: "cli.output_invalid", messageKey: "error.cli.output_invalid",
		})
		return 1
	}
	return 0
}

type credentialProvisionPlan struct {
	placementRef string
	authMaximum  uint64
	request      credentials.ProvisionCodexMicroVMCredentialsRequest
}

func buildCredentialProvisionPlan(
	snapshot config.Snapshot,
	requestBase string,
	uid int,
) (credentialProvisionPlan, error) {
	if snapshot.RuntimeProvider() != "codex" || snapshot.RuntimeIsolation() != "microvm" ||
		snapshot.IdentityProvider() != "local_token" || uid < 0 || uint64(uid) > math.MaxUint32 ||
		snapshot.RuntimeCodexAccountAuthMaxDocumentBytes() <= 0 ||
		uint64(snapshot.RuntimeCodexAccountAuthMaxDocumentBytes()) > filesource.MaximumMaterialBytes {
		return credentialProvisionPlan{}, errors.New("credential_provision.configuration_invalid")
	}
	placement, err := ports.NewAgentPlacementRef(snapshot.RuntimeMicroVMPlacementRef())
	if err != nil {
		return credentialProvisionPlan{}, errors.New("credential_provision.configuration_invalid")
	}
	actor := snapshot.IdentityLocalActor()
	project := credentials.ScopeRef(snapshot.ProjectDefault())
	request := credentials.ProvisionCodexMicroVMCredentialsRequest{
		ActorRef:     actor,
		OwnerRef:     credentials.OwnerRef(actor),
		SigningKeyID: snapshot.RuntimeMicroVMLaunchGrantKeyID(),
		Signing: credentials.ProvisionCredentialRequest{
			RequestRef: requestBase + ":signing",
			CredentialRef: credentials.CredentialRef(
				snapshot.RuntimeMicroVMLaunchGrantSigningCredentialRef(),
			),
			ScopeRefs:  []credentials.ScopeRef{project},
			PurposeRef: agentmicrovm.LaunchGrantSigningCredentialPurpose,
		},
		Auth: credentials.ProvisionCredentialRequest{
			RequestRef:    requestBase + ":auth",
			CredentialRef: credentials.CredentialRef(snapshot.RuntimeCodexCredentialRef()),
			ScopeRefs:     []credentials.ScopeRef{project},
			PurposeRef:    credentials.PurposeRef(codex.ProviderRef),
		},
	}
	if err := credentials.ValidateProvisionCodexMicroVMCredentialsRequest(request); err != nil {
		return credentialProvisionPlan{}, errors.New("credential_provision.configuration_invalid")
	}
	return credentialProvisionPlan{
		placementRef: placement.String(),
		authMaximum:  uint64(snapshot.RuntimeCodexAccountAuthMaxDocumentBytes()),
		request:      request,
	}, nil
}

func provisionWithOwnedCredentialStore(
	ctx context.Context,
	store credentialProvisionOwnedStore,
	authSource credentials.MaterialSource,
	keySource credentials.Ed25519PrivateKeySource,
	request credentials.ProvisionCodexMicroVMCredentialsRequest,
	provision func(
		context.Context,
		credentials.ProvisionStore,
		credentials.MaterialSource,
		credentials.Ed25519PrivateKeySource,
		credentials.ProvisionCodexMicroVMCredentialsRequest,
	) (credentials.ProvisionCodexMicroVMCredentialsResult, error),
) (result credentials.ProvisionCodexMicroVMCredentialsResult, err error) {
	defer func() {
		if recover() != nil {
			err = errCredentialProvisionPanic
		}
		if closeCredentialProvisionStore(store) != nil {
			err = errCredentialProvisionClose
		}
	}()
	return provision(ctx, store, authSource, keySource, request)
}

func closeCredentialProvisionStore(store credentialProvisionOwnedStore) (err error) {
	defer func() {
		if recover() != nil {
			err = errCredentialProvisionClose
		}
	}()
	if store == nil {
		return errCredentialProvisionClose
	}
	if err := store.Close(); err != nil {
		return errCredentialProvisionClose
	}
	return nil
}

func validCredentialProvisionConfigPath(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}

func validCredentialProvisionAuthPath(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00') &&
		filepath.IsAbs(value) && filepath.Clean(value) == value && filepath.Base(value) != string(filepath.Separator)
}

func validCredentialProvisionRequestBase(value string) bool {
	return strings.HasPrefix(value, "request:") && len(value) > len("request:") &&
		strings.TrimSpace(value) == value && !strings.ContainsRune(value, '\x00')
}

type credentialProvisionOutput struct {
	SchemaVersion          int                             `json:"schema_version"`
	Status                 string                          `json:"status"`
	PlacementRef           string                          `json:"placement_ref"`
	Signing                credentialProvisionStatusOutput `json:"signing"`
	Auth                   credentialProvisionStatusOutput `json:"auth"`
	SigningKeyID           string                          `json:"signing_key_id,omitempty"`
	SigningPublicKeyBase64 string                          `json:"signing_public_key_base64,omitempty"`
	Error                  *credentialProvisionErrorOutput `json:"error,omitempty"`
}

type credentialProvisionStatusOutput struct {
	State              string                             `json:"state"`
	RequestedAuthority credentialProvisionAuthorityOutput `json:"requested_authority"`
	Receipts           []credentialProvisionReceiptOutput `json:"receipts"`
}

type credentialProvisionAuthorityOutput struct {
	CredentialRef string   `json:"credential_ref"`
	OwnerRef      string   `json:"owner_ref"`
	ScopeRefs     []string `json:"scope_refs"`
	PurposeRef    string   `json:"purpose_ref"`
	Version       uint64   `json:"version"`
}

type credentialProvisionReceiptOutput struct {
	CredentialRef string `json:"credential_ref"`
	OwnerRef      string `json:"owner_ref"`
	ScopeRef      string `json:"scope_ref,omitempty"`
	PurposeRef    string `json:"purpose_ref"`
	Version       uint64 `json:"version"`
	RequestRef    string `json:"request_ref"`
	ActorRef      string `json:"actor_ref"`
	Operation     string `json:"operation"`
	OccurredAt    string `json:"occurred_at"`
}

type credentialProvisionErrorOutput struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func projectCredentialProvisionOutput(
	placementRef string,
	result credentials.ProvisionCodexMicroVMCredentialsResult,
	err error,
) credentialProvisionOutput {
	status := "complete"
	var projectedError *credentialProvisionErrorOutput
	if err != nil {
		status = "partial"
		projectedError = projectCredentialProvisionError(err)
	}
	return credentialProvisionOutput{
		SchemaVersion:          credentialProvisionOutputSchema,
		Status:                 status,
		PlacementRef:           placementRef,
		Signing:                projectCredentialProvisionStatus(result.Signing),
		Auth:                   projectCredentialProvisionStatus(result.Auth),
		SigningKeyID:           result.SigningKeyID,
		SigningPublicKeyBase64: result.SigningPublicKeyBase64,
		Error:                  projectedError,
	}
}

func projectCredentialProvisionStatus(
	status credentials.CredentialProvisionStatus,
) credentialProvisionStatusOutput {
	scopes := make([]string, len(status.RequestedAuthority.ScopeRefs))
	for index, scope := range status.RequestedAuthority.ScopeRefs {
		scopes[index] = scope.String()
	}
	receipts := make([]credentialProvisionReceiptOutput, len(status.Receipts))
	for index, receipt := range status.Receipts {
		receipts[index] = credentialProvisionReceiptOutput{
			CredentialRef: receipt.CredentialRef.String(), OwnerRef: receipt.OwnerRef.String(),
			ScopeRef: receipt.ScopeRef.String(), PurposeRef: receipt.PurposeRef.String(),
			Version: uint64(receipt.Version), RequestRef: receipt.RequestRef,
			ActorRef: receipt.ActorRef, Operation: receipt.Operation,
			OccurredAt: receipt.OccurredAt.UTC().Format(time.RFC3339Nano),
		}
	}
	return credentialProvisionStatusOutput{
		State: string(status.State),
		RequestedAuthority: credentialProvisionAuthorityOutput{
			CredentialRef: status.RequestedAuthority.CredentialRef.String(),
			OwnerRef:      status.RequestedAuthority.OwnerRef.String(), ScopeRefs: scopes,
			PurposeRef: status.RequestedAuthority.PurposeRef.String(),
			Version:    uint64(status.RequestedAuthority.Version),
		},
		Receipts: receipts,
	}
}

func projectCredentialProvisionError(err error) *credentialProvisionErrorOutput {
	projection := &credentialProvisionErrorOutput{Code: "cli.credential_provision_failed"}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		projection.Code = "cli.credential_provision_canceled"
	case errors.Is(err, errCredentialProvisionClose):
		projection.Code = "cli.credential_provision_close_failed"
	case errors.Is(err, errCredentialProvisionPanic):
		projection.Code = "cli.credential_provision_failed"
	default:
		var credentialError *credentials.Error
		if errors.As(err, &credentialError) {
			projection.Code = string(credentialError.Code)
			if safeCredentialProvisionErrorField(credentialError.Field) {
				projection.Field = credentialError.Field
			}
		}
	}
	return projection
}

func safeCredentialProvisionErrorField(value string) bool {
	if value == "" || len(value) > 80 {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
			character == '_' || character == ':' || character == '-' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func hasCredentialProvisionProjection(result credentials.ProvisionCodexMicroVMCredentialsResult) bool {
	return result.Signing.State != "" || result.Auth.State != "" ||
		result.Signing.RequestedAuthority.CredentialRef != "" ||
		result.Auth.RequestedAuthority.CredentialRef != "" ||
		len(result.Signing.Receipts) != 0 || len(result.Auth.Receipts) != 0 ||
		result.SigningKeyID != "" || result.SigningPublicKeyBase64 != ""
}

func writeCredentialProvisionOperationError(
	writer io.Writer,
	catalog *i18n.Catalog,
	locale string,
	err error,
) {
	projection := projectCredentialProvisionError(err)
	messageKey := "error.cli.credential_provision_failed"
	if projection.Code == "cli.credential_provision_canceled" {
		messageKey = "error.cli.credential_provision_canceled"
	}
	writeCredentialProvisionDiagnostic(writer, catalog, locale, projection.Code, messageKey)
}

func writeCredentialProvisionDiagnostic(
	writer io.Writer,
	catalog *i18n.Catalog,
	locale string,
	code string,
	messageKey string,
) {
	writeCLIDiagnostic(writer, catalog, locale, cliDiagnostic{code: code, messageKey: messageKey})
}
