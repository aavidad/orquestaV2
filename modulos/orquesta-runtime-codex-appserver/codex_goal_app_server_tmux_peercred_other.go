//go:build !linux

package orquestaruntimecodexappserver

func codexAppServerTmuxSocketPeerIdentityV0(string) (codexAppServerTmuxProcessIdentityV0, error) {
	return codexAppServerTmuxProcessIdentityV0{}, errCodexAppServerTmuxPeerCredUnavailableV0
}
