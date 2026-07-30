// Este fichero clasifica una entrada ya confinada y delega su tratamiento acotado.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func (state *scanState) scanEntry(
	root anchoredRoot,
	summary *rootSummary,
	parentFD int,
	parent []string,
	parentBytes int64,
	entry os.DirEntry,
) error {
	if err := state.checkBudget(); err != nil {
		return err
	}
	parts, pathBytes, limit := boundedChildPath(parent, parentBytes, entry.Name(), state.budget)
	if limit != "" {
		summary.Complete, state.complete = false, false
		return state.emit(record{
			Schema: schemaVersion, Kind: "exclusion", Root: root.alias,
			Path: encodePath(parent), Reason: limit, RejectedSegment: encodeBinary(entry.Name()),
		}, summary)
	}
	if denied(parts, root.denied) {
		return state.writeExclusion(root.alias, parts, "sensitive_path", summary)
	}
	mountID, err := state.mountIDReader(parentFD, entry.Name(), unix.AT_SYMLINK_NOFOLLOW)
	if err != nil {
		summary.Complete, state.complete = false, false
		return state.writeError(root.alias, parts, "stat_mount", err, summary)
	}
	if mountID != root.mountID {
		return state.writeExclusion(root.alias, parts, "mount_boundary", summary)
	}
	var stat unix.Stat_t
	if err := unix.Fstatat(parentFD, entry.Name(), &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		summary.Complete, state.complete = false, false
		return state.writeError(root.alias, parts, "lstat", err, summary)
	}
	switch stat.Mode & unix.S_IFMT {
	case unix.S_IFDIR:
		return state.scanDirectoryEntry(root, summary, parentFD, parts, pathBytes, &stat)
	case unix.S_IFREG:
		return state.scanRegularEntry(root, summary, parentFD, entry.Name(), parts, &stat)
	case unix.S_IFLNK:
		return state.scanObservedEntry("symlink", root, summary, parentFD, entry.Name(), parts, &stat)
	default:
		return state.scanObservedEntry("special", root, summary, parentFD, entry.Name(), parts, &stat)
	}
}
func (state *scanState) scanDirectoryEntry(
	root anchoredRoot,
	summary *rootSummary,
	parentFD int,
	parts []string,
	pathBytes int64,
	stat *unix.Stat_t,
) error {
	if state.beforeOpenDirectory != nil {
		state.beforeOpenDirectory(root.alias, parts)
	}
	err := state.scanChildDirectory(root, summary, parentFD, parts, pathBytes, stat)
	if err == nil || errors.Is(err, errBudget) {
		return err
	}
	if errors.Is(err, unix.EXDEV) {
		return state.writeExclusion(root.alias, parts, "mount_boundary", summary)
	}
	summary.Complete, state.complete = false, false
	return state.writeError(root.alias, parts, "open_directory", err, summary)
}
func (state *scanState) scanRegularEntry(
	root anchoredRoot,
	summary *rootSummary,
	parentFD int,
	name string,
	parts []string,
	stat *unix.Stat_t,
) error {
	value := metadataRecord("file", root.alias, root.mode, parts, stat)
	issue := ""
	if root.mode == modeMetadata {
		value.ContentState = "not_requested"
	} else {
		issue = state.hashContent(parentFD, name, root, parts, stat, &value)
		if issue != "" {
			summary.Complete, state.complete = false, false
		}
	}
	if state.beforeVerifyEntry != nil {
		state.beforeVerifyEntry(root.alias, parts)
	}
	if err := verifyEntrySnapshot(parentFD, name, root.mountID, stat); err != nil {
		if errors.Is(err, unix.EXDEV) {
			return state.writeExclusion(root.alias, parts, "mount_boundary", summary)
		}
		summary.Complete, state.complete = false, false
		value.ChangedDuringScan = true
		if issue == "" {
			issue, value.ContentState, value.ErrorCode = "changed", "changed", stableErrorCode(err)
		} else if err := state.writeError(root.alias, parts, "verify_entry", err, summary); err != nil {
			return err
		}
	}
	if issue == "time_budget" {
		if err := state.emitIgnoringTime(value, summary); err != nil {
			return err
		}
		return errBudget
	}
	if err := state.emit(value, summary); err != nil {
		return err
	}
	return state.writeRegularIssue(root.alias, parts, issue, value.ErrorCode, summary)
}
func (state *scanState) writeRegularIssue(
	alias string,
	parts []string,
	issue string,
	errorCode string,
	summary *rootSummary,
) error {
	if issue == "" {
		return nil
	}
	value := record{
		Schema: schemaVersion, Kind: "exclusion", Root: alias,
		Path: encodePath(parts), Reason: issue,
	}
	if issue == "read_error" || issue == "changed" {
		value.Kind, value.Operation, value.ErrorCode = "error", "hash_content", errorCode
		if issue == "changed" {
			value.ErrorCode = "changed_during_scan"
		}
	}
	return state.emit(value, summary)
}
func (state *scanState) scanObservedEntry(
	kind string,
	root anchoredRoot,
	summary *rootSummary,
	parentFD int,
	name string,
	parts []string,
	stat *unix.Stat_t,
) error {
	value := metadataRecord(kind, root.alias, root.mode, parts, stat)
	if kind == "special" {
		value.SpecialType = specialType(stat.Mode)
	}
	if state.beforeVerifyEntry != nil {
		state.beforeVerifyEntry(root.alias, parts)
	}
	keep, err := state.verifyObservedEntry(root, summary, parentFD, name, parts, stat, &value)
	if err != nil || !keep {
		return err
	}
	return state.emit(value, summary)
}
func (state *scanState) scanChildDirectory(
	root anchoredRoot,
	summary *rootSummary,
	parentFD int,
	parts []string,
	pathBytes int64,
	stat *unix.Stat_t,
) error {
	fd, err := openBeneath(parentFD, parts[len(parts)-1], root.directoryFlags)
	if err != nil {
		if root.directoryFlags&unix.O_NOATIME != 0 &&
			(errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES)) {
			return errNoAtime
		}
		return err
	}
	file := os.NewFile(uintptr(fd), "directorio-confinado")
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil {
		return errors.Join(err, file.Close())
	}
	if !sameSnapshot(stat, &opened) {
		return errors.Join(errChangedDuringScan, file.Close())
	}
	if err := state.emit(metadataRecord("directory", root.alias, root.mode, parts, stat), summary); err != nil {
		return errors.Join(err, file.Close())
	}
	return errors.Join(state.scanDirectory(root, summary, fd, parts, pathBytes), file.Close())
}
