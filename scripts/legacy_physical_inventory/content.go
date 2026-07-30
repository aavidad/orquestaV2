// Este fichero hashea únicamente regulares autorizados y detecta mutaciones.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"golang.org/x/sys/unix"
	"io"
	"os"
)

func (state *scanState) hashContent(parentFD int, name string, root anchoredRoot, parts []string, before *unix.Stat_t, value *record) string {
	if state.expired() {
		value.ContentState = "skipped_time_limit"
		value.ErrorCode = "budget_exhausted"
		return "time_budget"
	}
	if before.Size > state.budget.maxFileBytes {
		value.ContentState = "skipped_file_limit"
		return "file_size_limit"
	}
	remaining := state.budget.maxHashBytes - state.hashedBytes
	if before.Size > remaining {
		value.ContentState = "skipped_global_limit"
		return "global_hash_budget"
	}
	fd, err := openBeneath(parentFD, name, root.fileFlags)
	if err != nil {
		value.ContentState = "read_error"
		if root.fileFlags&unix.O_NOATIME != 0 &&
			(errors.Is(err, unix.EPERM) || errors.Is(err, unix.EACCES)) {
			value.ErrorCode = stableErrorCode(errNoAtime)
		} else {
			value.ErrorCode = stableErrorCode(err)
		}
		return "read_error"
	}
	file := os.NewFile(uintptr(fd), "regular-confinado")
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil || opened.Mode&unix.S_IFMT != unix.S_IFREG {
		closeErr := file.Close()
		value.ContentState = "read_error"
		if err != nil {
			value.ErrorCode = stableErrorCode(err)
		} else if closeErr != nil {
			value.ErrorCode = stableErrorCode(closeErr)
		} else {
			value.ErrorCode = "not_regular"
		}
		return "read_error"
	}
	if !sameSnapshot(before, &opened) {
		closeErr := file.Close()
		value.ContentState = "changed"
		value.ChangedDuringScan = true
		if closeErr != nil {
			value.ErrorCode = stableErrorCode(closeErr)
		}
		return "changed"
	}
	hasher := sha256.New()
	count, copyErr := state.hashBlocks(file, hasher, before.Size)
	state.hashedBytes += count
	if errors.Is(copyErr, errBudget) {
		closeErr := file.Close()
		value.ContentState = "skipped_time_limit"
		value.ContentBytes = count
		value.ErrorCode = "budget_exhausted"
		if closeErr != nil {
			value.ContentState = "read_error"
			value.ErrorCode = stableErrorCode(closeErr)
			return "read_error"
		}
		return "time_budget"
	}
	if state.afterRead != nil {
		state.afterRead(root.alias, parts)
	}
	var after unix.Stat_t
	statErr := unix.Fstat(fd, &after)
	closeErr := file.Close()
	changed := copyErr != nil || statErr != nil || closeErr != nil || count != before.Size ||
		!sameSnapshot(before, &opened) || !sameSnapshot(&opened, &after)
	if changed {
		value.ContentState = "changed"
		value.ChangedDuringScan = true
		value.ContentBytes = count
		if copyErr != nil {
			value.ErrorCode = stableErrorCode(copyErr)
		} else if statErr != nil {
			value.ErrorCode = stableErrorCode(statErr)
		} else if closeErr != nil {
			value.ErrorCode = stableErrorCode(closeErr)
		}
		return "changed"
	}
	value.ContentState = "hashed"
	value.ContentSHA256 = hex.EncodeToString(hasher.Sum(nil))
	value.ContentBytes = count
	return ""
}
func (state *scanState) hashBlocks(file *os.File, destination io.Writer, expected int64) (int64, error) {
	buffer := make([]byte, 128*1024)
	var total int64
	for total < expected {
		if state.expired() {
			return total, errBudget
		}
		remaining := expected - total
		block := buffer
		if remaining < int64(len(block)) {
			block = block[:remaining]
		}
		count, err := file.Read(block)
		if count > 0 {
			if _, writeErr := destination.Write(block[:count]); writeErr != nil {
				return total, writeErr
			}
			total += int64(count)
		}
		if state.expired() {
			return total, errBudget
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return total, err
		}
		if count == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}
func sameSnapshot(left, right *unix.Stat_t) bool {
	return left.Dev == right.Dev &&
		left.Ino == right.Ino &&
		left.Mode == right.Mode &&
		left.Size == right.Size &&
		left.Uid == right.Uid &&
		left.Gid == right.Gid &&
		left.Mtim == right.Mtim &&
		left.Ctim == right.Ctim
}
