package orquestaserver

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

type DaemonIdentityV0 struct {
	ProcessRef     string
	DaemonEpochRef string
}

func NewDaemonIdentityV0(pid int, startedAt time.Time) DaemonIdentityV0 {
	started := strings.TrimSpace(formatTimeV0(startedAt))
	seed := strconv.Itoa(pid) + "|" + started
	sum := sha256.Sum256([]byte(seed))
	ref := hex.EncodeToString(sum[:])[:16]
	return DaemonIdentityV0{
		ProcessRef:     "process-ref-orquesta-server-" + ref,
		DaemonEpochRef: "daemon-epoch-ref-orquesta-server-" + ref,
	}
}
