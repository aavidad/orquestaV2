package main

import (
	"context"
	"encoding/json"
	"net/http"

	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

var requiredOPESRegistryFinalPkgFilesV0 = []string{
	"manifest_cierre.json",
	"tema_final.md",
	"tests.json",
	"visuales_plan.md",
	"html/index.html",
	"rag/manifest.json",
	"audio/guion_audio.md",
	"tutor/tutor_prompt.md",
	"qa_final.md",
}

type opesRegistryFinalPkgSummaryV0 struct {
	CourseID                 string                                  `json:"course_id"`
	QueueRef                 string                                  `json:"queue_ref"`
	DryRun                   bool                                    `json:"dry_run"`
	Seen                     int                                     `json:"seen"`
	Submitted                int                                     `json:"submitted"`
	Skipped                  int                                     `json:"skipped"`
	RemainingCandidates      int                                     `json:"remaining_candidates"`
	CompletedNonTerminal     int                                     `json:"completed_nonterminal,omitempty"`
	CompletedNonTerminalRefs []opesRegistryFinalPkgCompletionDriftV0 `json:"completed_nonterminal_refs,omitempty"`
	Reconciled               int                                     `json:"reconciled,omitempty"`
	ReconcileSkipped         int                                     `json:"reconcile_skipped,omitempty"`
	Active                   []string                                `json:"active,omitempty"`
	Results                  []opesRegistryFinalPkgResultV0          `json:"results,omitempty"`
	Errors                   []opesRegistryFinalPkgPublicErrorV0     `json:"errors,omitempty"`
}

type opesRegistryFinalPkgResultV0 struct {
	TopicID string `json:"topic_id"`
	RunRef  string `json:"run_ref,omitempty"`
	Status  string `json:"status"`
}

type opesRegistryFinalPkgCompletionDriftV0 struct {
	RunRef  string `json:"run_ref"`
	TopicID string `json:"topic_id"`
	Status  string `json:"status,omitempty"`
	Reason  string `json:"reason"`
}

type opesRegistryFinalPkgPublicErrorV0 struct {
	TopicID string `json:"topic_id,omitempty"`
	Code    string `json:"code"`
}

type opesRegistryFinalPkgSubmitterV0 func(
	context.Context,
	*http.Client,
	string,
	orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) (string, error)

type opesRegistryFinalPkgAppChangeStateV0 struct {
	Records []struct {
		Request json.RawMessage `json:"request"`
	} `json:"records"`
}

type opesRegistryFinalPkgRegistryV0 struct {
	Courses map[string]struct {
		Topics map[string]opesRegistryFinalPkgTopicV0 `json:"topics"`
	} `json:"courses"`
}

type opesRegistryFinalPkgTopicV0 struct {
	Lock map[string]any `json:"lock"`
}

type opesRegistryFinalPkgRunProjectionV0 struct {
	Status          string
	Deliveries      []string
	ClosedTasks     []string
	AcceptedReviews []string
	Blockers        []string
}

func runOPESRegistryFinalPkgOnceV0(
	ctx context.Context,
	config opesRegistryFinalPkgConfigV0,
	submitter opesRegistryFinalPkgSubmitterV0,
) (opesRegistryFinalPkgSummaryV0, error) {
	if submitter == nil {
		submitter = submitOPESRegistryFinalPkgRunV0
	}
	if err := validateOPESRegistryFinalPkgConfigV0(config); err != nil {
		return opesRegistryFinalPkgSummaryV0{}, err
	}
	registry, err := readOPESRegistryFinalPkgRegistryV0(config.RegistryPath)
	if err != nil {
		return opesRegistryFinalPkgSummaryV0{}, err
	}
	state, err := readOPESRegistryFinalPkgAppChangeStateV0(config.AppChangeState)
	if err != nil {
		return opesRegistryFinalPkgSummaryV0{}, err
	}
	template, err := findOPESRegistryFinalPkgTemplateRequestV0(state, config.TemplateRunRef)
	if err != nil {
		return opesRegistryFinalPkgSummaryV0{}, err
	}
	existing, err := collectOPESRegistryFinalPkgExistingRunRefsV0(state)
	if err != nil {
		return opesRegistryFinalPkgSummaryV0{}, err
	}
	active := collectOPESRegistryFinalPkgActiveRunRefsV0(config.OrchestrationRuns)
	activeFinalPkg, completedNonTerminal := inspectOPESRegistryFinalPkgRunRefsV0(config, active)
	summary := opesRegistryFinalPkgSummaryV0{
		CourseID:                 config.CourseID,
		QueueRef:                 config.QueueRef,
		DryRun:                   config.DryRun,
		Active:                   activeFinalPkg,
		CompletedNonTerminal:     len(completedNonTerminal),
		CompletedNonTerminalRefs: completedNonTerminal,
	}
	reconcileOPESRegistryFinalPkgCompletedRunsV0(ctx, config, completedNonTerminal, &summary)
	if len(activeFinalPkg) >= config.MaxInFlight {
		summary.Results = append(summary.Results, opesRegistryFinalPkgResultV0{Status: "max_in_flight"})
		return summary, nil
	}
	candidates, err := selectOPESRegistryFinalPkgCandidatesV0(registry, config, existing, active)
	if err != nil {
		return opesRegistryFinalPkgSummaryV0{}, err
	}
	summary.Seen = len(candidates)
	capacity := config.BatchSize
	if remainingSlots := config.MaxInFlight - len(activeFinalPkg); remainingSlots < capacity {
		capacity = remainingSlots
	}
	if capacity < 0 {
		capacity = 0
	}
	selected := candidates
	if len(selected) > capacity {
		selected = selected[:capacity]
	}
	summary.RemainingCandidates = len(candidates) - len(selected)
	client := &http.Client{Timeout: config.HTTPTimeout}
	for _, topicID := range selected {
		request, err := buildOPESRegistryFinalPkgRequestV0(config, template, topicID)
		if err != nil {
			summary.Skipped++
			summary.Errors = append(summary.Errors, opesRegistryFinalPkgPublicErrorV0{TopicID: topicID, Code: err.Error()})
			summary.Results = append(summary.Results, opesRegistryFinalPkgResultV0{TopicID: topicID, Status: "build_error"})
			continue
		}
		if config.DryRun {
			summary.Results = append(summary.Results, opesRegistryFinalPkgResultV0{
				TopicID: topicID,
				RunRef:  request.RunRef,
				Status:  "dry_run",
			})
			continue
		}
		runRef, err := submitter(ctx, client, config.OrquestaBaseURL, request)
		if err != nil {
			summary.Skipped++
			summary.Errors = append(summary.Errors, opesRegistryFinalPkgPublicErrorV0{TopicID: topicID, Code: err.Error()})
			summary.Results = append(summary.Results, opesRegistryFinalPkgResultV0{
				TopicID: topicID,
				RunRef:  request.RunRef,
				Status:  "submit_error",
			})
			continue
		}
		summary.Submitted++
		summary.Results = append(summary.Results, opesRegistryFinalPkgResultV0{
			TopicID: topicID,
			RunRef:  runRef,
			Status:  "submitted",
		})
	}
	if len(candidates) == 0 && len(summary.Results) == 0 {
		summary.Results = []opesRegistryFinalPkgResultV0{{Status: "no_candidates"}}
	}
	return summary, nil
}

func submitOPESRegistryFinalPkgRunV0(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
) (string, error) {
	return submitOPESExternalWorkRunV0(ctx, client, baseURL, request)
}
