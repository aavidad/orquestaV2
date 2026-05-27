package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
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
		source := filepath.Clean(filepath.Join(runtimeDir, runRef))
		runtimeRoot := filepath.Clean(runtimeDir)
		if source != runtimeRoot && !strings.HasPrefix(source, runtimeRoot+string(os.PathSeparator)) {
			continue
		}
		if _, err := os.Stat(source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return archived, err
		}
		if err := os.MkdirAll(targetRoot, 0o700); err != nil {
			return archived, err
		}
		target := filepath.Join(targetRoot, startupSafeRefPartV0(runRef))
		summary, err := summarizeStartupRuntimeDirV0(source, runRef)
		if err != nil {
			return archived, err
		}
		if err := writeStartupJSONFileV0(filepath.Join(target, "manifest.json"), summary); err != nil {
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

func startupRevisionRefV0(value time.Time) string {
	return "revision-ref-orquesta-startup-" + strings.ToLower(startupRevisionTimestampV0(value))
}

func startupReadyMessageV0(base string, compaction startupStateCompactionV0) string {
	base = strings.TrimSpace(base)
	if !compaction.CompactionNeeded {
		return base
	}
	return fmt.Sprintf(
		"%s; compactacion_activa revision_ref=%s cola=%d control=%d archivos=%d",
		base,
		compaction.RevisionRef,
		compaction.QueueRemoved,
		compaction.ControlRemoved,
		compaction.RuntimeArchived,
	)
}

func startupReadyEvidenceRefsV0(base []string, compaction startupStateCompactionV0) []string {
	refs := append([]string(nil), base...)
	if compaction.CompactionNeeded {
		refs = append(refs, "evidence-ref-orquesta-startup-state-compacted")
		refs = append(refs, compaction.RevisionRef)
	}
	return appendStartupEvidenceRefV0(refs, "")
}

func startupRevisionSummaryFromCompactionV0(compaction startupStateCompactionV0) orquestaserver.StartupRevisionSummaryV0 {
	if !compaction.CompactionNeeded {
		return orquestaserver.StartupRevisionSummaryV0{}
	}
	return orquestaserver.StartupRevisionSummaryV0{
		RevisionRef:     compaction.RevisionRef,
		QueueRemoved:    compaction.QueueRemoved,
		QueueKept:       compaction.QueueKept,
		ControlRemoved:  compaction.ControlRemoved,
		ControlKept:     compaction.ControlKept,
		RuntimeArchived: compaction.RuntimeArchived,
		Artifacts:       compaction.Artifacts,
		RetentionDays:   compaction.RetentionDays,
		MaxBytes:        compaction.MaxBytes,
	}
}
