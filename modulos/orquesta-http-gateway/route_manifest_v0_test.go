package orquestahttpgateway

import (
	"strings"
	"testing"
)

func TestPublicRouteManifestV0DeclaraInventarioYPrecedenciaV0(t *testing.T) {
	entries := PublicRouteManifestV0()
	if issues := ValidateRouteManifestV0(entries); len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	byRef := routeManifestEntryByRefV0(entries)
	for _, ref := range []string{
		RouteRefAppChangeV0,
		RouteRefAppSpecV0,
		RouteRefAppDirectorV0,
		RouteRefAppDirectorPreviewV0,
		RouteRefAppDirectorGoalObserveV0,
		RouteRefAppIntakeGuidedTurnV0,
		RouteRefAppIntakeWizardBotV0,
		RouteRefAutoprogrammingObserveGoalV0,
		RouteRefAutoprogrammingObserveActiveGoalsV0,
		RouteRefAppVCSV0,
		RouteRefMCPJSONRPCV0,
		RouteRefWorkspaceTimelineV0,
	} {
		if byRef[ref].Ref == "" {
			t.Fatalf("manifest missing %s", ref)
		}
	}
	if byRef[RouteRefAppChangeV0].Kind != RouteManifestKindPrefixV0 {
		t.Fatalf("app-change kind=%s", byRef[RouteRefAppChangeV0].Kind)
	}
	for _, ref := range []string{RouteRefAppSpecV0, RouteRefAppDirectorV0, RouteRefAppDirectorPreviewV0, RouteRefAppDirectorGoalObserveV0, RouteRefAppIntakeGuidedTurnV0, RouteRefAppIntakeWizardBotV0, RouteRefAppVCSV0} {
		if !containsRouteManifestStringV0(byRef[ref].ShadowsPrefixRefs, RouteRefAppChangeV0) {
			t.Fatalf("%s no declara shadow de %s", ref, RouteRefAppChangeV0)
		}
	}
	if byRef[RouteRefAppDirectorPreviewV0].SecurityProfile != RouteSecurityControlPlaneReadV0 {
		t.Fatalf("preview security=%s", byRef[RouteRefAppDirectorPreviewV0].SecurityProfile)
	}
}

func TestPublicRouteManifestV0DeclaraContratosE3InternosV0(t *testing.T) {
	byRef := routeManifestEntryByRefV0(PublicRouteManifestV0())
	cases := []struct {
		ref       string
		contract  string
		pageRoute bool
	}{
		{ref: RouteRefNuevaAppV0, contract: RouteContractNuevaAppSolicitarV0, pageRoute: true},
		{ref: RouteRefNuevaAppV0, contract: RouteContractNuevaAppWizardV0, pageRoute: true},
		{ref: RouteRefNuevaAppV0, contract: RouteContractNuevaAppWizardBotV0, pageRoute: true},
		{ref: RouteRefAppSpecV0, contract: RouteContractNuevaAppSolicitarV0},
		{ref: RouteRefAppIntakeGuidedTurnV0, contract: RouteContractNuevaAppWizardV0},
		{ref: RouteRefAppIntakeWizardBotV0, contract: RouteContractNuevaAppWizardBotV0},
		{ref: RouteRefAutoprogrammingPageV0, contract: RouteContractAutoprogrammingPrepareRunV0, pageRoute: true},
		{ref: RouteRefAutoprogrammingPageV0, contract: RouteContractAutoprogrammingStatusV0, pageRoute: true},
		{ref: RouteRefAutoprogrammingPrepareRunV0, contract: RouteContractAutoprogrammingPrepareRunV0},
		{ref: RouteRefAutoprogrammingStatusV0, contract: RouteContractAutoprogrammingStatusV0},
		{ref: RouteRefHumanDirectorWorkReviewPlanV0, contract: RouteContractOperatorDirectorReviewPlanV0},
	}
	for _, tc := range cases {
		entry := byRef[tc.ref]
		if entry.Ref == "" {
			t.Fatalf("ruta E3 sin manifest: %s", tc.ref)
		}
		if !containsRouteManifestStringV0(entry.ContractRefs, tc.contract) {
			t.Fatalf("ruta E3 %s no declara contrato %s: %+v", tc.ref, tc.contract, entry.ContractRefs)
		}
		if !tc.pageRoute && !containsRouteManifestStringV0(entry.Methods, routeMethodPostV0) {
			t.Fatalf("ruta E3 API %s no declara POST: %+v", tc.ref, entry.Methods)
		}
	}
}

func TestValidateRouteManifestV0DetectaColisionesYShadowsNoDeclarados(t *testing.T) {
	entries := []RouteManifestEntryV0{
		manifestEntryForTestV0("route-ref-prefix", "/api/v0/apps/", RouteManifestKindPrefixV0, "owner-a"),
		manifestEntryForTestV0("route-ref-exact", "/api/v0/apps/vcs", RouteManifestKindExactV0, "owner-b"),
		manifestEntryForTestV0("route-ref-dup", "/api/v0/apps/vcs", RouteManifestKindExactV0, "owner-c"),
	}

	issues := ValidateRouteManifestV0(entries)

	if !hasRouteManifestIssueV0(issues, routeIssueDuplicateExactV0) {
		t.Fatalf("missing duplicate issue: %+v", issues)
	}
	if !hasRouteManifestIssueV0(issues, routeIssueMissingShadowV0) {
		t.Fatalf("missing shadow issue: %+v", issues)
	}
}

func TestValidateRouteManifestV0DetectaRefsDuplicadasYEntradasInvalidas(t *testing.T) {
	entries := []RouteManifestEntryV0{
		manifestEntryForTestV0("route-ref-dup", "/api/v0/dup-a", RouteManifestKindExactV0, "owner-a"),
		manifestEntryForTestV0("route-ref-dup", "/api/v0/dup-b", RouteManifestKindExactV0, "owner-b"),
		{Ref: "route-ref-incomplete", Pattern: "/api/v0/incomplete", Kind: RouteManifestKindExactV0},
		manifestEntryForTestV0("route-ref-unknown-kind", "/api/v0/unknown", "wildcard", "owner-c"),
	}

	issues := ValidateRouteManifestV0(entries)

	for _, code := range []string{
		routeIssueDuplicateRefV0,
		routeIssueIncompleteV0,
		routeIssueUnknownKindV0,
	} {
		if !hasRouteManifestIssueV0(issues, code) {
			t.Fatalf("missing %s issue: %+v", code, issues)
		}
	}
}

func TestGatewayRouteRegistrationsV0EstanEnManifestV0(t *testing.T) {
	registrations := gatewayRouteRegistrationsV0(RouteHandlersV0{})

	if issue := validateGatewayRouteManifestPublicV0(); issue.Code != "" {
		t.Fatalf("public manifest issue=%+v", issue)
	}
	if issue := validateGatewayRouteRegistrationsV0(registrations); issue.Code != "" {
		t.Fatalf("issue=%+v", issue)
	}
}

func TestGatewayRouteRegistrationsV0DetectaRegistroDuplicadoYDesalineado(t *testing.T) {
	registrations := []gatewayRouteRegistrationV0{
		{ref: RouteRefAppSpecV0, route: RouteAppSpecV0},
		{ref: RouteRefAppDirectorV0, route: RouteAppSpecV0},
	}
	if issue := validateGatewayRouteRegistrationsV0(registrations); issue.Code != routeIssueRegistrationDuplicateV0 {
		t.Fatalf("duplicate issue=%+v", issue)
	}

	registrations = []gatewayRouteRegistrationV0{
		{ref: RouteRefAppSpecV0, route: RouteAppDirectorV0},
	}
	if issue := validateGatewayRouteRegistrationsV0(registrations); issue.Code != routeIssueRegistrationMismatchV0 {
		t.Fatalf("mismatch issue=%+v", issue)
	}
}

func TestRouteManifestEvidenceRefsV0SonCompactasV0(t *testing.T) {
	refs := RouteManifestEvidenceRefsV0(PublicRouteManifestV0())
	if len(refs) == 0 {
		t.Fatal("missing evidence refs")
	}
	for _, ref := range refs {
		if strings.Contains(ref, "http://") || strings.Contains(ref, "?") {
			t.Fatalf("evidence ref no compacta=%q", ref)
		}
	}
}

func hasRouteManifestIssueV0(issues []RouteManifestIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func manifestEntryForTestV0(ref string, pattern string, kind string, owner string) RouteManifestEntryV0 {
	return RouteManifestEntryV0{
		Ref:             ref,
		Pattern:         pattern,
		Kind:            kind,
		Owner:           owner,
		Methods:         []string{routeMethodPostV0},
		SecurityProfile: RouteSecurityControlPlaneMutationV0,
	}
}
