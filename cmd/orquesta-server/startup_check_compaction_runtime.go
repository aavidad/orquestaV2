package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func archiveStartupRuntimeDirsV0(runtimeDir string, revisionDir string, runRefs []string) (int, error) {
	if strings.TrimSpace(runtimeDir) == "" || strings.TrimSpace(revisionDir) == "" {
		return 0, nil
	}
	targetRoot := filepath.Join(revisionDir, "runtime_archived")
	archived := 0
	for _, runRef := range runRefs {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" {
			continue
		}
		source := filepath.Join(runtimeDir, runRef)
		if _, err := os.Stat(source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return archived, err
		}
		if err := os.MkdirAll(targetRoot, 0o755); err != nil {
			return archived, err
		}
		target := filepath.Join(targetRoot, runRef)
		if err := os.Rename(source, target); err != nil {
			return archived, err
		}
		archived++
	}
	return archived, nil
}

func startupRevisionTimestampV0(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	return value.UTC().Format("20060102T150405Z")
}

func startupReadyMessageV0(base string, compaction startupStateCompactionV0) string {
	base = strings.TrimSpace(base)
	if !compaction.CompactionNeeded {
		return base
	}
	return fmt.Sprintf(
		"%s; compactacion_activa cola=%d control=%d runtime=%d revision=%s",
		base,
		compaction.QueueRemoved,
		compaction.ControlRemoved,
		compaction.RuntimeArchived,
		compaction.RevisionDir,
	)
}

func startupReadyEvidenceRefsV0(base []string, compaction startupStateCompactionV0) []string {
	refs := append([]string(nil), base...)
	if compaction.CompactionNeeded {
		refs = append(refs, "evidence-ref-orquesta-startup-state-compacted")
	}
	return appendStartupEvidenceRefV0(refs, "")
}
