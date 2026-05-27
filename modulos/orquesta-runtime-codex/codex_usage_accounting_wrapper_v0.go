package orquestaruntimecodex

import "strings"

func codexUsageAccountingShellFunctionV0(profile CodexConnectorProfileV0) string {
	stdoutPath := shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexStdoutFileNameV0)
	stderrPath := shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexStderrFileNameV0)
	reportPath := shellQuoteV0(profile.RuntimeWorkDir + "/" + CodexUsageAccountingFileNameV0)
	tempPath := shellQuoteV0(profile.RuntimeWorkDir + "/." + CodexUsageAccountingFileNameV0 + ".tmp")
	var b strings.Builder
	b.WriteString("orquesta_codex_write_usage_accounting_v0() {\n")
	b.WriteString("  awk '\n")
	b.WriteString(codexUsageAccountingAwkProgramV0())
	b.WriteString("  ' ")
	b.WriteString(stdoutPath)
	b.WriteString(" ")
	b.WriteString(stderrPath)
	b.WriteString(" > ")
	b.WriteString(tempPath)
	b.WriteString("\n")
	b.WriteString("  if [ -s ")
	b.WriteString(tempPath)
	b.WriteString(" ]; then mv ")
	b.WriteString(tempPath)
	b.WriteString(" ")
	b.WriteString(reportPath)
	b.WriteString("; else : > ")
	b.WriteString(tempPath)
	b.WriteString("; fi\n")
	b.WriteString("}\n")
	return b.String()
}

func codexUsageAccountingAwkProgramV0() string {
	return `
function digits(value, clean) {
  clean = value
  gsub(/[^0-9]/, "", clean)
  if (clean == "") return 0
  return clean + 0
}
function metric(line) {
  sub(/^.*(input_tokens|prompt_tokens|output_tokens|completion_tokens|total_tokens|input tokens|prompt tokens|output tokens|completion tokens|total tokens|quota remaining|remaining tokens|quota limit|limit tokens)[^0-9]*/, "", line)
  return digits(line)
}
function apply_status(value) {
  if (value == "" || value == "not_configured") return
  if (value == "rate_limited") value = "limited"
  if (status == "" || status == "not_configured") {
    status = value
    return
  }
  if (status == "exhausted" || value == "exhausted") status = "exhausted"
  else if (status == "limited" || value == "limited") status = "limited"
  else if (status == "available" || value == "available") status = "available"
  else status = "unknown"
}
function comma() {
  if (first == 0) printf(",")
  first = 0
}
BEGIN {
  status = ""
  input = 0
  output = 0
  total = 0
  remaining = 0
  limit = 0
  expect_total = 0
}
{
  low = tolower($0)
  if (expect_total && low ~ /^[[:space:]]*[0-9][0-9,._ ]*[[:space:]]*$/) {
    total = digits(low)
  }
  expect_total = 0
  if (low ~ /usage limit reached|quota exceeded/) apply_status("exhausted")
  else if (low ~ /rate limit exceeded|rate limit|quota status:[[:space:]]*limited|quota_status[^a-z0-9_]*limited/) apply_status("limited")
  else if (low ~ /quota status:[[:space:]]*available|quota_status[^a-z0-9_]*available/) apply_status("available")
  else if (low ~ /quota status:[[:space:]]*unknown|quota_status[^a-z0-9_]*unknown/) apply_status("unknown")
  if (low ~ /input_tokens|prompt_tokens|input tokens|prompt tokens/) {
    value = metric(low)
    if (value > 0) input = value
  }
  if (low ~ /output_tokens|completion_tokens|output tokens|completion tokens/) {
    value = metric(low)
    if (value > 0) output = value
  }
  if (low ~ /total_tokens|total tokens/) {
    value = metric(low)
    if (value > 0) total = value
  }
  if (low ~ /tokens used/) {
    value = low
    sub(/^.*tokens used[^0-9]*/, "", value)
    value = digits(value)
    if (value > 0) total = value
    else expect_total = 1
  }
  if (low ~ /quota remaining|remaining tokens/) {
    value = metric(low)
    if (value > 0) remaining = value
  }
  if (low ~ /quota limit|limit tokens/) {
    value = metric(low)
    if (value > 0) limit = value
  }
}
END {
  if (total == 0 && input + output > 0) total = input + output
  if (status == "") status = "unknown"
  printf("{\"usage\":{")
  first = 1
  if (input > 0) { comma(); printf("\"input_tokens\":%d", input) }
  if (output > 0) { comma(); printf("\"output_tokens\":%d", output) }
  if (total > 0) { comma(); printf("\"total_tokens\":%d", total) }
  printf("},\"quota\":{\"status\":\"%s\"", status)
  if (status == "unknown") printf(",\"reason\":\"quota_observed_unavailable\"")
  if (remaining > 0) printf(",\"remaining\":%d", remaining)
  if (limit > 0) printf(",\"limit\":%d", limit)
  printf("}}\n")
}`
}
