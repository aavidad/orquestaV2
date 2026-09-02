package agentmicrovm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeHistoricalAuthorityInvalid   = "agentmicrovm.historical_authority_invalid"
	CodeHistoricalSubjectMismatch    = "agentmicrovm.historical_subject_mismatch"
	CodePreserveUnavailable          = "agentmicrovm.preserve_unavailable"
	CodePreserveResponseInsufficient = "agentmicrovm.preserve_response_insufficient"
	CodePreserveRecoveryUnsupported  = "agentmicrovm.preserve_recovery_unsupported"
)

type historicalPreservationClient interface {
	Preservar(context.Context, string, string, microvm.SolicitudPreservacion) (microvm.RespuestaPreservacion, error)
	RecuperarManifiestoPreservacion(
		context.Context,
		string,
		microvm.SolicitudRecuperacionManifiestoPreservacion,
	) (microvm.RespuestaManifiestoPreservacion, error)
}

type physicalPreservationManifestV1 struct {
	Protocol  string            `json:"protocolo"`
	Context   json.RawMessage   `json:"contexto"`
	Artifacts []json.RawMessage `json:"artefactos"`
}

type physicalPreservationContextV1 struct {
	Reference         string  `json:"referencia"`
	LifecycleRevision uint64  `json:"revision_lifecycle"`
	Fence             uint64  `json:"cerca"`
	WorkRevision      uint64  `json:"revision_trabajo"`
	PlanSHA256        string  `json:"plan_sha256"`
	GrantSHA256       string  `json:"concesion_sha256"`
	KernelSHA256      string  `json:"kernel_sha256"`
	InitramfsSHA256   string  `json:"initramfs_sha256"`
	ProfileSHA256     *string `json:"perfil_sha256"`
}

type physicalPreservationArtifactV1 struct {
	Origin        string          `json:"origen"`
	Class         string          `json:"clase"`
	ContentRef    string          `json:"contenido"`
	ContentSHA256 string          `json:"contenido_sha256"`
	ContentBytes  uint64          `json:"contenido_bytes"`
	RootSHA256    *string         `json:"raiz_sha256"`
	Files         *uint32         `json:"archivos"`
	UsefulBytes   uint64          `json:"bytes_utiles"`
	Detail        json.RawMessage `json:"detalle"`
}

const physicalPreservationProtocolV1 = "agentmicrovm.preservacion.v1"

// PreserveWithAuthority mutates the exact historical execution once and then
// reads its immutable manifest through the public sibling connector. The
// manifest itself supplies the physical receipt ref and seal time; application
// supplies the launch-time digests against which its causal context is bound.
func (adapter *Adapter) PreserveWithAuthority(
	ctx context.Context,
	authority ports.AgentHistoricalRuntimeAuthority,
	request ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	if adapter == nil || nilInterface(ctx) || ports.ValidateAgentHistoricalRuntimeAuthority(authority) != nil ||
		ports.ValidateAgentPreserveRequest(request) != nil {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, nil)
	}
	if authority.Subject != request.Subject {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalSubjectMismatch, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentPreserveReceipt{}, err
	}
	revision, err := strconv.ParseUint(request.ExpectedToken.Revision.String(), 10, 64)
	if err != nil || revision == 0 || revision > maxDurableCounter {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, err)
	}
	fence, err := strconv.ParseUint(request.ExpectedToken.Fence.String(), 10, 64)
	if err != nil || fence == 0 || fence > maxDurableCounter || fence != authority.Key.ActionFence {
		return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, err)
	}
	client, ok := adapter.client.(historicalPreservationClient)
	if !ok || nilInterface(client) {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveUnavailable, nil)
	}
	response, err := client.Preservar(ctx, request.IdempotencyKey, request.Subject.ExternalRef,
		microvm.SolicitudPreservacion{RevisionEsperada: revision, Cerca: fence})
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentPreserveReceipt{}, contextErr
		}
		return ports.AgentPreserveReceipt{}, fail(CodePreserveUnavailable, err)
	}
	if response.Ejecucion.Referencia != request.Subject.ExternalRef ||
		response.Ejecucion.Estado != "preservada" || response.Ejecucion.Cerca != fence ||
		response.Ejecucion.Revision <= revision || response.Ejecucion.Revision > maxDurableCounter ||
		response.RevisionTrabajo == 0 || response.RevisionTrabajo > maxDurableCounter ||
		!preservationDigestValid(response.ManifiestoSHA256) || response.ManifiestoBytes == 0 {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, nil)
	}
	manifestResponse, err := client.RecuperarManifiestoPreservacion(
		ctx,
		request.Subject.ExternalRef,
		microvm.SolicitudRecuperacionManifiestoPreservacion{
			Cerca:            fence,
			RevisionTrabajo:  response.RevisionTrabajo,
			ManifiestoRef:    response.ManifiestoSHA256,
			ManifiestoSHA256: response.ManifiestoSHA256,
			ManifiestoBytes:  response.ManifiestoBytes,
		},
	)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentPreserveReceipt{}, contextErr
		}
		return ports.AgentPreserveReceipt{}, fail(CodePreserveUnavailable, err)
	}
	if manifestResponse.Referencia != request.Subject.ExternalRef || manifestResponse.Cerca != fence ||
		manifestResponse.RevisionTrabajo != response.RevisionTrabajo ||
		manifestResponse.ManifiestoRef != response.ManifiestoSHA256 ||
		manifestResponse.ManifiestoSHA256 != response.ManifiestoSHA256 ||
		manifestResponse.ManifiestoRef != manifestResponse.ManifiestoSHA256 ||
		manifestResponse.ManifiestoBytes != response.ManifiestoBytes {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, nil)
	}
	content, err := base64.StdEncoding.Strict().DecodeString(manifestResponse.ContenidoBase64)
	if err != nil || base64.StdEncoding.EncodeToString(content) != manifestResponse.ContenidoBase64 ||
		uint64(len(content)) != manifestResponse.ManifiestoBytes || manifestResponse.SelladaUnixMS == 0 ||
		manifestResponse.SelladaUnixMS > maxDurableCounter ||
		!validPhysicalPreservationManifest(content, request, authority, response) {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, err)
	}
	nextRevision, err := ports.NewAgentPhysicalRevision(strconv.FormatUint(response.Ejecucion.Revision, 10))
	if err != nil {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, err)
	}
	workRevision, err := ports.NewAgentPhysicalRevision(strconv.FormatUint(response.RevisionTrabajo, 10))
	if err != nil {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, err)
	}
	sealedAt := time.UnixMilli(int64(manifestResponse.SelladaUnixMS)).UTC()
	receipt := ports.AgentPreserveReceipt{
		Subject:       request.Subject,
		PreviousToken: request.ExpectedToken,
		NextToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: request.ExpectedToken.PhysicalToken,
			Revision:      nextRevision,
			Fence:         request.ExpectedToken.Fence,
			State:         ports.AgentEnvironmentPreserved,
		},
		IdempotencyKey: request.IdempotencyKey,
		Manifest: ports.AgentPhysicalPreservationManifest{
			Ref:          manifestResponse.ManifiestoRef,
			SHA256:       manifestResponse.ManifiestoSHA256,
			Content:      content,
			ContentBytes: manifestResponse.ManifiestoBytes,
			WorkRevision: workRevision,
			Causality: ports.AgentPhysicalPreservationCausality{
				PlanSHA256: authority.Digests.PlanSHA256, GrantSHA256: authority.Digests.GrantSHA256,
				KernelSHA256: authority.Digests.KernelSHA256, InitramfsSHA256: authority.Digests.InitramfsSHA256,
				ProfileSHA256: authority.Digests.ProfileSHA256,
			},
			SealedAt: sealedAt,
		},
		ReceiptRef:  manifestResponse.ManifiestoRef,
		ConfirmedAt: sealedAt,
	}
	if ports.ValidateAgentPreserveReceipt(request, receipt) != nil {
		return ports.AgentPreserveReceipt{}, fail(CodePreserveResponseInsufficient, nil)
	}
	return receipt, nil
}

// Preserve cannot recover the missing historical launch fence and digests
// from AgentPreserveRequest, so the generic lifecycle boundary is closed.
func (adapter *Adapter) Preserve(context.Context, ports.AgentPreserveRequest) (ports.AgentPreserveReceipt, error) {
	return ports.AgentPreserveReceipt{}, fail(CodeHistoricalAuthorityInvalid, nil)
}

// ReconcilePreserve is read-only, but the neutral request contains neither the
// manifest ref nor its exact byte count required by the stable sibling API.
func (adapter *Adapter) ReconcilePreserve(context.Context, ports.AgentPreserveRequest) (ports.AgentPreserveReceipt, error) {
	return ports.AgentPreserveReceipt{}, fail(CodePreserveRecoveryUnsupported, nil)
}

func validPhysicalPreservationManifest(
	content []byte,
	request ports.AgentPreserveRequest,
	authority ports.AgentHistoricalRuntimeAuthority,
	response microvm.RespuestaPreservacion,
) bool {
	var manifest physicalPreservationManifestV1
	if !decodeExactPreservationJSON(content, &manifest, "protocolo", "contexto", "artefactos") ||
		manifest.Protocol != physicalPreservationProtocolV1 {
		return false
	}
	var physicalContext physicalPreservationContextV1
	if !decodeExactPreservationJSON(manifest.Context, &physicalContext,
		"referencia", "revision_lifecycle", "cerca", "revision_trabajo", "plan_sha256",
		"concesion_sha256", "kernel_sha256", "initramfs_sha256", "perfil_sha256") ||
		physicalContext.Reference != request.Subject.ExternalRef ||
		physicalContext.LifecycleRevision != response.Ejecucion.Revision ||
		physicalContext.Fence != response.Ejecucion.Cerca ||
		physicalContext.WorkRevision != response.RevisionTrabajo ||
		physicalContext.PlanSHA256 != authority.Digests.PlanSHA256 ||
		physicalContext.GrantSHA256 != authority.Digests.GrantSHA256 ||
		physicalContext.KernelSHA256 != authority.Digests.KernelSHA256 ||
		physicalContext.InitramfsSHA256 != authority.Digests.InitramfsSHA256 ||
		physicalContext.ProfileSHA256 == nil || *physicalContext.ProfileSHA256 != authority.Digests.ProfileSHA256 ||
		len(manifest.Artifacts) != len(response.Artefactos) {
		return false
	}
	var usefulBytes uint64
	for index, raw := range manifest.Artifacts {
		var artifact physicalPreservationArtifactV1
		if !decodeExactPreservationJSON(raw, &artifact,
			"origen", "clase", "contenido", "contenido_sha256", "contenido_bytes",
			"raiz_sha256", "archivos", "bytes_utiles", "detalle") ||
			artifact.Origin == "" || artifact.Class == "" || artifact.ContentRef == "" ||
			!preservationDigestValid(artifact.ContentSHA256) || !validPreservationDetail(artifact.Detail) {
			return false
		}
		public := response.Artefactos[index]
		if artifact.Origin != public.Origen || artifact.Class != public.Clase ||
			artifact.ContentSHA256 != public.ContenidoSHA256 || artifact.ContentBytes != public.ContenidoBytes ||
			!equalOptionalString(artifact.RootSHA256, public.RaizSHA256) ||
			!equalOptionalUint32(artifact.Files, public.Archivos) || artifact.UsefulBytes != public.BytesUtiles ||
			(^uint64(0)-usefulBytes) < artifact.UsefulBytes {
			return false
		}
		usefulBytes += artifact.UsefulBytes
	}
	return usefulBytes == response.BytesUtiles
}

func validPreservationDetail(raw json.RawMessage) bool {
	var object map[string]json.RawMessage
	if !decodePreservationJSON(raw, &object) {
		return false
	}
	var kind string
	if encoded, present := object["tipo"]; !present || json.Unmarshal(encoded, &kind) != nil {
		return false
	}
	switch kind {
	case "orden":
		var detail struct {
			Type            string `json:"tipo"`
			ExitCode        *int32 `json:"codigo_salida"`
			Signal          *int32 `json:"senal"`
			Exhausted       bool   `json:"agotada"`
			OutputTruncated bool   `json:"salida_truncada"`
		}
		return decodeExactPreservationJSON(raw, &detail,
			"tipo", "codigo_salida", "senal", "agotada", "salida_truncada")
	case "sincronizacion":
		var detail struct {
			Type                  string `json:"tipo"`
			Direction             string `json:"direccion"`
			RelativePath          string `json:"ruta_relativa"`
			RequestedWorkRevision uint64 `json:"revision_trabajo_solicitada"`
			ResultWorkRevision    uint64 `json:"revision_trabajo_resultado"`
			GuestConfirmed        bool   `json:"confirmada_huesped"`
		}
		return decodeExactPreservationJSON(raw, &detail,
			"tipo", "direccion", "ruta_relativa", "revision_trabajo_solicitada",
			"revision_trabajo_resultado", "confirmada_huesped") &&
			(detail.Direction == "entrada" || detail.Direction == "salida") && detail.RelativePath != "" &&
			detail.RequestedWorkRevision > 0 && detail.ResultWorkRevision > 0
	default:
		return false
	}
}

func decodeExactPreservationJSON(raw []byte, target any, fields ...string) bool {
	var object map[string]json.RawMessage
	if !decodePreservationJSON(raw, &object) || len(object) != len(fields) {
		return false
	}
	for _, field := range fields {
		if _, present := object[field]; !present {
			return false
		}
	}
	return decodePreservationJSON(raw, target)
}

func decodePreservationJSON(raw []byte, target any) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	return errors.Is(decoder.Decode(&struct{}{}), io.EOF)
}

func preservationDigestValid(value string) bool {
	if len(value) != 64 {
		return false
	}
	for index := range value {
		if (value[index] < '0' || value[index] > '9') && (value[index] < 'a' || value[index] > 'f') {
			return false
		}
	}
	return true
}

func equalOptionalString(left, right *string) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func equalOptionalUint32(left, right *uint32) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}
