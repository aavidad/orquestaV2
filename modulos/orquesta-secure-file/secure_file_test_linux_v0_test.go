//go:build linux

package orquestasecurefile

import "syscall"

func syscallMkfifoForTestV0(path string) error { return syscall.Mkfifo(path, 0o600) }
