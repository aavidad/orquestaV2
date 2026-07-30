// Este fichero gobierna presupuestos y diagnósticos del flujo sin recorrer el árbol.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
)

func (state *scanState) emit(value record, summary *rootSummary) error {
	if err := state.checkBudget(); err != nil {
		return err
	}
	if err := state.writer.write(value); err != nil {
		return err
	}
	state.entries++
	if summary != nil {
		summary.Counts[value.Kind]++
	}
	return nil
}
func (state *scanState) emitIgnoringTime(value record, summary *rootSummary) error {
	if state.entries >= state.budget.maxEntries-1 {
		return errBudget
	}
	if err := state.writer.write(value); err != nil {
		return err
	}
	state.entries++
	summary.Counts[value.Kind]++
	return nil
}
func (state *scanState) writeError(
	alias string,
	parts []string,
	operation string,
	cause error,
	summary *rootSummary,
) error {
	return state.emit(record{
		Schema: schemaVersion, Kind: "error", Root: alias, Path: encodePath(parts),
		Operation: operation, ErrorCode: stableErrorCode(cause),
	}, summary)
}
func (state *scanState) writeExclusion(
	alias string,
	parts []string,
	reason string,
	summary *rootSummary,
) error {
	return state.emit(record{
		Schema: schemaVersion, Kind: "exclusion", Root: alias,
		Path: encodePath(parts), Reason: reason,
	}, summary)
}
func (state *scanState) writeTerminalError(
	alias string,
	parts []string,
	operation string,
	cause error,
	summary *rootSummary,
) error {
	if state.entries >= state.budget.maxEntries {
		return errBudget
	}
	if err := state.writer.writeTerminal(record{
		Schema: schemaVersion, Kind: "error", Root: alias, Path: encodePath(parts),
		Operation: operation, ErrorCode: stableErrorCode(cause),
	}); err != nil {
		return err
	}
	state.entries++
	summary.Counts["error"]++
	return nil
}
func (state *scanState) checkBudget() error {
	if state.entries >= state.budget.maxEntries-1 || state.expired() {
		return errBudget
	}
	return nil
}
func (state *scanState) expired() bool {
	return !state.now().Before(state.started.Add(state.budget.timeout))
}
func (state *scanState) verifyObservedEntry(
	root anchoredRoot,
	summary *rootSummary,
	parentFD int,
	name string,
	parts []string,
	before *unix.Stat_t,
	value *record,
) (bool, error) {
	err := verifyEntrySnapshot(parentFD, name, root.mountID, before)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EXDEV) {
		emitErr := state.emit(record{
			Schema: schemaVersion, Kind: "exclusion", Root: root.alias,
			Path: encodePath(parts), Reason: "mount_boundary",
		}, summary)
		return false, emitErr
	}
	summary.Complete, state.complete = false, false
	value.ChangedDuringScan = true
	return true, state.writeError(root.alias, parts, "verify_entry", err, summary)
}
