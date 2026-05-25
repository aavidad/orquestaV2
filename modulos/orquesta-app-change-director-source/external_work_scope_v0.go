package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

func appChangeTaskWriteSetV0(request orquestaappchange.AppChangeRequestV0) []string {
	if len(request.AllowedWriteSet) > 0 {
		return append([]string(nil), request.AllowedWriteSet...)
	}
	if !appChangeHasExternalWorkV0(request) {
		return nil
	}
	return appChangeExternalWorkScopesV0(request.ExternalWork)
}

func appChangeExternalWorkScopesV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) []string {
	if work == nil {
		return nil
	}
	project := appChangeScopePartV0(work.ProjectRef)
	if project == "" {
		project = "external"
	}
	scopes := []string{}
	if kind := appChangeScopePartV0(work.WorkKind); kind != "" {
		scopes = append(scopes, "external/"+project+"/"+kind)
	}
	if job := appChangeScopePartV0(work.JobRef); job != "" {
		scopes = append(scopes, "external/"+project+"/"+job)
	}
	for _, ref := range work.WorkRefs {
		part := appChangeScopePartV0(ref)
		if part == "" {
			continue
		}
		scopes = append(scopes, "external/"+project+"/"+part)
	}
	return compactAppChangeSourceRefsV0(scopes)
}

func appChangeScopePartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(
		" ", "-",
		"\t", "-",
		"\n", "-",
		"\r", "-",
		"/", "-",
		"\\", "-",
	).Replace(value)
	return strings.Trim(value, "-")
}
