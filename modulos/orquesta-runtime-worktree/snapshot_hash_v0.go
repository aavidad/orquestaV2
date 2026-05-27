package orquestaruntimeworktree

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
	"strings"
)

func hashWorktreeFileV0(
	path string,
	rel string,
	size int64,
	budget WorktreeSnapshotReadBudgetV0,
	totalBefore int64,
) (WorktreeSnapshotFileV0, int64, *WorktreeIssueV0) {
	file, err := os.Open(path)
	if err != nil {
		issue := worktreeIssueV0(WorktreeIssueSnapshotUnreadableV0, "file", rel)
		return WorktreeSnapshotFileV0{}, 0, &issue
	}
	defer file.Close()
	reader := newWorktreeFileHashReaderV0(rel, sha256.New(), budget, totalBefore)
	if err := reader.readV0(file); err != nil {
		return WorktreeSnapshotFileV0{}, 0, err
	}
	return WorktreeSnapshotFileV0{
		Path:      rel,
		Digest:    hex.EncodeToString(reader.sumV0()),
		Size:      size,
		LineCount: reader.lineCountV0(),
	}, reader.bytesRead, nil
}

type worktreeFileHashReaderV0 struct {
	rel         string
	hash        hash.Hash
	budget      WorktreeSnapshotReadBudgetV0
	totalBefore int64
	bytesRead   int64
	goFile      bool
	lineBreaks  int
	lastByte    byte
	sawBytes    bool
}

func newWorktreeFileHashReaderV0(
	rel string,
	hashValue hash.Hash,
	budget WorktreeSnapshotReadBudgetV0,
	totalBefore int64,
) *worktreeFileHashReaderV0 {
	return &worktreeFileHashReaderV0{
		rel:         rel,
		hash:        hashValue,
		budget:      budget,
		totalBefore: totalBefore,
		goFile:      strings.HasSuffix(rel, ".go"),
	}
}

func (reader *worktreeFileHashReaderV0) readV0(file *os.File) *WorktreeIssueV0 {
	buffer := make([]byte, 32*1024)
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			reader.bytesRead += int64(n)
			if issue := reader.budgetIssueV0(); issue != nil {
				return issue
			}
			_, _ = reader.hash.Write(chunk)
			reader.countLinesV0(chunk)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			issue := worktreeIssueV0(WorktreeIssueSnapshotUnreadableV0, "file", reader.rel)
			return &issue
		}
	}
}

func (reader *worktreeFileHashReaderV0) budgetIssueV0() *WorktreeIssueV0 {
	if reader.bytesRead > reader.budget.MaxFileBytes {
		issue := worktreeIssueV0(WorktreeIssueSnapshotFileTooLargeV0, "file", reader.rel)
		return &issue
	}
	if reader.totalBefore+reader.bytesRead > reader.budget.MaxTotalBytes {
		issue := worktreeIssueV0(
			WorktreeIssueSnapshotTooLargeV0,
			"max_total_bytes",
			"worktree_snapshot_total_budget_exceeded",
		)
		return &issue
	}
	return nil
}

func (reader *worktreeFileHashReaderV0) countLinesV0(chunk []byte) {
	if !reader.goFile || len(chunk) == 0 {
		return
	}
	reader.sawBytes = true
	for _, item := range chunk {
		if item == '\n' {
			reader.lineBreaks++
		}
		reader.lastByte = item
	}
}

func (reader *worktreeFileHashReaderV0) lineCountV0() int {
	if !reader.goFile || !reader.sawBytes {
		return 0
	}
	lines := reader.lineBreaks
	if reader.lastByte != '\n' {
		lines++
	}
	return lines
}

func (reader *worktreeFileHashReaderV0) sumV0() []byte {
	return reader.hash.Sum(nil)
}
