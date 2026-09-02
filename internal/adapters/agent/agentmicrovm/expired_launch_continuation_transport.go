package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/application"
)

const (
	operationPrepareExpiredLaunchContinuation  = "preparar_continuacion_lanzamiento_caducado"
	operationContinueExpiredLaunchContinuation = "continuar_lanzamiento_caducado"
)

var (
	expiredContinuationRequestKeyDomain      = []byte("agentmicrovm.clave-solicitud-continuacion-lanzamiento-caducado.v1\x00")
	expiredContinuationOriginalRequestDomain = []byte("agentmicrovm.solicitud-original-continuacion-lanzamiento-caducado.v1\x00")
)

var (
	ErrExpiredLaunchContinuationInvalid     = errors.New("agentmicrovm.expired_launch_continuation_invalid")
	ErrExpiredLaunchContinuationUnsupported = errors.New("agentmicrovm.expired_launch_continuation_unsupported")
	ErrExpiredLaunchContinuationDivergent   = errors.New("agentmicrovm.expired_launch_continuation_divergent")
)

// ExpiredLaunchContinuationClientV1 deliberately excludes Lanzar and the old
// reconciliation endpoint. No implementation of this port can fall back to an
// ordinary launch when continuation is absent or rejected.
type ExpiredLaunchContinuationClientV1 interface {
	Capacidades(context.Context) (microvm.RespuestaCapacidades, error)
	PrepararContinuacionLanzamientoCaducado(
		context.Context,
		string,
		microvm.SolicitudLanzamiento,
	) (microvm.RespuestaPreparacionContinuacionLanzamientoCaducadoV1, error)
	ContinuarLanzamientoCaducado(
		context.Context,
		string,
		microvm.SolicitudContinuacionLanzamientoCaducadoV1,
	) (microvm.RespuestaEjecucion, error)
}

var _ ExpiredLaunchContinuationClientV1 = (*microvm.Cliente)(nil)

// ExpiredLaunchContinuationCausalBindingV41 is an alias, not a second contract.
// Application owns the exact EffectAttempt/attempt binding.
type ExpiredLaunchContinuationCausalBindingV41 = application.ExpiredAgentLaunchContinuationCausalBindingV41

type ExpiredLaunchContinuationSubjectV41 struct {
	Binding          ExpiredLaunchContinuationCausalBindingV41
	IdempotencyKey   string
	Original         microvm.SolicitudLanzamiento
	ProfileSHA256    string
	PlanSHA256       string
	ConcessionSHA256 string
	ExpectedManifest microvm.ExpiredLaunchContinuationManifestV1
}

type PreparedExpiredLaunchContinuationV41 struct {
	Subject        ExpiredLaunchContinuationSubjectV41
	Manifest       microvm.ExpiredLaunchContinuationManifestV1
	ManifestSHA256 string
	ManifestBytes  []byte
}

type ExpiredLaunchContinuationIssuanceV41 struct {
	AuthorityRef  string
	IssuedUnixMS  uint64
	ExpiresUnixMS uint64
	KeyID         string
	KeyEpoch      uint64
	TrustRevision uint64
}

type IssuedExpiredLaunchContinuationV41 struct {
	Prepared        PreparedExpiredLaunchContinuationV41
	Authority       microvm.ExpiredLaunchContinuationAuthorityV1
	AuthoritySHA256 string
	AuthorityBytes  []byte
	PublicKey       []byte
}

type ExpiredLaunchContinuationTransportV41 struct {
	client ExpiredLaunchContinuationClientV1
}

func NewExpiredLaunchContinuationTransportV41(
	client ExpiredLaunchContinuationClientV1,
) (*ExpiredLaunchContinuationTransportV41, error) {
	if nilInterface(client) {
		return nil, ErrExpiredLaunchContinuationInvalid
	}
	return &ExpiredLaunchContinuationTransportV41{client: client}, nil
}

func (transport *ExpiredLaunchContinuationTransportV41) Prepare(
	ctx context.Context,
	subject ExpiredLaunchContinuationSubjectV41,
) (PreparedExpiredLaunchContinuationV41, error) {
	prepared, err := transport.PrepareObserved(ctx, subject)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	if !reflect.DeepEqual(prepared.Manifest, subject.ExpectedManifest) {
		return PreparedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationDivergent
	}
	return prepared, nil
}

// PrepareObserved performs the sibling's observational preparation and admits
// the returned immutable manifest. Rust's preparation endpoint validates and
// derives durable state only: it does not persist authority or launch a VM.
func (transport *ExpiredLaunchContinuationTransportV41) PrepareObserved(
	ctx context.Context,
	subject ExpiredLaunchContinuationSubjectV41,
) (PreparedExpiredLaunchContinuationV41, error) {
	if transport == nil || nilInterface(transport.client) || ctx == nil || validateExpiredContinuationSubjectBaseV41(subject) != nil {
		return PreparedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	if err := ctx.Err(); err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	if err := requireExpiredContinuationCapabilities(ctx, transport.client); err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	response, err := transport.client.PrepararContinuacionLanzamientoCaducado(
		ctx, subject.IdempotencyKey, cloneOriginalContinuationRequest(subject.Original),
	)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	manifestSHA256, digestErr := ExpiredLaunchContinuationManifestSHA256V1(response.Manifiesto)
	if digestErr != nil || response.ManifiestoSHA256 != manifestSHA256 ||
		validateObservedExpiredContinuationManifestV41(subject, response.Manifiesto) != nil {
		return PreparedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationDivergent
	}
	manifestBytes, err := json.Marshal(response.Manifiesto)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	subject.ExpectedManifest = response.Manifiesto
	return PreparedExpiredLaunchContinuationV41{
		Subject: cloneExpiredContinuationSubjectV41(subject), Manifest: response.Manifiesto,
		ManifestSHA256: manifestSHA256, ManifestBytes: manifestBytes,
	}, nil
}

func validateObservedExpiredContinuationManifestV41(
	subject ExpiredLaunchContinuationSubjectV41,
	manifest microvm.ExpiredLaunchContinuationManifestV1,
) error {
	if manifest.Fence != subject.Binding.ActionFence || manifest.Generation != subject.Binding.PlanGeneration ||
		manifest.RequestKeySHA256 != expiredContinuationRequestKeySHA256(subject.IdempotencyKey) ||
		manifest.OriginalRequestSHA256 != expiredContinuationOriginalRequestSHA256(subject.IdempotencyKey, subject.Original) {
		return ErrExpiredLaunchContinuationDivergent
	}
	return nil
}

// Issue signs only the exact manifest admitted by Prepare. The private key is
// borrowed for this call and is never retained in the returned artifact.
func (transport *ExpiredLaunchContinuationTransportV41) Issue(
	prepared PreparedExpiredLaunchContinuationV41,
	issuance ExpiredLaunchContinuationIssuanceV41,
	privateKey ed25519.PrivateKey,
) (IssuedExpiredLaunchContinuationV41, error) {
	if transport == nil || validatePreparedExpiredContinuationV41(prepared) != nil ||
		len(privateKey) != ed25519.PrivateKeySize {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	content := microvm.ExpiredLaunchContinuationAuthorityContentV1{
		Schema:       ExpiredLaunchContinuationAuthoritySchemaV1,
		Audience:     ExpiredLaunchContinuationAudienceV1,
		Purpose:      ExpiredLaunchContinuationPurposeV1,
		AuthorityRef: issuance.AuthorityRef, ManifestSHA256: prepared.ManifestSHA256,
		IssuedUnixMS: issuance.IssuedUnixMS, ExpiresUnixMS: issuance.ExpiresUnixMS,
		KeyID: issuance.KeyID, KeyEpoch: issuance.KeyEpoch,
		TrustRevision: issuance.TrustRevision, Algorithm: ExpiredLaunchContinuationAlgorithmV1,
	}
	authority, err := SignExpiredLaunchContinuationAuthorityV1(privateKey, content, prepared.Manifest)
	if err != nil {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	authoritySHA256, err := ExpiredLaunchContinuationAuthoritySHA256V1(authority)
	if err != nil {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	authorityBytes, err := json.Marshal(authority)
	if err != nil {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return IssuedExpiredLaunchContinuationV41{
		Prepared: clonePreparedExpiredContinuationV41(prepared), Authority: authority,
		AuthoritySHA256: authoritySHA256, AuthorityBytes: authorityBytes,
		PublicKey: append([]byte(nil), publicKey...),
	}, nil
}

func (transport *ExpiredLaunchContinuationTransportV41) Continue(
	ctx context.Context,
	issued IssuedExpiredLaunchContinuationV41,
) (microvm.RespuestaEjecucion, error) {
	if transport == nil || nilInterface(transport.client) || ctx == nil || validateIssuedExpiredContinuationV41(issued) != nil {
		return microvm.RespuestaEjecucion{}, ErrExpiredLaunchContinuationInvalid
	}
	if err := ctx.Err(); err != nil {
		return microvm.RespuestaEjecucion{}, err
	}
	if err := requireExpiredContinuationCapabilities(ctx, transport.client); err != nil {
		return microvm.RespuestaEjecucion{}, err
	}
	response, err := transport.client.ContinuarLanzamientoCaducado(
		ctx,
		issued.Prepared.Subject.IdempotencyKey,
		microvm.SolicitudContinuacionLanzamientoCaducadoV1{
			SolicitudOriginal: cloneOriginalContinuationRequest(issued.Prepared.Subject.Original),
			Autoridad:         issued.Authority,
		},
	)
	if err != nil {
		return microvm.RespuestaEjecucion{}, err
	}
	manifest := issued.Prepared.Manifest
	if response.Referencia != manifest.ExecutionRef || response.Cerca != manifest.Fence ||
		response.Estado != "disponible" || response.Revision == 0 || response.VCPU == 0 ||
		response.MemoriaMiB == 0 || response.Identidad == nil || response.Identidad.PID == 0 ||
		response.Identidad.InicioTicks == 0 || response.ProcesoVivo == nil || !*response.ProcesoVivo ||
		response.EstadoMotor == nil || *response.EstadoMotor != "Running" {
		return microvm.RespuestaEjecucion{}, ErrExpiredLaunchContinuationDivergent
	}
	return response, nil
}

func BuildExpiredAgentLaunchContinuationRecordV41(
	subjectRef string,
	issued IssuedExpiredLaunchContinuationV41,
	preparedAt time.Time,
	admittedUnixMS uint64,
) (application.ExpiredAgentLaunchContinuationRecordV41, error) {
	if subjectRef == "" || strings.ContainsAny(subjectRef, "\x00\r\n") || preparedAt.IsZero() ||
		validateIssuedExpiredContinuationV41(issued) != nil ||
		admittedUnixMS < issued.Authority.Content.IssuedUnixMS ||
		admittedUnixMS >= issued.Authority.Content.ExpiresUnixMS {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, ErrExpiredLaunchContinuationInvalid
	}
	signature, err := base64.StdEncoding.DecodeString(issued.Authority.SignatureBase64)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, ErrExpiredLaunchContinuationInvalid
	}
	subject, manifest, content := issued.Prepared.Subject, issued.Prepared.Manifest, issued.Authority.Content
	binding := subject.Binding
	return application.ExpiredAgentLaunchContinuationRecordV41{
		SubjectRef: subjectRef, ReconciliationAuthorityRef: binding.ReconciliationAuthorityRef,
		ReconciliationAttemptRef: binding.ReconciliationAttemptRef,
		ProjectRef:               binding.ProjectRef, GoalRef: binding.GoalRef, WorkItemRef: binding.WorkItemRef,
		ExecutionRef: binding.ExecutionRef, ActionRef: binding.ActionRef,
		EffectIntentRef: binding.EffectIntentRef, EffectIntentDigest: binding.EffectIntentDigest,
		EffectAttemptRef: binding.EffectAttemptRef, PlanGeneration: binding.PlanGeneration,
		WorkItemGeneration: binding.WorkItemGeneration, ActionFence: binding.ActionFence,
		RequestKeySHA256: manifest.RequestKeySHA256, OriginalRequestSHA256: manifest.OriginalRequestSHA256,
		AMVLaunchRef: manifest.LaunchRef, AMVExecutionRef: manifest.ExecutionRef,
		AMVRunRef: manifest.RunRef, AMVFence: manifest.Fence, AMVGeneration: manifest.Generation,
		AMVCID: manifest.CID, AMVIdentitySHA256: manifest.IdentitySHA256,
		SourceDigest:            manifest.SourceDigest,
		ProfileDescriptorBytes:  append([]byte(nil), subject.Original.Perfil...),
		ProfileDescriptorSHA256: subject.ProfileSHA256,
		PlanBytes:               append([]byte(nil), subject.Original.Plan...), PlanSHA256: subject.PlanSHA256,
		ConcessionBytes:     append([]byte(nil), subject.Original.Concesion...),
		ConcessionSHA256:    subject.ConcessionSHA256,
		ManifestBytes:       append([]byte(nil), issued.Prepared.ManifestBytes...),
		ManifestBytesSHA256: bytesSHA256(issued.Prepared.ManifestBytes),
		ManifestSHA256:      issued.Prepared.ManifestSHA256, PreparedAt: preparedAt.UTC().Round(0),
		AuthorityRef: content.AuthorityRef, AuthoritySHA256: issued.AuthoritySHA256,
		AuthorityBytes:       append([]byte(nil), issued.AuthorityBytes...),
		AuthorityBytesSHA256: bytesSHA256(issued.AuthorityBytes), KeyID: content.KeyID,
		KeyEpoch: content.KeyEpoch, TrustRevision: content.TrustRevision,
		PublicKey: append([]byte(nil), issued.PublicKey...), Signature: signature,
		IssuedUnixMS: content.IssuedUnixMS, ExpiresUnixMS: content.ExpiresUnixMS,
		AdmittedUnixMS: admittedUnixMS,
	}, nil
}

func issuedExpiredAgentLaunchContinuationFromRecordV41(
	record application.ExpiredAgentLaunchContinuationRecordV41,
	idempotencyKey string,
) (IssuedExpiredLaunchContinuationV41, error) {
	manifest, err := DecodeExpiredLaunchContinuationManifestV1(record.ManifestBytes)
	if err != nil {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	authority, err := DecodeExpiredLaunchContinuationAuthorityV1(record.AuthorityBytes)
	if err != nil {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	original := microvm.SolicitudLanzamiento{
		Perfil:    append(json.RawMessage(nil), record.ProfileDescriptorBytes...),
		Plan:      append(json.RawMessage(nil), record.PlanBytes...),
		Concesion: append(json.RawMessage(nil), record.ConcessionBytes...),
	}
	subject := ExpiredLaunchContinuationSubjectV41{
		Binding: ExpiredLaunchContinuationCausalBindingV41{
			ReconciliationAuthorityRef: record.ReconciliationAuthorityRef,
			ReconciliationAttemptRef:   record.ReconciliationAttemptRef,
			ProjectRef:                 record.ProjectRef, GoalRef: record.GoalRef, WorkItemRef: record.WorkItemRef,
			ExecutionRef: record.ExecutionRef, ActionRef: record.ActionRef,
			EffectIntentRef: record.EffectIntentRef, EffectIntentDigest: record.EffectIntentDigest,
			EffectAttemptRef: record.EffectAttemptRef, PlanGeneration: record.PlanGeneration,
			WorkItemGeneration: record.WorkItemGeneration, ActionFence: record.ActionFence,
		},
		IdempotencyKey: idempotencyKey, Original: original,
		ProfileSHA256: record.ProfileDescriptorSHA256, PlanSHA256: record.PlanSHA256,
		ConcessionSHA256: record.ConcessionSHA256, ExpectedManifest: manifest,
	}
	prepared := PreparedExpiredLaunchContinuationV41{
		Subject: subject, Manifest: manifest, ManifestSHA256: record.ManifestSHA256,
		ManifestBytes: append([]byte(nil), record.ManifestBytes...),
	}
	issued := IssuedExpiredLaunchContinuationV41{
		Prepared: prepared, Authority: authority, AuthoritySHA256: record.AuthoritySHA256,
		AuthorityBytes: append([]byte(nil), record.AuthorityBytes...),
		PublicKey:      append([]byte(nil), record.PublicKey...),
	}
	signature, signatureErr := base64.StdEncoding.DecodeString(authority.SignatureBase64)
	if record.RequestKeySHA256 != manifest.RequestKeySHA256 ||
		record.OriginalRequestSHA256 != manifest.OriginalRequestSHA256 ||
		expiredContinuationRequestKeySHA256(idempotencyKey) != manifest.RequestKeySHA256 ||
		expiredContinuationOriginalRequestSHA256(idempotencyKey, original) != manifest.OriginalRequestSHA256 ||
		record.AMVLaunchRef != manifest.LaunchRef || record.AMVExecutionRef != manifest.ExecutionRef ||
		record.AMVRunRef != manifest.RunRef || record.AMVFence != manifest.Fence ||
		record.AMVGeneration != manifest.Generation || record.AMVCID != manifest.CID ||
		record.AMVIdentitySHA256 != manifest.IdentitySHA256 || record.SourceDigest != manifest.SourceDigest ||
		record.AuthorityRef != authority.Content.AuthorityRef ||
		record.KeyID != authority.Content.KeyID || record.KeyEpoch != authority.Content.KeyEpoch ||
		record.TrustRevision != authority.Content.TrustRevision ||
		record.IssuedUnixMS != authority.Content.IssuedUnixMS ||
		record.ExpiresUnixMS != authority.Content.ExpiresUnixMS ||
		record.AdmittedUnixMS < record.IssuedUnixMS || record.AdmittedUnixMS >= record.ExpiresUnixMS ||
		signatureErr != nil || !bytes.Equal(record.Signature, signature) ||
		bytesSHA256(record.ManifestBytes) != record.ManifestBytesSHA256 ||
		bytesSHA256(record.AuthorityBytes) != record.AuthorityBytesSHA256 ||
		validateIssuedExpiredContinuationV41(issued) != nil {
		return IssuedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationInvalid
	}
	return issued, nil
}

func requireExpiredContinuationCapabilities(
	ctx context.Context,
	client ExpiredLaunchContinuationClientV1,
) error {
	capabilities, err := client.Capacidades(ctx)
	if err != nil {
		return err
	}
	return ValidateExpiredLaunchContinuationCapabilitiesV41(capabilities)
}

func ValidateExpiredLaunchContinuationCapabilitiesV41(
	capabilities microvm.RespuestaCapacidades,
) error {
	if capabilities.Protocolo != microvm.ProtocoloLocal ||
		!containsOperationOnce(capabilities.Operaciones, operationPrepareExpiredLaunchContinuation) ||
		!containsOperationOnce(capabilities.Operaciones, operationContinueExpiredLaunchContinuation) {
		return ErrExpiredLaunchContinuationUnsupported
	}
	return nil
}

func containsOperationOnce(values []string, expected string) bool {
	matches := 0
	for _, value := range values {
		if value == expected {
			matches++
		}
	}
	return matches == 1
}

func validateExpiredContinuationSubjectV41(subject ExpiredLaunchContinuationSubjectV41) error {
	if validateExpiredContinuationSubjectBaseV41(subject) != nil ||
		subject.Binding.ActionFence != subject.ExpectedManifest.Fence ||
		subject.Binding.PlanGeneration != subject.ExpectedManifest.Generation ||
		expiredContinuationRequestKeySHA256(subject.IdempotencyKey) != subject.ExpectedManifest.RequestKeySHA256 ||
		expiredContinuationOriginalRequestSHA256(subject.IdempotencyKey, subject.Original) != subject.ExpectedManifest.OriginalRequestSHA256 {
		return ErrExpiredLaunchContinuationInvalid
	}
	_, err := ExpiredLaunchContinuationManifestSHA256V1(subject.ExpectedManifest)
	return err
}

func validateExpiredContinuationSubjectBaseV41(subject ExpiredLaunchContinuationSubjectV41) error {
	binding := subject.Binding
	for _, value := range []string{
		binding.ReconciliationAuthorityRef, binding.ReconciliationAttemptRef,
		binding.ProjectRef, binding.GoalRef, binding.WorkItemRef, binding.ExecutionRef,
		binding.ActionRef, binding.EffectIntentRef, binding.EffectAttemptRef,
	} {
		if value == "" || len(value) > 256 || strings.ContainsAny(value, "\x00\r\n") {
			return ErrExpiredLaunchContinuationInvalid
		}
	}
	if !validLowerSHA256(binding.EffectIntentDigest) || binding.PlanGeneration == 0 ||
		binding.WorkItemGeneration == 0 || binding.ActionFence == 0 ||
		subject.IdempotencyKey == "" || len(subject.IdempotencyKey) > 128 ||
		strings.ContainsAny(subject.IdempotencyKey, "\x00\r\n") ||
		bytesSHA256(subject.Original.Perfil) != subject.ProfileSHA256 ||
		bytesSHA256(subject.Original.Plan) != subject.PlanSHA256 ||
		bytesSHA256(subject.Original.Concesion) != subject.ConcessionSHA256 {
		return ErrExpiredLaunchContinuationInvalid
	}
	return nil
}

func validatePreparedExpiredContinuationV41(prepared PreparedExpiredLaunchContinuationV41) error {
	if validateExpiredContinuationSubjectV41(prepared.Subject) != nil ||
		!reflect.DeepEqual(prepared.Manifest, prepared.Subject.ExpectedManifest) {
		return ErrExpiredLaunchContinuationInvalid
	}
	digest, err := ExpiredLaunchContinuationManifestSHA256V1(prepared.Manifest)
	if err != nil || digest != prepared.ManifestSHA256 || bytesSHA256(prepared.ManifestBytes) == "" {
		return ErrExpiredLaunchContinuationInvalid
	}
	var decoded microvm.ExpiredLaunchContinuationManifestV1
	if err := json.Unmarshal(prepared.ManifestBytes, &decoded); err != nil || !reflect.DeepEqual(decoded, prepared.Manifest) {
		return ErrExpiredLaunchContinuationInvalid
	}
	return nil
}

func validateIssuedExpiredContinuationV41(issued IssuedExpiredLaunchContinuationV41) error {
	if validatePreparedExpiredContinuationV41(issued.Prepared) != nil ||
		len(issued.PublicKey) != ed25519.PublicKeySize ||
		VerifyExpiredLaunchContinuationAuthorityV1(ed25519.PublicKey(issued.PublicKey), issued.Authority) != nil ||
		issued.Authority.Content.ManifestSHA256 != issued.Prepared.ManifestSHA256 ||
		!reflect.DeepEqual(issued.Authority.Manifest, issued.Prepared.Manifest) {
		return ErrExpiredLaunchContinuationInvalid
	}
	digest, err := ExpiredLaunchContinuationAuthoritySHA256V1(issued.Authority)
	if err != nil || digest != issued.AuthoritySHA256 || bytesSHA256(issued.AuthorityBytes) == "" {
		return ErrExpiredLaunchContinuationInvalid
	}
	var decoded microvm.ExpiredLaunchContinuationAuthorityV1
	if err := json.Unmarshal(issued.AuthorityBytes, &decoded); err != nil || !reflect.DeepEqual(decoded, issued.Authority) {
		return ErrExpiredLaunchContinuationInvalid
	}
	return nil
}

func cloneOriginalContinuationRequest(value microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
	return microvm.SolicitudLanzamiento{
		Perfil: append(json.RawMessage(nil), value.Perfil...), Plan: append(json.RawMessage(nil), value.Plan...),
		Concesion: append(json.RawMessage(nil), value.Concesion...),
	}
}

func cloneExpiredContinuationSubjectV41(value ExpiredLaunchContinuationSubjectV41) ExpiredLaunchContinuationSubjectV41 {
	value.Original = cloneOriginalContinuationRequest(value.Original)
	return value
}

func clonePreparedExpiredContinuationV41(value PreparedExpiredLaunchContinuationV41) PreparedExpiredLaunchContinuationV41 {
	value.Subject = cloneExpiredContinuationSubjectV41(value.Subject)
	value.ManifestBytes = append([]byte(nil), value.ManifestBytes...)
	return value
}

func bytesSHA256(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func expiredContinuationRequestKeySHA256(value string) string {
	digest := sha256.New()
	_, _ = digest.Write(expiredContinuationRequestKeyDomain)
	appendDigestField(digest, []byte(value))
	return hex.EncodeToString(digest.Sum(nil))
}

func expiredContinuationOriginalRequestSHA256(
	key string,
	request microvm.SolicitudLanzamiento,
) string {
	digest := sha256.New()
	_, _ = digest.Write(expiredContinuationOriginalRequestDomain)
	appendDigestField(digest, []byte(key))
	appendDigestField(digest, request.Perfil)
	appendDigestField(digest, request.Plan)
	appendDigestField(digest, request.Concesion)
	return hex.EncodeToString(digest.Sum(nil))
}
