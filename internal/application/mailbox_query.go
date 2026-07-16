package application

import (
	"context"
	"errors"

	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) GetMailbox(
	ctx context.Context,
	access Access,
	request GetMailboxRequest,
) (MailboxRecord, error) {
	if orchestrator == nil {
		return MailboxRecord{}, errors.New("application.unavailable")
	}
	if err := validateMailboxAddressedRequest(
		"query", request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
	); err != nil {
		return MailboxRecord{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return MailboxRecord{}, err
	}
	if _, err := access.authenticatedExecution(request.RecipientExecutionRef); err != nil {
		return MailboxRecord{}, err
	}
	if _, err := orchestrator.authorizeRead(
		ctx, access, identity.PermissionGoalsGet, request.MessageRef.String(),
	); err != nil {
		return MailboxRecord{}, err
	}
	endpoint := MailboxEndpoint{PrincipalRef: principal.Ref, WorkItemRef: request.RecipientWorkItemRef, ExecutionRef: request.RecipientExecutionRef}
	record, err := orchestrator.state.GetMailbox(ctx, projectRef, request.GoalRef, request.MessageRef, endpoint)
	if err != nil {
		return MailboxRecord{}, err
	}
	if err := validateMailboxRecordAddress(record, projectRef, request.GoalRef, request.MessageRef, endpoint); err != nil {
		return MailboxRecord{}, err
	}
	return cloneMailboxRecord(record), nil
}

func (orchestrator *Orchestrator) ListMailbox(
	ctx context.Context,
	access Access,
	request ListMailboxRequest,
) (ListMailboxResult, error) {
	if orchestrator == nil {
		return nil, errors.New("application.unavailable")
	}
	if request.GoalRef.String() == "" || request.RecipientWorkItemRef.String() == "" ||
		request.RecipientExecutionRef.String() == "" || request.Limit <= 0 {
		return nil, errors.New("application.mailbox_query_invalid")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return nil, err
	}
	if _, err := access.authenticatedExecution(request.RecipientExecutionRef); err != nil {
		return nil, err
	}
	if _, err := orchestrator.authorizeRead(
		ctx, access, identity.PermissionGoalsGet, request.GoalRef.String(),
	); err != nil {
		return nil, err
	}
	endpoint := MailboxEndpoint{PrincipalRef: principal.Ref, WorkItemRef: request.RecipientWorkItemRef, ExecutionRef: request.RecipientExecutionRef}
	records, err := orchestrator.state.ListMailbox(ctx, projectRef, request.GoalRef, endpoint, request.Limit)
	if err != nil {
		return nil, err
	}
	if len(records) > request.Limit {
		return nil, &StateError{Code: StateConflict}
	}
	result := make(ListMailboxResult, 0, len(records))
	for _, record := range records {
		if err := validateMailboxRecordAddress(record, projectRef, request.GoalRef, record.Envelope.Ref, endpoint); err != nil {
			return nil, err
		}
		result = append(result, cloneMailboxRecord(record))
	}
	return result, nil
}
