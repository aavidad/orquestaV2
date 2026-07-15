package identity

import (
	"errors"

	"orquesta/internal/goal"
)

type ProjectHierarchyInput struct {
	WorkspaceRef               WorkspaceRef
	GroupRef                   GroupRef
	GroupParentWorkspaceRef    WorkspaceRef
	ProjectRef                 goal.ProjectRef
	ProjectParentGroupRef      GroupRef
	RepositoryRef              RepositoryRef
	RepositoryParentProjectRef goal.ProjectRef
}

// ProjectHierarchy is a logical ownership chain. Its refs have no filesystem
// or version-control interpretation.
type ProjectHierarchy struct {
	workspaceRef               WorkspaceRef
	groupRef                   GroupRef
	groupParentWorkspaceRef    WorkspaceRef
	projectRef                 goal.ProjectRef
	projectParentGroupRef      GroupRef
	repositoryRef              RepositoryRef
	repositoryParentProjectRef goal.ProjectRef
}

func NewProjectHierarchy(input ProjectHierarchyInput) (ProjectHierarchy, error) {
	if input.WorkspaceRef.String() == "" || input.GroupRef.String() == "" ||
		input.GroupParentWorkspaceRef.String() == "" || input.ProjectRef.String() == "" ||
		input.ProjectParentGroupRef.String() == "" || input.RepositoryRef.String() == "" ||
		input.RepositoryParentProjectRef.String() == "" {
		return ProjectHierarchy{}, errors.New("identity.hierarchy_ref_required")
	}
	if input.GroupParentWorkspaceRef != input.WorkspaceRef {
		return ProjectHierarchy{}, errors.New("identity.group_parent_mismatch")
	}
	if input.ProjectParentGroupRef != input.GroupRef {
		return ProjectHierarchy{}, errors.New("identity.project_parent_mismatch")
	}
	if input.RepositoryParentProjectRef != input.ProjectRef {
		return ProjectHierarchy{}, errors.New("identity.repository_parent_mismatch")
	}
	return ProjectHierarchy{
		workspaceRef: input.WorkspaceRef, groupRef: input.GroupRef,
		groupParentWorkspaceRef: input.GroupParentWorkspaceRef,
		projectRef:              input.ProjectRef, projectParentGroupRef: input.ProjectParentGroupRef,
		repositoryRef:              input.RepositoryRef,
		repositoryParentProjectRef: input.RepositoryParentProjectRef,
	}, nil
}

func (hierarchy ProjectHierarchy) WorkspaceRef() WorkspaceRef {
	return hierarchy.workspaceRef
}

func (hierarchy ProjectHierarchy) GroupRef() GroupRef {
	return hierarchy.groupRef
}

func (hierarchy ProjectHierarchy) GroupParentWorkspaceRef() WorkspaceRef {
	return hierarchy.groupParentWorkspaceRef
}

func (hierarchy ProjectHierarchy) ProjectRef() goal.ProjectRef {
	return hierarchy.projectRef
}

func (hierarchy ProjectHierarchy) ProjectParentGroupRef() GroupRef {
	return hierarchy.projectParentGroupRef
}

func (hierarchy ProjectHierarchy) RepositoryRef() RepositoryRef {
	return hierarchy.repositoryRef
}

func (hierarchy ProjectHierarchy) RepositoryParentProjectRef() goal.ProjectRef {
	return hierarchy.repositoryParentProjectRef
}

func (hierarchy ProjectHierarchy) Snapshot() ProjectHierarchyInput {
	return ProjectHierarchyInput{
		WorkspaceRef: hierarchy.workspaceRef, GroupRef: hierarchy.groupRef,
		GroupParentWorkspaceRef: hierarchy.groupParentWorkspaceRef,
		ProjectRef:              hierarchy.projectRef, ProjectParentGroupRef: hierarchy.projectParentGroupRef,
		RepositoryRef:              hierarchy.repositoryRef,
		RepositoryParentProjectRef: hierarchy.repositoryParentProjectRef,
	}
}
