//go:build !linux

package codex

import (
	"errors"
	"os"
	"time"
)

type cgroupIdentity struct{ Device, Inode uint64 }
type codexCgroupRoot struct {
	rootIdentity, controlIdentity cgroupIdentity
}
type codexCgroupLeaf struct {
	name     string
	identity cgroupIdentity
}

func platformCgroupRequired() bool { return false }
func validCgroupName(string) bool  { return false }

func openCodexCgroupRoot(string) (*codexCgroupRoot, error) {
	return nil, errors.New(CodeControlUnsupported)
}

func (root *codexCgroupRoot) populated(processRecord) (bool, error) {
	return false, errors.New(CodeControlUnsupported)
}
func (root *codexCgroupRoot) drain(processRecord, time.Duration) error {
	return errors.New(CodeControlUnsupported)
}
func (root *codexCgroupRoot) kill(processRecord) error {
	return errors.New(CodeControlUnsupported)
}
func (root *codexCgroupRoot) leafForRecord(processRecord) (*os.File, error) {
	return nil, errors.New(CodeControlUnsupported)
}
func (root *codexCgroupRoot) populateRecord(*processRecord, *codexCgroupLeaf) {}

func (root *codexCgroupRoot) remove(processRecord) error  { return errors.New(CodeControlUnsupported) }
func (root *codexCgroupRoot) close() error                { return nil }
func (leaf *codexCgroupLeaf) apply(*os.Process) error     { return errors.New(CodeControlUnsupported) }
func (leaf *codexCgroupLeaf) close() error                { return nil }
func (leaf *codexCgroupLeaf) destroy(time.Duration) error { return nil }

func validateSupervisorCgroups(processRecord, *os.File, *os.File) error {
	return errors.New(CodeControlUnsupported)
}

func moveSupervisorToControl(processRecord, *os.File, *os.File) error {
	return errors.New(CodeControlUnsupported)
}

func (adapter *Adapter) allocateExecutionCgroup(string) (*codexCgroupLeaf, error) {
	return nil, errors.New(CodeControlUnsupported)
}
func (adapter *Adapter) cleanupOrphanedCgroup(string, time.Duration) error { return nil }
func (adapter *Adapter) cleanupTerminalCgroup(*executionState) error       { return nil }

func duplicateCgroupFile(*os.File, string) (*os.File, error) {
	return nil, errors.New(CodeControlUnsupported)
}
