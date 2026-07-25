//go:build !linux

package codex

func runSupervisorTestProcess([]string) (int, bool) {
	return 0, false
}
