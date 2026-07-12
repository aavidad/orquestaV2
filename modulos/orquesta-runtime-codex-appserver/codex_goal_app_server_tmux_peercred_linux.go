//go:build linux

package orquestaruntimecodexappserver

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

func codexAppServerTmuxSocketPeerIdentityV0(socketPath string) (codexAppServerTmuxProcessIdentityV0, error) {
	socketPath = strings.TrimSpace(socketPath)
	connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return codexAppServerTmuxProcessIdentityV0{}, err
	}
	defer connection.Close()
	raw, err := connection.SyscallConn()
	if err != nil {
		return codexAppServerTmuxProcessIdentityV0{}, err
	}
	var credentials *syscall.Ucred
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return codexAppServerTmuxProcessIdentityV0{}, err
	}
	if controlErr != nil {
		return codexAppServerTmuxProcessIdentityV0{}, controlErr
	}
	if credentials == nil || credentials.Pid <= 0 {
		return codexAppServerTmuxProcessIdentityV0{}, errors.New("codex_app_server_tmux_peer_credentials_invalid")
	}
	pid := int(credentials.Pid)
	startRef := codexAppServerTmuxProcessStartRefV0(pid)
	if startRef == "" {
		return codexAppServerTmuxProcessIdentityV0{}, errors.New("codex_app_server_tmux_peer_identity_incomplete")
	}
	return codexAppServerTmuxProcessIdentityV0{PID: pid, StartRef: startRef}, nil
}
