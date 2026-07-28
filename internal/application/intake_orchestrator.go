package application

import (
	"context"
	"errors"

	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) CreateIntake(
	ctx context.Context,
	access Access,
	request CreateIntakeRequest,
) (IntakeResult, error) {
	if orchestrator == nil || orchestrator.intake == nil {
		return IntakeResult{}, errors.New("application.unavailable")
	}
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(
		IntakeOperationCreate, request.RequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntakeResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionGoalsCreate,
		projectRef.String(),
		orchestrator.clock.Now(),
		authorizationRequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	request.AuthorizationReceipt = authorization
	return orchestrator.intake.CreateIntake(ctx, request)
}

func (orchestrator *Orchestrator) GetIntake(
	ctx context.Context,
	access Access,
	request GetIntakeRequest,
) (IntakeRecord, error) {
	if orchestrator == nil || orchestrator.intake == nil {
		return IntakeRecord{}, errors.New("application.unavailable")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntakeRecord{}, err
	}
	if _, err = orchestrator.authorizeRead(
		ctx, access, identity.PermissionGoalsGet, string(request.StateRef),
	); err != nil {
		return IntakeRecord{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	return orchestrator.intake.GetIntake(ctx, request)
}

func (orchestrator *Orchestrator) ApplyIntake(
	ctx context.Context,
	access Access,
	request ApplyIntakeRequest,
) (IntakeResult, error) {
	if orchestrator == nil || orchestrator.intake == nil {
		return IntakeResult{}, errors.New("application.unavailable")
	}
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(
		IntakeOperationApply, request.RequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntakeResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionGoalsCreate,
		projectRef.String(),
		orchestrator.clock.Now(),
		authorizationRequestRef,
	)
	if err != nil {
		return IntakeResult{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	request.AuthorizationReceipt = authorization
	return orchestrator.intake.ApplyIntake(ctx, request)
}

// ApplyWizardGaps binds authenticated authority before the versioned Wizard
// evaluator compiles findings into the shared Intake writer.
func (orchestrator *Orchestrator) ApplyWizardGaps(
	ctx context.Context,
	access Access,
	request ApplyWizardGapsRequest,
) (ApplyWizardGapsResult, error) {
	if orchestrator == nil || orchestrator.wizardGaps == nil {
		return ApplyWizardGapsResult{}, errors.New("application.unavailable")
	}
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(
		IntakeOperationApply, request.RequestRef,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionGoalsCreate,
		projectRef.String(),
		orchestrator.clock.Now(),
		authorizationRequestRef,
	)
	if err != nil {
		return ApplyWizardGapsResult{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	request.AuthorizationReceipt = authorization
	return orchestrator.wizardGaps.ApplyWizardGaps(ctx, request)
}

func (orchestrator *Orchestrator) PrepareIntakeDossier(
	ctx context.Context,
	access Access,
	request PrepareIntakeDossierRequest,
) (IntakeDossierResult, error) {
	if orchestrator == nil || orchestrator.intakeDossier == nil {
		return IntakeDossierResult{}, errors.New("application.unavailable")
	}
	authorizationRequestRef, err := IntakeDossierAuthorizationRequestRef(request.RequestRef)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntakeDossierResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionGoalsCreate,
		projectRef.String(),
		orchestrator.clock.Now(),
		authorizationRequestRef,
	)
	if err != nil {
		return IntakeDossierResult{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	request.AuthorizationReceipt = authorization
	return orchestrator.intakeDossier.PrepareIntakeDossier(ctx, request)
}

// PrepareWizardDossier exposes the same authorization boundary as generic
// dossier preparation. Template resolution and projection remain owned by the
// IntakeDossierService; Orchestrator only binds authenticated authority.
func (orchestrator *Orchestrator) PrepareWizardDossier(
	ctx context.Context,
	access Access,
	request PrepareWizardDossierRequest,
) (WizardDossierResult, error) {
	if orchestrator == nil || orchestrator.intakeDossier == nil {
		return WizardDossierResult{}, errors.New("application.unavailable")
	}
	authorizationRequestRef, err := IntakeDossierAuthorizationRequestRef(
		request.RequestRef,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return WizardDossierResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionGoalsCreate,
		projectRef.String(),
		orchestrator.clock.Now(),
		authorizationRequestRef,
	)
	if err != nil {
		return WizardDossierResult{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	request.AuthorizationReceipt = authorization
	return orchestrator.intakeDossier.PrepareWizardDossier(ctx, request)
}

func (orchestrator *Orchestrator) GetIntakeDossier(
	ctx context.Context,
	access Access,
	request GetIntakeDossierRequest,
) (IntakeDossierRecord, error) {
	if orchestrator == nil || orchestrator.intakeDossier == nil {
		return IntakeDossierRecord{}, errors.New("application.unavailable")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntakeDossierRecord{}, err
	}
	if _, err = orchestrator.authorizeRead(
		ctx,
		access,
		identity.PermissionGoalsGet,
		string(request.DossierRef),
	); err != nil {
		return IntakeDossierRecord{}, err
	}
	request.ActorRef = principal.ActorRef
	request.ProjectRef = projectRef
	return orchestrator.intakeDossier.GetIntakeDossier(ctx, request)
}
