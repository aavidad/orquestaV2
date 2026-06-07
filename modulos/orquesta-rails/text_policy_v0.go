package orquestarails

import (
	"os"
	"strings"
)

const DetailProhibitedRailsEnvV0 = "ORQUESTA_DETAIL_PROHIBITED_RAILS"

// OperationalSensitiveFragmentsV0 is the shared permissive rail for text fields.
// It intentionally avoids generic operational words such as runtime, provider,
// model, git, db or sql. Those are normal opaque refs in Orquesta.
var OperationalSensitiveFragmentsV0 = []string{
	"api_key=",
	"api-key=",
	"api_key:",
	"api-key:",
	"access_token=",
	"access-token=",
	"access_token:",
	"access-token:",
	"refresh_token=",
	"refresh-token=",
	"refresh_token:",
	"refresh-token:",
	"client_secret=",
	"client-secret=",
	"client_secret:",
	"client-secret:",
	"authorization:",
	"bearer ",
	"sk-",
	"password=",
	"password:",
	"passwd=",
	"passwd:",
	"pwd=",
	"pwd:",
	"secret=",
	"secret:",
	"secreto=",
	"secreto:",
	"credential=",
	"credential:",
	"credencial=",
	"credencial:",
	"-----begin ",
	"prompt=",
	"completion=",
	"transcript=",
	"raw_prompt=",
	"raw_transcript=",
	"raw_text=",
	"full_text=",
	"dsn=",
	"database_url=",
}

// OperationalDetailMarkersV0 centralizes legacy strict rail markers that are
// kept only for opt-in review/automejora. With detail rails disabled they never
// block runtime flow.
var OperationalDetailMarkersV0 = []string{
	"/home/",
	"/users/",
	`c:\users\`,
	"$home",
	"~/",
	"home=",
	"code_home",
	"codex_home",
	"access_token",
	"refresh_token",
	"bearer ",
	"oauth",
	"token",
	"secret",
	"secreto",
	"password",
	"credential",
	"credencial",
	"api_key",
	"apikey",
	"postgres",
	"sqlite",
	"provider=",
	"model=",
	"begin private key",
	"transcript",
	"prompt=",
	"completion=",
}

func ValuesContainOperationalSensitiveDetailV0(values []string) bool {
	for _, value := range values {
		if TextContainsOperationalSensitiveDetailV0(value) {
			return true
		}
	}
	return false
}

func TextContainsOperationalDetailMarkerV0(value string) bool {
	if !DetailProhibitedRailsEnabledForFieldV0("*", "*") {
		return false
	}
	return textContainsOperationalDetailMarkerIgnoringEnvV0(value)
}

func textContainsOperationalDetailMarkerIgnoringEnvV0(value string) bool {
	lower := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	for _, marker := range OperationalDetailMarkersV0 {
		if strings.Contains(lower, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func BytesContainOperationalDetailMarkerV0(data []byte) bool {
	return TextContainsOperationalDetailMarkerV0(string(data))
}

func TextContainsOperationalSensitiveDetailV0(value string) bool {
	return textContainsOperationalSensitiveDetailIgnoringEnvV0(value)
}

func textContainsOperationalSensitiveDetailIgnoringEnvV0(value string) bool {
	lower := strings.ToLower(value)
	for _, fragment := range OperationalSensitiveFragmentsV0 {
		if ContainsFragmentWithBoundaryV0(lower, fragment) {
			return true
		}
	}
	return textContainsCredentialDSNV0(lower)
}

func DetailProhibitedRailsEnabledV0() bool {
	if !RailsEnforcedV0() {
		return false
	}
	if SecurityModeProgrammingEnabledV0() {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(DetailProhibitedRailsEnvV0))) {
	case "1", "on", "true", "enabled", "enable", "strict":
		return true
	default:
		return false
	}
}

func ContainsFragmentWithBoundaryV0(lowerValue string, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if fragment == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(lowerValue[start:], fragment)
		if index < 0 {
			return false
		}
		absolute := start + index
		if hasTokenBoundaryForFragmentV0(lowerValue, fragment, absolute, absolute+len(fragment)) {
			return true
		}
		start = absolute + len(fragment)
	}
}

func hasTokenBoundaryForFragmentV0(value string, fragment string, start int, end int) bool {
	before := true
	if len(fragment) > 0 && isASCIIAlnumV0(fragment[0]) {
		before = start == 0 || !isASCIIAlnumV0(value[start-1])
	}
	after := true
	if len(fragment) > 0 && isASCIIAlnumV0(fragment[len(fragment)-1]) {
		after = end >= len(value) || !isASCIIAlnumV0(value[end])
	}
	return before && after
}

func isASCIIAlnumV0(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
}

func textContainsCredentialDSNV0(lower string) bool {
	for _, scheme := range []string{
		"postgres://",
		"postgresql://",
		"mysql://",
		"mongodb://",
		"redis://",
	} {
		start := 0
		for {
			index := strings.Index(lower[start:], scheme)
			if index < 0 {
				break
			}
			absolute := start + index + len(scheme)
			end := strings.IndexAny(lower[absolute:], "/?# \t\r\n")
			authority := lower[absolute:]
			if end >= 0 {
				authority = lower[absolute : absolute+end]
			}
			if at := strings.LastIndex(authority, "@"); at > 0 && strings.Contains(authority[:at], ":") {
				return true
			}
			start = absolute
		}
	}
	return false
}
