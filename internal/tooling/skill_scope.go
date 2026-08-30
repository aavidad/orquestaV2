package tooling

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const (
	SkillScopeGlobal  SkillScopeKind = "global"
	SkillScopeProduct SkillScopeKind = "product"
	SkillScopeProject SkillScopeKind = "project"
	SkillScopeRole    SkillScopeKind = "role"
	SkillScopeGoal    SkillScopeKind = "goal"

	MaxSkillScopes          = 32
	MaxSkillScopeValueBytes = 200

	ErrorSkillScopeInvalid = "tooling.skill_scope_invalid"
)

type SkillScopeKind string

// SkillScope is one declarative visibility selector. Authorization remains an
// application concern: matching a scope never grants any skill permission.
type SkillScope struct {
	Kind       SkillScopeKind `json:"kind"`
	ProductID  string         `json:"product_id,omitempty"`
	ProjectRef string         `json:"project_ref,omitempty"`
	Role       identity.Role  `json:"role,omitempty"`
	GoalRef    string         `json:"goal_ref,omitempty"`
}

// SkillScopeContext contains only explicit, already-typed visibility
// coordinates. It deliberately carries no actor or membership proof: those
// remain application authorization inputs and a match never replaces them.
// Its private representation prevents callers from forging a validated zero
// value or inferring project, role, or Goal from another field.
type SkillScopeContext struct {
	productID  string
	projectRef goal.ProjectRef
	role       identity.Role
	goalRef    goal.GoalRef
	valid      bool
}

func NewSkillScopeContext(
	productID string,
	projectRef goal.ProjectRef,
	role identity.Role,
	goalRef goal.GoalRef,
) (SkillScopeContext, error) {
	if productID != "" && !validSkillScopeValue(productID) {
		return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "product_id")
	}
	if value := projectRef.String(); value != "" {
		if !validSkillScopeValue(value) {
			return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "project_ref")
		}
		if _, err := goal.NewProjectRef(value); err != nil {
			return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "project_ref")
		}
	}
	if role != "" && identity.ValidateRole(role) != nil {
		return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "role")
	}
	if value := goalRef.String(); value != "" {
		if !validSkillScopeValue(value) {
			return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "goal_ref")
		}
		if _, err := goal.NewGoalRef(value); err != nil {
			return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "goal_ref")
		}
	}
	if (role != "" || goalRef.String() != "") && projectRef.String() == "" {
		return SkillScopeContext{}, skillContractError(ErrorSkillScopeInvalid, "project_ref")
	}
	return SkillScopeContext{
		productID: productID, projectRef: projectRef, role: role, goalRef: goalRef, valid: true,
	}, nil
}

// ResolveScoped resolves one exact skill version only when at least one of its
// declared scopes matches the explicit request context. A match is visibility,
// not authorization, and does not modify the registered permission set.
func (registry *SkillRegistry) ResolveScoped(
	id string,
	version string,
	context SkillScopeContext,
) (SkillRegistration, bool, error) {
	if !context.valid {
		return SkillRegistration{}, false, skillContractError(ErrorSkillScopeInvalid, "context")
	}
	registration, found := registry.Lookup(id, version)
	if !found {
		return SkillRegistration{}, false, nil
	}
	for _, scope := range registration.Spec.Scopes {
		if scope.matches(context) {
			return registration, true, nil
		}
	}
	return SkillRegistration{}, false, nil
}

func canonicalSkillScopes(source []SkillScope) ([]SkillScope, error) {
	if len(source) == 0 || len(source) > MaxSkillScopes {
		return nil, skillContractError(ErrorSkillSpecInvalid, "scopes")
	}
	scopes := append([]SkillScope(nil), source...)
	for _, scope := range scopes {
		if !validSkillScope(scope) {
			return nil, skillContractError(ErrorSkillSpecInvalid, "scopes")
		}
	}
	sort.Slice(scopes, func(left, right int) bool {
		if skillScopeKindOrder(scopes[left].Kind) != skillScopeKindOrder(scopes[right].Kind) {
			return skillScopeKindOrder(scopes[left].Kind) < skillScopeKindOrder(scopes[right].Kind)
		}
		return skillScopeKey(scopes[left]) < skillScopeKey(scopes[right])
	})
	for index := range scopes {
		if index > 0 && skillScopeKey(scopes[index]) == skillScopeKey(scopes[index-1]) {
			return nil, skillContractError(ErrorSkillSpecInvalid, "scopes")
		}
		if scopes[index].Kind == SkillScopeGlobal && len(scopes) != 1 {
			return nil, skillContractError(ErrorSkillSpecInvalid, "scopes")
		}
	}
	return scopes, nil
}

func validSkillScope(scope SkillScope) bool {
	switch scope.Kind {
	case SkillScopeGlobal:
		return scope.ProductID == "" && scope.ProjectRef == "" && scope.Role == "" && scope.GoalRef == ""
	case SkillScopeProduct:
		return validSkillScopeValue(scope.ProductID) && scope.ProjectRef == "" && scope.Role == "" && scope.GoalRef == ""
	case SkillScopeProject:
		if !validSkillScopeValue(scope.ProjectRef) {
			return false
		}
		_, err := goal.NewProjectRef(scope.ProjectRef)
		return err == nil && scope.ProductID == "" && scope.Role == "" && scope.GoalRef == ""
	case SkillScopeRole:
		return identity.ValidateRole(scope.Role) == nil && scope.ProductID == "" && scope.ProjectRef == "" && scope.GoalRef == ""
	case SkillScopeGoal:
		if !validSkillScopeValue(scope.ProjectRef) || !validSkillScopeValue(scope.GoalRef) {
			return false
		}
		_, err := goal.NewGoalRef(scope.GoalRef)
		_, projectErr := goal.NewProjectRef(scope.ProjectRef)
		return err == nil && projectErr == nil && scope.ProductID == "" && scope.Role == ""
	default:
		return false
	}
}

func validSkillScopeValue(value string) bool {
	if value == "" || len(value) > MaxSkillScopeValueBytes || !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

// skillScopeKindOrder fixes canonical serialization only. Non-global scopes
// are an explicit union: matching more than one never overrides another or
// grants a different result.
func skillScopeKindOrder(kind SkillScopeKind) int {
	switch kind {
	case SkillScopeGlobal:
		return 0
	case SkillScopeProduct:
		return 1
	case SkillScopeProject:
		return 2
	case SkillScopeRole:
		return 3
	case SkillScopeGoal:
		return 4
	default:
		return 5
	}
}

func skillScopeKey(scope SkillScope) string {
	return string(scope.Kind) + "\x00" + scope.ProductID + "\x00" + scope.ProjectRef +
		"\x00" + string(scope.Role) + "\x00" + scope.GoalRef
}

func (scope SkillScope) matches(context SkillScopeContext) bool {
	switch scope.Kind {
	case SkillScopeGlobal:
		return true
	case SkillScopeProduct:
		return scope.ProductID == context.productID
	case SkillScopeProject:
		return scope.ProjectRef == context.projectRef.String()
	case SkillScopeRole:
		return scope.Role == context.role
	case SkillScopeGoal:
		return scope.ProjectRef == context.projectRef.String() && scope.GoalRef == context.goalRef.String()
	default:
		return false
	}
}
