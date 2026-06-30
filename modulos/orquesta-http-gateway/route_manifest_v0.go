package orquestahttpgateway

import (
	"net/http"
	"strings"
)

const (
	RouteManifestKindExactV0  = "exact"
	RouteManifestKindPrefixV0 = "prefix"

	RouteSecurityBrowserHTMLV0          = "browser_html"
	RouteSecurityControlPlaneReadV0     = "control_plane_read"
	RouteSecurityControlPlaneMutationV0 = "control_plane_mutation"
	RouteSecurityMCPJSONRPCV0           = "mcp_jsonrpc"

	RouteRefHomeV0                              = "route-ref-home-v0"
	RouteRefNuevaAppV0                          = "route-ref-nueva-app-v0"
	RouteRefNuevaAppGuideV0                     = "route-ref-nueva-app-guide-v0"
	RouteRefOpsDashboardV0                      = "route-ref-ops-dashboard-v0"
	RouteRefAutoprogrammingPageV0               = "route-ref-autoprogramming-page-v0"
	RouteRefAppChangePageV0                     = "route-ref-app-change-page-v0"
	RouteRefDirectorStatsPageV0                 = "route-ref-director-stats-page-v0"
	RouteRefRunControlPageV0                    = "route-ref-run-control-page-v0"
	RouteRefRunQueuePageV0                      = "route-ref-run-queue-page-v0"
	RouteRefAppSpecV0                           = "route-ref-app-spec-v0"
	RouteRefAppDirectorV0                       = "route-ref-app-director-v0"
	RouteRefAppDirectorPreviewV0                = "route-ref-app-director-preview-v0"
	RouteRefAppDirectorGoalObserveV0            = "route-ref-app-director-goal-observe-v0"
	RouteRefAppIntakeGuidedTurnV0               = "route-ref-app-intake-guided-turn-v0"
	RouteRefAppChangeV0                         = "route-ref-app-change-prefix-v0"
	RouteRefAppVCSV0                            = "route-ref-app-vcs-v0"
	RouteRefDirectorStatsV0                     = "route-ref-director-stats-v0"
	RouteRefRunControlV0                        = "route-ref-run-control-v0"
	RouteRefRuntimeModelsV0                     = "route-ref-runtime-models-v0"
	RouteRefRunQueuePriorityV0                  = "route-ref-run-queue-priority-v0"
	RouteRefQueueGlobalStatusV0                 = "route-ref-queue-global-status-v0"
	RouteRefRunSupervisorV0                     = "route-ref-run-supervisor-v0"
	RouteRefOpsAgentRuntimeDetailV0             = "route-ref-ops-agent-runtime-detail-v0"
	RouteRefOperationalStatusV0                 = "route-ref-operational-status-v0"
	RouteRefFunctionContractListV0              = "route-ref-function-contract-list-v0"
	RouteRefFunctionContractViewV0              = "route-ref-function-contract-view-v0"
	RouteRefServerShutdownV0                    = "route-ref-server-shutdown-v0"
	RouteRefHumanDirectorWorkReviewPlanV0       = "route-ref-human-director-work-review-plan-v0"
	RouteRefAutoprogrammingValidateRequestV0    = "route-ref-autoprogramming-validate-request-v0"
	RouteRefAutoprogrammingSelfImprovementV0    = "route-ref-autoprogramming-self-improvement-v0"
	RouteRefAutoprogrammingPrepareRunV0         = "route-ref-autoprogramming-prepare-run-v0"
	RouteRefAutoprogrammingObserveGoalV0        = "route-ref-autoprogramming-observe-goal-v0"
	RouteRefAutoprogrammingObserveActiveGoalsV0 = "route-ref-autoprogramming-observe-active-goals-v0"
	RouteRefAutoprogrammingStatusV0             = "route-ref-autoprogramming-status-v0"
	RouteRefAutoprogrammingSuperviseV0          = "route-ref-autoprogramming-supervise-v0"
	RouteRefGovernanceCatalogQueryV0            = "route-ref-governance-catalog-query-v0"
	RouteRefDomainWorkV0                        = "route-ref-domain-work-v0"
	RouteRefExternalWorkDryRunV0                = "route-ref-external-work-dry-run-v0"
	RouteRefExternalWorkRunV0                   = "route-ref-external-work-run-v0"
	RouteRefCodebaseQueryV0                     = "route-ref-codebase-query-v0"
	RouteRefCodebaseStatusV0                    = "route-ref-codebase-status-v0"
	RouteRefMCPJSONRPCV0                        = "route-ref-mcp-jsonrpc-v0"
	RouteRefWorkspaceTimelineV0                 = "route-ref-workspace-timeline-v0"
)

const (
	routeAppVCSOverlayV0        = "/api/v0/apps/vcs"
	routeMCPJSONRPCOverlayV0    = "/mcp"
	routeWorkspaceTimelineV0    = "/api/v0/observability/workspace-timeline"
	routeOwnerHTTPGatewayV0     = "orquesta-http-gateway"
	routeOwnerAppGatewayV0      = "orquesta-app-gateway"
	routeOwnerMCPV0             = "orquesta-mcp"
	routeOwnerServerV0          = "cmd/orquesta-server"
	routeMethodAnyV0            = "ANY"
	routeMethodPostV0           = http.MethodPost
	routeMethodGetV0            = http.MethodGet
	routeIssueDuplicateExactV0  = "route_manifest_duplicate_exact"
	routeIssueDuplicatePrefixV0 = "route_manifest_duplicate_prefix"
	routeIssueDuplicateRefV0    = "route_manifest_duplicate_ref"
	routeIssueIncompleteV0      = "route_manifest_incomplete_entry"
	routeIssueUnknownKindV0     = "route_manifest_unknown_kind"
	routeIssueMissingShadowV0   = "route_manifest_prefix_shadow_not_declared"
)

type RouteManifestEntryV0 struct {
	Ref               string
	Pattern           string
	Kind              string
	Owner             string
	Methods           []string
	SecurityProfile   string
	ShadowsPrefixRefs []string
}

type RouteManifestIssueV0 struct {
	Code          string
	RouteRef      string
	OtherRouteRef string
	EvidenceRef   string
}

func PublicRouteManifestV0() []RouteManifestEntryV0 {
	return copyRouteManifestEntriesV0(routeManifestEntriesV0())
}

func ValidateRouteManifestV0(entries []RouteManifestEntryV0) []RouteManifestIssueV0 {
	var issues []RouteManifestIssueV0
	refs := map[string]RouteManifestEntryV0{}
	exacts := map[string]RouteManifestEntryV0{}
	prefixes := map[string]RouteManifestEntryV0{}
	for _, entry := range entries {
		entry.Ref = strings.TrimSpace(entry.Ref)
		entry.Pattern = strings.TrimSpace(entry.Pattern)
		entry.Owner = strings.TrimSpace(entry.Owner)
		entry.SecurityProfile = strings.TrimSpace(entry.SecurityProfile)
		if entry.Ref == "" || entry.Pattern == "" || entry.Owner == "" || entry.SecurityProfile == "" || len(entry.Methods) == 0 {
			issues = append(issues, routeManifestIssueV0(routeIssueIncompleteV0, entry, ""))
			continue
		}
		if previous, ok := refs[entry.Ref]; ok {
			issues = append(issues, routeManifestIssueV0(routeIssueDuplicateRefV0, entry, previous.Ref))
		}
		refs[entry.Ref] = entry
		switch entry.Kind {
		case RouteManifestKindExactV0:
			if previous, ok := exacts[entry.Pattern]; ok {
				issues = append(issues, routeManifestIssueV0(routeIssueDuplicateExactV0, entry, previous.Ref))
			}
			exacts[entry.Pattern] = entry
		case RouteManifestKindPrefixV0:
			if previous, ok := prefixes[entry.Pattern]; ok {
				issues = append(issues, routeManifestIssueV0(routeIssueDuplicatePrefixV0, entry, previous.Ref))
			}
			prefixes[entry.Pattern] = entry
		default:
			issues = append(issues, routeManifestIssueV0(routeIssueUnknownKindV0, entry, ""))
		}
	}
	for _, exact := range exacts {
		for _, prefix := range prefixes {
			if strings.HasPrefix(exact.Pattern, prefix.Pattern) &&
				!containsRouteManifestStringV0(exact.ShadowsPrefixRefs, prefix.Ref) {
				issues = append(issues, routeManifestIssueV0(routeIssueMissingShadowV0, exact, prefix.Ref))
			}
		}
	}
	return issues
}

func RouteManifestEvidenceRefsV0(entries []RouteManifestEntryV0) []string {
	refs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.Ref) == "" || strings.TrimSpace(entry.Owner) == "" {
			continue
		}
		refs = append(refs, entry.Ref+"-owner-"+entry.Owner)
	}
	return refs
}

func routeManifestEntryByRefV0(entries []RouteManifestEntryV0) map[string]RouteManifestEntryV0 {
	out := map[string]RouteManifestEntryV0{}
	for _, entry := range entries {
		out[entry.Ref] = entry
	}
	return out
}

func routeManifestIssueV0(code string, entry RouteManifestEntryV0, otherRef string) RouteManifestIssueV0 {
	return RouteManifestIssueV0{
		Code:          code,
		RouteRef:      entry.Ref,
		OtherRouteRef: otherRef,
		EvidenceRef:   "route-manifest-issue-ref-" + code,
	}
}

func containsRouteManifestStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func copyRouteManifestEntriesV0(entries []RouteManifestEntryV0) []RouteManifestEntryV0 {
	out := make([]RouteManifestEntryV0, len(entries))
	for i, entry := range entries {
		entry.Methods = append([]string{}, entry.Methods...)
		entry.ShadowsPrefixRefs = append([]string{}, entry.ShadowsPrefixRefs...)
		out[i] = entry
	}
	return out
}
