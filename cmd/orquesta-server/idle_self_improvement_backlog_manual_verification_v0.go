package main

import "strings"

func idleSelfImprovementBacklogManualVerificationCriteriaV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	out := make([]string, 0, len(section.ManualVerifications))
	for _, verification := range section.ManualVerifications {
		out = append(out, "verificacion manual requerida: "+strings.TrimSpace(verification))
	}
	return compactServerStackStringsV0(out)
}

func idleSelfImprovementBacklogManualVerificationContextRefsV0(
	section idleSelfImprovementBacklogSectionV0,
) []string {
	out := make([]string, 0, len(section.ManualVerifications))
	for _, verification := range section.ManualVerifications {
		out = append(out, "backlog_manual_verification:"+strings.TrimSpace(verification))
	}
	return compactServerStackStringsV0(out)
}
