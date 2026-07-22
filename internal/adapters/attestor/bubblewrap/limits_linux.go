//go:build linux

package bubblewrap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

const (
	descriptorGlobalReserve = 64
	descriptorRunReserve    = 16
)

type inheritedResourceEnvelope struct {
	noFile, fileSize, addressSpace, cpu unix.Rlimit
}

func currentResourceEnvelope(limits Limits) (string, error) {
	envelope := inheritedResourceEnvelope{}
	for resource, target := range map[int]*unix.Rlimit{
		unix.RLIMIT_NOFILE: &envelope.noFile, unix.RLIMIT_FSIZE: &envelope.fileSize,
		unix.RLIMIT_AS: &envelope.addressSpace, unix.RLIMIT_CPU: &envelope.cpu,
	} {
		if unix.Getrlimit(resource, target) != nil {
			return "", resourceError()
		}
	}
	if !resourceEnvelopeAllowed(limits, envelope) {
		return "", resourceError()
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%d:%d:%d:%d:%d:%d",
		envelope.noFile.Cur, envelope.noFile.Max, envelope.fileSize.Cur, envelope.fileSize.Max,
		envelope.addressSpace.Cur, envelope.addressSpace.Max, envelope.cpu.Cur, envelope.cpu.Max)))
	return hex.EncodeToString(digest[:]), nil
}

func resourceEnvelopeAllowed(limits Limits, envelope inheritedResourceEnvelope) bool {
	finite := func(value unix.Rlimit) bool {
		return value.Cur > 0 && value.Max > 0 && value.Cur <= value.Max &&
			value.Cur != unix.RLIM_INFINITY && value.Max != unix.RLIM_INFINITY
	}
	if !finite(envelope.noFile) || !finite(envelope.fileSize) || !finite(envelope.addressSpace) || !finite(envelope.cpu) {
		return false
	}
	concurrent := uint64(limits.MaxConcurrentRuns)
	memory := uint64(limits.MemoryMaxBytes)
	if concurrent == 0 || memory == 0 || concurrent > ^uint64(0)/memory {
		return false
	}
	cpuSeconds := uint64(limits.Timeout / time.Second)
	if limits.Timeout%time.Second != 0 {
		cpuSeconds++
	}
	return envelope.fileSize.Max <= uint64(limits.MaxSubjectBytes) &&
		envelope.addressSpace.Max <= memory*concurrent && envelope.cpu.Max <= cpuSeconds
}

func currentSnapshotFileLimit(limits Limits) (int64, error) {
	var resource unix.Rlimit
	open, err := os.ReadDir("/proc/self/fd")
	if err != nil || unix.Getrlimit(unix.RLIMIT_NOFILE, &resource) != nil {
		return 0, resourceError()
	}
	soft := resource.Cur
	if soft > uint64(^uint64(0)>>1) {
		soft = uint64(^uint64(0) >> 1)
	}
	limit := snapshotFileLimit(int64(soft), int64(len(open)), limits.MaxConcurrentRuns,
		SubjectEntryLimit(limits.MaxSubjectBytes))
	if limit <= 0 {
		return 0, resourceError()
	}
	return limit, nil
}

func snapshotFileLimit(soft, open, concurrent, subjectLimit int64) int64 {
	available := soft - open - descriptorGlobalReserve
	if available <= 0 || concurrent <= 0 || subjectLimit <= 0 {
		return 0
	}
	perRun := available/concurrent - descriptorRunReserve
	if perRun <= 0 {
		return 0
	}
	if perRun > subjectLimit {
		return subjectLimit
	}
	return perRun
}
