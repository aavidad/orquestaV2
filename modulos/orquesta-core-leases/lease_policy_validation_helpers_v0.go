package orquestacoreleases

import (
	"regexp"
	"strconv"

	orquestarails "orquesta/modulos/orquesta-rails"
)

func validAgentLeaseUTCInstantV0(value string) bool {
	if !agentLeaseUTCInstantPatternV0.MatchString(value) {
		return false
	}
	year, ok := atoiAgentLeaseV0(value[0:4])
	if !ok {
		return false
	}
	month, ok := atoiAgentLeaseV0(value[5:7])
	if !ok || month < 1 || month > 12 {
		return false
	}
	day, ok := atoiAgentLeaseV0(value[8:10])
	if !ok || day < 1 || day > daysInAgentLeaseMonthV0(year, month) {
		return false
	}
	hour, ok := atoiAgentLeaseV0(value[11:13])
	if !ok || hour > 23 {
		return false
	}
	minute, ok := atoiAgentLeaseV0(value[14:16])
	if !ok || minute > 59 {
		return false
	}
	second, ok := atoiAgentLeaseV0(value[17:19])
	return ok && second <= 59
}

func containsForbiddenAgentLeaseDetailV0(value string) bool {
	return orquestarails.TextContainsOperationalRawDetailForFieldV0(
		"agent_lease",
		"ref",
		value,
	)
}

func isOneOfAgentLeaseV0(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func atoiAgentLeaseV0(value string) (int, bool) {
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func daysInAgentLeaseMonthV0(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isAgentLeaseLeapYearV0(year) {
			return 29
		}
		return 28
	default:
		return 31
	}
}

func isAgentLeaseLeapYearV0(year int) bool {
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}

var (
	agentLeaseOpaqueRefPatternV0  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{2,511}$`)
	agentLeaseUTCInstantPatternV0 = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$`)
	forbiddenAgentLeaseJSONKeysV0 = map[string]bool{
		"access_token": true, "api_key": true, "authorization": true,
		"client_secret": true, "db": true, "database": true,
		"home": true, "model": true, "modelo": true,
		"oauth": true, "password": true, "pid": true,
		"process": true, "process_ref": true, "provider": true,
		"proveedor": true, "prompt": true, "raw_prompt": true,
		"raw_text": true, "raw_transcript": true, "runtime": true,
		"runtime_provider": true, "secrets": true, "transcript": true,
	}
)
