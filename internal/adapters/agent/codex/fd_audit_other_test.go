//go:build !linux

package codex

func platformDescriptorCloseOnExec(int) bool {
	return false
}
