package main

import (
	"context"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	defaultAutoprogrammingPromotionRepoRefV0 = "repo-ref-orquesta-project"
	defaultAutoprogrammingPromotionAppRefV0  = "app-ref-orquesta-project"
	defaultAutoprogrammingPromotionMessageV0 = "chore: promote autoprogramming staging"
)

type serverAutoprogrammingPromotionPortV0 struct {
	ProjectWorkDir string
	ArchiveDir     string
	RepoRef        string
	AppRef         string
	CommitMessage  string
	Connector      orquestaruntimeworktree.GitStagingPromotionConnectorV0
}

func autoprogrammingPromotionConfigFromEnvV0(
	config orquestaserver.ConfigV0,
) orquestaappcodexstack.AutoprogrammingPromotionConfigV0 {
	if !boolEnvOrDefaultV0("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED", false) {
		return orquestaappcodexstack.AutoprogrammingPromotionConfigV0{}
	}
	archiveDir := absDirEnvOrDefaultV0(
		"ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ARCHIVE_DIR",
		filepath.Join(config.StateDir, "autoprogramming-promotion-archive"),
	)
	port := serverAutoprogrammingPromotionPortV0{
		ProjectWorkDir: strings.TrimSpace(config.ProjectWorkDir),
		ArchiveDir:     archiveDir,
		RepoRef:        envOrDefaultV0("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_REPO_REF", defaultAutoprogrammingPromotionRepoRefV0),
		AppRef:         envOrDefaultV0("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_APP_REF", defaultAutoprogrammingPromotionAppRefV0),
		CommitMessage:  envOrDefaultV0("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_COMMIT_MESSAGE", defaultAutoprogrammingPromotionMessageV0),
	}
	return orquestaappcodexstack.AutoprogrammingPromotionConfigV0{
		Enabled:       true,
		Port:          port,
		AppRef:        port.AppRef,
		RepoRef:       port.RepoRef,
		CommitMessage: port.CommitMessage,
	}
}

func (port serverAutoprogrammingPromotionPortV0) PromoteAutoprogrammingStagingV0(
	ctx context.Context,
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	result, issues := port.Connector.PromoteStagingWorktreeV0(ctx, orquestaruntimeworktree.StagingPromotionRequestV0{
		PromotionRef:   command.PromotionRef,
		RunRef:         command.RunRef,
		ProjectRef:     command.ProjectRef,
		AppRef:         port.AppRef,
		RepoRef:        port.RepoRef,
		WorktreeRef:    command.WorktreeRef,
		BranchRef:      command.BranchRef,
		ProjectWorkDir: port.ProjectWorkDir,
		CommitMessage:  port.CommitMessage,
		WriteSet:       append([]string(nil), command.WriteSet...),
		EvidenceRefs:   append([]string(nil), command.EvidenceRefs...),
	})
	return autoprogrammingPromotionEffectFromWorktreeV0(result, issues), nil
}

func (port serverAutoprogrammingPromotionPortV0) ArchiveAutoprogrammingStagingV0(
	ctx context.Context,
	command orquestaautoprogramming.AutoprogrammingStagingCleanupCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	result, issues := port.Connector.ArchiveStagingWorktreeV0(ctx, orquestaruntimeworktree.StagingPromotionRequestV0{
		ArchiveRef:   command.ArchiveRef,
		PromotionRef: command.PromotionRef,
		RunRef:       command.RunRef,
		ProjectRef:   command.ProjectRef,
		AppRef:       port.AppRef,
		RepoRef:      port.RepoRef,
		WorktreeRef:  command.WorktreeRef,
		BranchRef:    command.BranchRef,
		ArchiveDir:   port.ArchiveDir,
		WriteSet:     append([]string(nil), command.WriteSet...),
		EvidenceRefs: append([]string(nil), command.EvidenceRefs...),
	})
	return autoprogrammingPromotionEffectFromWorktreeV0(result, issues), nil
}

func autoprogrammingPromotionEffectFromWorktreeV0(
	result orquestaruntimeworktree.StagingPromotionResultV0,
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) orquestaautoprogramming.AutoprogrammingStagingEffectResultV0 {
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
		SchemaVersion:  orquestaautoprogramming.AutoprogrammingStagingPromotionSchemaVersionV0,
		Status:         strings.TrimSpace(result.Status),
		PromotionRef:   result.PromotionRef,
		ArchiveRef:     result.ArchiveRef,
		RunRef:         result.RunRef,
		ProjectRef:     result.ProjectRef,
		WorktreeRef:    result.WorktreeRef,
		BranchRef:      result.BranchRef,
		ChangedPaths:   append([]string(nil), result.ChangedPaths...),
		CommitRef:      result.CommitRef,
		CommitShortRef: result.CommitShortRef,
		Retryable:      result.Retryable,
		EvidenceRefs:   append([]string(nil), result.EvidenceRefs...),
		Issues:         autoprogrammingPromotionIssuesFromWorktreeV0(issues),
	}
}

func autoprogrammingPromotionIssuesFromWorktreeV0(
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) []orquestaautoprogramming.AutoprogrammingRequestIssueV0 {
	out := make([]orquestaautoprogramming.AutoprogrammingRequestIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, orquestaautoprogramming.AutoprogrammingRequestIssueV0{
			Code:    string(issue.Code),
			Field:   issue.Field,
			Message: issue.MessageKey,
		})
	}
	return out
}
