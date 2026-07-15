package identity

import (
	"errors"
	"strings"
)

type PrincipalRef struct{ value string }
type WorkspaceRef struct{ value string }
type GroupRef struct{ value string }
type RepositoryRef struct{ value string }

func NewPrincipalRef(value string) (PrincipalRef, error) {
	value, err := validateOpaqueRef("principal_ref", value)
	return PrincipalRef{value: value}, err
}

func NewWorkspaceRef(value string) (WorkspaceRef, error) {
	value, err := validateOpaqueRef("workspace_ref", value)
	return WorkspaceRef{value: value}, err
}

func NewGroupRef(value string) (GroupRef, error) {
	value, err := validateOpaqueRef("group_ref", value)
	return GroupRef{value: value}, err
}

func NewRepositoryRef(value string) (RepositoryRef, error) {
	value, err := validateOpaqueRef("repository_ref", value)
	return RepositoryRef{value: value}, err
}

func (ref PrincipalRef) String() string  { return ref.value }
func (ref WorkspaceRef) String() string  { return ref.value }
func (ref GroupRef) String() string      { return ref.value }
func (ref RepositoryRef) String() string { return ref.value }

func validateOpaqueRef(field, value string) (string, error) {
	if !validOpaqueValue(value) {
		return "", errors.New("identity.invalid_" + field)
	}
	return value, nil
}

func validOpaqueValue(value string) bool {
	return value != "" && strings.TrimSpace(value) == value &&
		!strings.ContainsRune(value, '\x00') && !strings.ContainsAny(value, "\r\n")
}

func validCode(value string) bool {
	return validOpaqueValue(value) && !strings.ContainsAny(value, "\r\n")
}
