package orquestaruntime

import (
	"fmt"
	"regexp"
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func (v *runtimeLaunchRequestValidatorV0) requireOpaque(field, value string, missingCode RuntimeLaunchErrorCodeV0) {
	if value == "" {
		v.add(missingCode, field)
		return
	}
	v.optionalOpaque(field, value)
}

func (v *runtimeLaunchRequestValidatorV0) requireProviderRef(field, value string) {
	v.requireOpaqueWithoutConcrete(field, value, ReferenciaNoOpacaV0, looksLikeConcreteProviderValueV0)
}

func (v *runtimeLaunchRequestValidatorV0) requireModelRef(field, value string, missingCode RuntimeLaunchErrorCodeV0) {
	v.requireOpaqueWithoutConcrete(field, value, missingCode, looksLikeConcreteModelValueV0)
}

func (v *runtimeLaunchRequestValidatorV0) requireOpaqueWithoutConcrete(
	field string,
	value string,
	missingCode RuntimeLaunchErrorCodeV0,
	looksConcrete func(string) bool,
) {
	if value == "" {
		v.add(missingCode, field)
		return
	}
	switch {
	case looksLikeSecret(value):
		v.add(SecretoDetectadoV0, field)
	case looksConcrete(value):
		v.add(ReferenciaNoOpacaV0, field)
	case !opaqueRefPatternV0.MatchString(value):
		v.add(ReferenciaNoOpacaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) optionalOpaque(field, value string) {
	if value == "" {
		return
	}
	switch {
	case looksLikeSecret(value):
		v.add(SecretoDetectadoV0, field)
	case !opaqueRefPatternV0.MatchString(value):
		v.add(ReferenciaNoOpacaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireHomeRef(field, value string) {
	if value == "" {
		v.add(HomeRefRequeridoV0, field)
		return
	}
	switch {
	case looksLikeSecret(value):
		v.add(SecretoDetectadoV0, field)
	case looksLikeHomePath(value):
		v.add(RutaHomeRealDetectadaV0, field)
	case !opaqueRefPatternV0.MatchString(value):
		v.add(ReferenciaNoOpacaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireRelativePath(field, value string, code RuntimeLaunchErrorCodeV0) {
	if !isRelativeContractPath(value) {
		v.add(code, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireConst(field, got, want string, code RuntimeLaunchErrorCodeV0) {
	if got != want {
		v.add(code, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireUTCTime(field, value string) {
	if !utcInstantPatternV0.MatchString(value) {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireLocale(field, value string) {
	if !localePatternV0.MatchString(value) {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requirePositiveWindow(field string, value int) {
	if value < 1 || value > 3600 {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireNonEmpty(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
	}
}

func (v *runtimeLaunchRequestValidatorV0) requireNonEmptyList(field string, values []string) {
	if len(values) == 0 {
		v.add(RuntimeLaunchRequestInvalidaV0, field)
		return
	}
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			v.add(RuntimeLaunchRequestInvalidaV0, fmt.Sprintf("%s[%d]", field, i))
		}
	}
}

func (v *runtimeLaunchRequestValidatorV0) add(code RuntimeLaunchErrorCodeV0, field string) {
	v.errors = append(v.errors, RuntimeLaunchErrorV0{
		Code:          code,
		MessageKey:    "orquesta.runtime.launch." + string(code),
		Field:         field,
		Retryable:     code == RuntimeNoDisponibleV0,
		CorrelationID: v.correlationID,
	})
}

func isRelativeContractPath(value string) bool {
	if !relativePathAllowedPatternV0.MatchString(value) {
		return false
	}
	return orquestarails.WorkspaceRelativePathAllowedV0(value, false)
}

func looksLikeHomePath(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(value)
	return strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "~") ||
		strings.Contains(value, "/") ||
		strings.Contains(value, "\\") ||
		strings.Contains(low, "$home") ||
		(len(value) >= 2 && isASCIIAlpha(value[0]) && value[1] == ':')
}

func looksLikeSecret(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(low, "bearer") ||
		strings.HasPrefix(low, "sk-") ||
		strings.HasPrefix(low, "ghp_") ||
		strings.HasPrefix(low, "xoxb-") ||
		strings.Contains(low, "access_token") ||
		strings.Contains(low, "refresh_token") ||
		strings.Contains(low, "api_key") ||
		strings.Contains(low, "secret=") ||
		strings.Count(value, ".") == 2 && strings.HasPrefix(value, "eyJ")
}

func looksLikeConcreteProviderValueV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "openai") ||
		strings.Contains(low, "anthropic") ||
		strings.Contains(low, "gemini") ||
		strings.Contains(low, "mistral") ||
		strings.Contains(low, "bedrock") ||
		strings.Contains(low, "openrouter")
}

func looksLikeConcreteModelValueV0(value string) bool {
	if !orquestarails.DetailProhibitedRailsEnabledV0() {
		return false
	}
	low := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(low, "gpt-") ||
		strings.Contains(low, "claude") ||
		strings.Contains(low, "gemini") ||
		strings.Contains(low, "llama") ||
		strings.Contains(low, "mistral") ||
		strings.HasPrefix(low, "o1") ||
		strings.HasPrefix(low, "o3") ||
		strings.HasPrefix(low, "o4-")
}

func isOneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func isASCIIAlpha(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func containsParentTraversal(value string) bool {
	return value == ".." ||
		strings.HasPrefix(value, "../") ||
		strings.HasSuffix(value, "/..") ||
		strings.Contains(value, "/../")
}

func contextBundleMatchesContractModuleV0(bundleTarget string, contract RuntimeFunctionContractV0) bool {
	module := moduleFromRuntimeContractPathV0(contract.ArchivoObjetivo)
	return module == "" || module == bundleTarget
}

func moduleFromRuntimeContractPathV0(path string) string {
	const prefix = "modulos/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	index := strings.IndexByte(rest, '/')
	if index <= 0 {
		return ""
	}
	return rest[:index]
}

var (
	opaqueRefPatternV0           = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,511}$`)
	relativePathAllowedPatternV0 = regexp.MustCompile(`^[A-Za-z0-9._+={}/,@-]{1,240}$`)
	localePatternV0              = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	utcInstantPatternV0          = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`)
)
