package application

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	agentEnvironmentManifestMediaType  = "application/vnd.orquesta.agent-environment-manifest.v1+json"
	agentEnvironmentInventoryMediaType = "application/vnd.orquesta.agent-environment-inventory.v1+json"
)

var errAgentEnvironmentPreservationBuilderInvalid = errors.New("application.agent_environment_preservation_builder_invalid")

type agentEnvironmentPreservationInventory struct {
	SchemaVersion uint64                                 `json:"schema_version"`
	Execution     agentEnvironmentPreservationExecution  `json:"execution"`
	Package       agentEnvironmentPreservationPackage    `json:"package"`
	Causality     agentEnvironmentPreservationCausality  `json:"causality"`
	Workspace     *agentEnvironmentPreservationWorkspace `json:"workspace"`
	Change        *agentEnvironmentPreservationChange    `json:"change"`
}

type agentEnvironmentPreservationExecution struct {
	GoalRef       string `json:"goal_ref"`
	WorkItemRef   string `json:"work_item_ref"`
	ExecutionRef  string `json:"execution_ref"`
	Attempt       uint64 `json:"attempt"`
	ExternalRef   string `json:"external_ref"`
	Fence         uint64 `json:"fence"`
	WorkRevision  string `json:"work_revision"`
	PhysicalToken string `json:"physical_token"`
}

type agentEnvironmentPreservationPackage struct {
	ArtifactRef     string `json:"artifact_ref"`
	SHA256          string `json:"sha256"`
	Bytes           uint64 `json:"bytes"`
	PhysicalRef     string `json:"physical_ref"`
	PhysicalReceipt string `json:"physical_receipt"`
	SealedAt        string `json:"sealed_at"`
}

type agentEnvironmentPreservationCausality struct {
	PlanSHA256      string `json:"plan_sha256"`
	GrantSHA256     string `json:"grant_sha256"`
	KernelSHA256    string `json:"kernel_sha256"`
	InitramfsSHA256 string `json:"initramfs_sha256"`
	ProfileSHA256   string `json:"profile_sha256"`
}

type agentEnvironmentPreservationWorkspace struct {
	Ref            string `json:"ref"`
	BindingSHA256  string `json:"binding_sha256"`
	BaseOID        string `json:"base_oid"`
	ObjectFormat   string `json:"object_format"`
	WriteSetSHA256 string `json:"write_set_sha256"`
	TargetRef      string `json:"target_ref"`
}

type agentEnvironmentPreservationChange struct {
	Ref            string `json:"ref"`
	ChangeSHA256   string `json:"change_sha256"`
	DiffSHA256     string `json:"diff_sha256"`
	HeadOID        string `json:"head_oid"`
	TreeOID        string `json:"tree_oid"`
	WriteSetSHA256 string `json:"write_set_sha256"`
}

// NewAgentEnvironmentPreservationBuilder composes the application-owned B12.2
// enrichment against the deployment's existing immutable ArtifactStore.
func NewAgentEnvironmentPreservationBuilder(
	artifacts ArtifactStore,
) (AgentEnvironmentPreservationBuilder, error) {
	if artifacts == nil {
		return nil, errAgentEnvironmentPreservationBuilderInvalid
	}
	return func(ctx context.Context, receipt ports.AgentPreserveReceipt, record GoalRecord,
		registeredAt time.Time,
	) (ComprobantePreservacionEntornoAgente, error) {
		return buildAgentEnvironmentPreservation(ctx, artifacts, receipt, record, registeredAt)
	}, nil
}

func buildAgentEnvironmentPreservation(
	ctx context.Context,
	artifacts ArtifactStore,
	receipt ports.AgentPreserveReceipt,
	record GoalRecord,
	registeredAt time.Time,
) (ComprobantePreservacionEntornoAgente, error) {
	request := ports.AgentPreserveRequest{
		Subject: receipt.Subject, ExpectedToken: receipt.PreviousToken, IdempotencyKey: receipt.IdempotencyKey,
	}
	if err := ctx.Err(); err != nil {
		return ComprobantePreservacionEntornoAgente{}, err
	}
	if ports.ValidateAgentPreserveReceipt(request, receipt) != nil ||
		receipt.NextToken.State != ports.AgentEnvironmentPreserved ||
		receipt.Manifest.Causality.ProfileSHA256 == "" || registeredAt.IsZero() ||
		registeredAt.Before(receipt.ConfirmedAt) ||
		validateAgentEnvironmentPreservationExecution(record, receipt) != nil {
		return ComprobantePreservacionEntornoAgente{}, errAgentEnvironmentPreservationBuilderInvalid
	}
	fence, err := strconv.ParseUint(receipt.NextToken.Fence.String(), 10, 64)
	if err != nil || fence == 0 {
		return ComprobantePreservacionEntornoAgente{}, errAgentEnvironmentPreservationBuilderInvalid
	}
	workspace, change, err := agentEnvironmentPreservationWorkspaceFacts(record, receipt)
	if err != nil {
		return ComprobantePreservacionEntornoAgente{}, err
	}
	packageRequest := ports.PutArtifactRequest{
		MediaType: agentEnvironmentManifestMediaType,
		Content:   append([]byte(nil), receipt.Manifest.Content...),
	}
	packageArtifact, err := artifacts.Put(ctx, packageRequest)
	if err != nil {
		return ComprobantePreservacionEntornoAgente{}, err
	}
	if ports.ValidateStoredArtifact(packageRequest, packageArtifact) != nil ||
		packageArtifact.Digest != receipt.Manifest.SHA256 {
		return ComprobantePreservacionEntornoAgente{}, errAgentEnvironmentPreservationBuilderInvalid
	}
	inventory := agentEnvironmentPreservationInventory{
		SchemaVersion: 1,
		Execution: agentEnvironmentPreservationExecution{
			GoalRef: receipt.Subject.GoalRef.String(), WorkItemRef: receipt.Subject.WorkItemRef.String(),
			ExecutionRef: receipt.Subject.ExecutionRef.String(), Attempt: receipt.Subject.ExecutionAttempt,
			ExternalRef: receipt.Subject.ExternalRef, Fence: fence,
			WorkRevision:  receipt.Manifest.WorkRevision.String(),
			PhysicalToken: receipt.NextToken.PhysicalToken.String(),
		},
		Package: agentEnvironmentPreservationPackage{
			ArtifactRef: packageArtifact.Ref.String(), SHA256: packageArtifact.Digest,
			Bytes: receipt.Manifest.ContentBytes, PhysicalRef: receipt.Manifest.Ref,
			PhysicalReceipt: receipt.ReceiptRef,
			SealedAt:        receipt.Manifest.SealedAt.UTC().Format(time.RFC3339Nano),
		},
		Causality: agentEnvironmentPreservationCausality{
			PlanSHA256:      receipt.Manifest.Causality.PlanSHA256,
			GrantSHA256:     receipt.Manifest.Causality.GrantSHA256,
			KernelSHA256:    receipt.Manifest.Causality.KernelSHA256,
			InitramfsSHA256: receipt.Manifest.Causality.InitramfsSHA256,
			ProfileSHA256:   receipt.Manifest.Causality.ProfileSHA256,
		},
		Workspace: workspace, Change: change,
	}
	inventoryContent, err := json.Marshal(inventory)
	if err != nil {
		return ComprobantePreservacionEntornoAgente{}, errAgentEnvironmentPreservationBuilderInvalid
	}
	inventoryRequest := ports.PutArtifactRequest{
		MediaType: agentEnvironmentInventoryMediaType, Content: inventoryContent,
	}
	inventoryArtifact, err := artifacts.Put(ctx, inventoryRequest)
	if err != nil {
		return ComprobantePreservacionEntornoAgente{}, err
	}
	if ports.ValidateStoredArtifact(inventoryRequest, inventoryArtifact) != nil {
		return ComprobantePreservacionEntornoAgente{}, errAgentEnvironmentPreservationBuilderInvalid
	}
	result := ports.ResultadoPreservacionEntornoAgente{
		Estado:       ports.EntornoAgentePreservadoPendienteRevision,
		EjecucionRef: receipt.Subject.ExecutionRef, IntentoEjecucion: receipt.Subject.ExecutionAttempt,
		Cerca: fence, IdentidadExterna: receipt.Subject.ExternalRef,
		PaqueteRef: packageArtifact.Ref, PaqueteDigest: packageArtifact.Digest,
		InventarioRef: inventoryArtifact.Ref, InventarioDigest: inventoryArtifact.Digest,
		ConfiguracionDigest: receipt.Manifest.Causality.ProfileSHA256,
		RootFSDigest:        receipt.Manifest.Causality.InitramfsSHA256,
		ComprobanteRef:      receipt.ReceiptRef, SelladoEn: receipt.Manifest.SealedAt,
		PreservadoEn: receipt.ConfirmedAt,
	}
	result.SelloDigest = ports.ResumenSelloPreservacionEntorno(result)
	fact := ComprobantePreservacionEntornoAgente{
		Ref:                 "environment-receipt:sha256:" + receipt.Manifest.SHA256,
		ClaveIdempotencia:   receipt.IdempotencyKey,
		ManifiestoFisicoRef: receipt.Manifest.Ref, ManifiestoFisicoDigest: receipt.Manifest.SHA256,
		ProyectoRef: record.Goal.Project(), ObjetivoRef: receipt.Subject.GoalRef,
		ItemRef: receipt.Subject.WorkItemRef, EjecucionRef: receipt.Subject.ExecutionRef,
		AlcanceEspacio: PreservacionEntornoSinEspacioTrabajo,
		Resultado:      result, RegistradoEn: registeredAt.UTC(),
	}
	if workspace != nil {
		binding, _ := exactAgentEnvironmentPreservationWorkspace(record, receipt.Subject.ExecutionRef)
		fact.AlcanceEspacio = PreservacionEntornoConEspacioTrabajo
		fact.EspacioTrabajoRef, fact.DigestBindingEspacio = binding.Ref, binding.Digest()
		fact.BaseOID, fact.FormatoObjeto = binding.BaseOID, binding.ObjectFormat
	}
	if change != nil {
		changeSet, _ := exactAgentEnvironmentPreservationChange(record, receipt.Subject.ExecutionRef)
		fact.CambioRef, fact.DigestCambio = changeSet.Ref, changeSet.Digest()
	}
	if ValidarComprobantePreservacionEntornoAgente(fact) != nil ||
		ValidarCausalidadPreservacionEntornoAgente(fact, record) != nil {
		return ComprobantePreservacionEntornoAgente{}, errAgentEnvironmentPreservationBuilderInvalid
	}
	return fact, nil
}

func validateAgentEnvironmentPreservationExecution(record GoalRecord, receipt ports.AgentPreserveReceipt) error {
	subject := receipt.Subject
	if record.Goal.Ref() != subject.GoalRef || record.Goal.Project().String() == "" ||
		record.Goal.SpecHash() != subject.SpecHash {
		return errAgentEnvironmentPreservationBuilderInvalid
	}
	var execution ExecutionRecord
	matches := 0
	for _, candidate := range record.Executions {
		if candidate.Ref == subject.ExecutionRef {
			execution, matches = candidate, matches+1
		}
	}
	if matches != 1 || execution.GoalRef != subject.GoalRef || execution.WorkItemRef != subject.WorkItemRef ||
		execution.AttemptNo != subject.ExecutionAttempt || execution.PlanGeneration != subject.PlanGeneration ||
		execution.AppSpecGeneration != subject.AppSpecGeneration || execution.SpecHash != subject.SpecHash ||
		execution.ProviderRef != subject.ProviderRef || execution.ModelRef != subject.ModelRef ||
		execution.AgentRef != subject.AgentRef || execution.ExternalRef != subject.ExternalRef ||
		!execution.RequierePreservacionEntorno || !validApplicationRef(execution.LaunchReceiptRef) {
		return errAgentEnvironmentPreservationBuilderInvalid
	}
	return nil
}

func agentEnvironmentPreservationWorkspaceFacts(
	record GoalRecord,
	receipt ports.AgentPreserveReceipt,
) (*agentEnvironmentPreservationWorkspace, *agentEnvironmentPreservationChange, error) {
	binding, bindingMatches := exactAgentEnvironmentPreservationWorkspace(record, receipt.Subject.ExecutionRef)
	changeSet, changeMatches := exactAgentEnvironmentPreservationChange(record, receipt.Subject.ExecutionRef)
	if bindingMatches > 1 || changeMatches > 1 {
		return nil, nil, errAgentEnvironmentPreservationBuilderInvalid
	}
	if bindingMatches == 0 {
		if changeMatches != 0 {
			return nil, nil, errAgentEnvironmentPreservationBuilderInvalid
		}
		return nil, nil, nil
	}
	if ValidateWorkspaceBinding(binding) != nil || binding.ProjectRef != record.Goal.Project() ||
		binding.GoalRef != receipt.Subject.GoalRef || binding.WorkItemRef != receipt.Subject.WorkItemRef ||
		binding.ExecutionAttempt != receipt.Subject.ExecutionAttempt ||
		binding.PlanGeneration != receipt.Subject.PlanGeneration ||
		binding.AppSpecGeneration != receipt.Subject.AppSpecGeneration || binding.SpecHash != receipt.Subject.SpecHash {
		return nil, nil, errAgentEnvironmentPreservationBuilderInvalid
	}
	workspace := &agentEnvironmentPreservationWorkspace{
		Ref: binding.Ref.String(), BindingSHA256: binding.Digest(), BaseOID: binding.BaseOID,
		ObjectFormat: string(binding.ObjectFormat), WriteSetSHA256: binding.WriteSetDigest,
		TargetRef: binding.TargetRef,
	}
	if changeMatches == 0 {
		return workspace, nil, nil
	}
	if ValidateChangeSet(changeSet) != nil || changeSet.WorkspaceRef != binding.Ref ||
		changeSet.ProjectRef != record.Goal.Project() || changeSet.GoalRef != receipt.Subject.GoalRef ||
		changeSet.WorkItemRef != receipt.Subject.WorkItemRef ||
		changeSet.ExecutionAttempt != receipt.Subject.ExecutionAttempt ||
		changeSet.PlanGeneration != receipt.Subject.PlanGeneration ||
		changeSet.AppSpecGeneration != receipt.Subject.AppSpecGeneration || changeSet.SpecHash != receipt.Subject.SpecHash {
		return nil, nil, errAgentEnvironmentPreservationBuilderInvalid
	}
	change := &agentEnvironmentPreservationChange{
		Ref: changeSet.Ref.String(), ChangeSHA256: changeSet.Digest(), DiffSHA256: changeSet.DiffDigest,
		HeadOID: changeSet.HeadOID, TreeOID: changeSet.TreeOID, WriteSetSHA256: changeSet.WriteSetDigest,
	}
	return workspace, change, nil
}

func exactAgentEnvironmentPreservationWorkspace(record GoalRecord, executionRef goal.ExecutionRef) (WorkspaceBinding, int) {
	var selected WorkspaceBinding
	matches := 0
	for _, binding := range record.WorkspaceBindings {
		if binding.ExecutionRef == executionRef {
			selected, matches = binding, matches+1
		}
	}
	return selected, matches
}

func exactAgentEnvironmentPreservationChange(record GoalRecord, executionRef goal.ExecutionRef) (ChangeSet, int) {
	var selected ChangeSet
	matches := 0
	for _, change := range record.ChangeSets {
		if change.ExecutionRef == executionRef {
			selected, matches = change, matches+1
		}
	}
	return selected, matches
}
