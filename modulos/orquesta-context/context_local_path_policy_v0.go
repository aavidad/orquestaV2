package orquestacontext

import "strings"

var contextLocalPathForbiddenFragmentsV0 = []string{
	"/home/",
	"\\home\\",
	"/users/",
	"\\users\\",
	"c:\\users\\",
	"$home",
	"~/",
	".orquesta-runtime",
	".orquesta-server",
	".orquesta-smoke-work",
	".orquesta-codex-runtime",
	".orquesta-local-runtime",
	".orquesta-control",
	".orquesta-runs",
	".orquesta-worktrees",
	".orquesta-logs",
	"agent_ack.json",
	"agent_packet.json",
	"agent_prompt.txt",
	"agent_shutdown_checkpoint_ack.json",
	"director_decisions.json",
	"orquesta_shutdown_request.json",
	"codex_stdout.log",
	"codex_stderr.log",
	"codex_last_message.txt",
	"orquesta.env",
	"orquesta.db",
	".orquesta-inbox.md",
	"server.log",
	"logs/",
	".ssl-key.log",
	"transcript=",
	"raw_transcript=",
}

func contextLocalPathPublicDetailForbiddenV0(value string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(value, `\/`, "/"))
	if strings.TrimSpace(normalized) == "" {
		return false
	}
	for _, fragment := range contextLocalPathForbiddenFragmentsV0 {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
