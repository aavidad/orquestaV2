package orquestarails

import (
	"os"
	"strings"
)

const DetailProhibitedRailsScopeEnvV0 = "ORQUESTA_DETAIL_PROHIBITED_RAILS_SCOPE"

// OperationalRawDetailFragmentsV0 is owned by orquesta-rails for raw content
// boundaries. It avoids generic words such as prompt, transcript, provider or
// model unless they carry value-like separators.
var OperationalRawDetailFragmentsV0 = []string{
	"sk-",
	"postgres://",
	"mysql://",
	"sqlite://",
	"mongodb://",
	"redis://",
	"dsn=",
	"database_url=",
	"/home/",
	"\\home\\",
	"/users/",
	"\\users\\",
	"c:\\users\\",
	"$home",
	"~/",
	"prompt=",
	"process=",
	"process_ref=",
	"pid=",
	"completion=",
	"transcript=",
	"raw_prompt=",
	"raw_transcript=",
	"raw_text=",
	"full_text=",
	"secret-token",
}

func ValuesContainOperationalSensitiveDetailForFieldV0(
	boundary string,
	field string,
	values []string,
) bool {
	for _, value := range values {
		if TextContainsOperationalSensitiveDetailForFieldV0(boundary, field, value) {
			return true
		}
	}
	return false
}

func TextContainsOperationalSensitiveDetailForFieldV0(boundary string, field string, value string) bool {
	return textContainsOperationalSensitiveDetailIgnoringEnvV0(value)
}

func TextContainsOperationalRawDetailForFieldV0(boundary string, field string, value string) bool {
	if textContainsOperationalSensitiveDetailIgnoringEnvV0(value) {
		return true
	}
	if !DetailProhibitedRailsEnabledForFieldV0(boundary, field) {
		return false
	}
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	for _, fragment := range OperationalRawDetailFragmentsV0 {
		if ContainsFragmentWithBoundaryV0(lower, fragment) {
			return true
		}
	}
	return false
}

func DetailProhibitedRailsEnabledForFieldV0(boundary string, field string) bool {
	if !DetailProhibitedRailsEnabledV0() {
		return false
	}
	return detailRailsScopeAllowsV0(os.Getenv(DetailProhibitedRailsScopeEnvV0), boundary, field)
}

func detailRailsScopeAllowsV0(scope string, boundary string, field string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return true
	}
	boundary = detailRailsScopePartV0(boundary)
	field = detailRailsScopePartV0(field)
	for _, item := range strings.FieldsFunc(scope, detailRailsScopeSeparatorV0) {
		if detailRailsScopeItemAllowsV0(item, boundary, field) {
			return true
		}
	}
	return false
}

func detailRailsScopeItemAllowsV0(item string, boundary string, field string) bool {
	item = strings.ToLower(strings.TrimSpace(item))
	switch item {
	case "", "off":
		return false
	case "*", "*.*", "all":
		return true
	}
	if item == boundary || item == boundary+".*" {
		return true
	}
	if field != "" && (item == "*."+field || item == boundary+"."+field) {
		return true
	}
	return false
}

func detailRailsScopePartV0(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func detailRailsScopeSeparatorV0(r rune) bool {
	return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
}
