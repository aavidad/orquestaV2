package orquestahttpgateway

const (
	routeIssueRegistrationDuplicateV0 = "route_registration_duplicate"
	routeIssueRegistrationMissingV0   = "route_registration_missing_manifest"
	routeIssueRegistrationMismatchV0  = "route_registration_manifest_mismatch"
)

func validateGatewayRouteManifestPublicV0() RouteManifestIssueV0 {
	issues := ValidateRouteManifestV0(PublicRouteManifestV0())
	if len(issues) == 0 {
		return RouteManifestIssueV0{}
	}
	return issues[0]
}

func validateGatewayRouteRegistrationsV0(registrations []gatewayRouteRegistrationV0) RouteManifestIssueV0 {
	manifest := routeManifestEntryByRefV0(PublicRouteManifestV0())
	seen := map[string]string{}
	for _, registration := range registrations {
		if previousRef, ok := seen[registration.route]; ok {
			return RouteManifestIssueV0{
				Code:          routeIssueRegistrationDuplicateV0,
				RouteRef:      registration.ref,
				OtherRouteRef: previousRef,
			}
		}
		seen[registration.route] = registration.ref
		entry, ok := manifest[registration.ref]
		if !ok {
			return RouteManifestIssueV0{Code: routeIssueRegistrationMissingV0, RouteRef: registration.ref}
		}
		if entry.Pattern != registration.route {
			return RouteManifestIssueV0{Code: routeIssueRegistrationMismatchV0, RouteRef: registration.ref}
		}
	}
	return RouteManifestIssueV0{}
}
