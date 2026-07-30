// Este fichero recorre descriptores confinados y nunca sigue enlaces.
package main

import (
	"errors"
	"golang.org/x/sys/unix"
	"sort"
	"time"
)

var (
	errBudget            = errors.New("budget_exhausted")
	errDirectoryBudget   = errors.New("directory_budget_exhausted")
	errChangedDuringScan = errors.New("changed_during_scan")
	errIncomplete        = errors.New("inventory_incomplete")
)

type scanState struct {
	writer              *recordWriter
	budget              budgetOptions
	started             time.Time
	entries             int64
	hashedBytes         int64
	complete            bool
	rootSummaries       []rootSummary
	afterRead           func(root string, parts []string)
	beforeOpenDirectory func(root string, parts []string)
	afterReadDirectory  func(root string, parts []string)
	beforeVerifyEntry   func(root string, parts []string)
	mountIDReader       func(directoryFD int, name string, flags int) (uint64, error)
	now                 func() time.Time
}

func run(opts options) (result error) {
	started := opts.started
	if started.IsZero() {
		started = time.Now()
		opts.started = started
	}
	if err := validateOptions(&opts); err != nil {
		return err
	}
	roots, err := anchorRootsBefore(opts.roots, started.Add(opts.budget.timeout))
	if err != nil {
		return errors.Join(errRootAnchor, err)
	}
	defer func() {
		result = errors.Join(result, closeAnchoredRoots(roots))
	}()
	publication, err := newPublication(opts, roots)
	if err != nil {
		return errors.Join(errPublication, err)
	}
	defer func() {
		result = errors.Join(result, publication.abort())
	}()
	state := &scanState{
		writer:              &recordWriter{file: publication.jsonl, maxBytes: opts.budget.maxOutputBytes},
		budget:              opts.budget,
		started:             started,
		complete:            true,
		afterRead:           opts.afterRead,
		beforeOpenDirectory: opts.beforeOpenDirectory,
		afterReadDirectory:  opts.afterReadDirectory,
		beforeVerifyEntry:   opts.beforeVerifyEntry,
		mountIDReader:       opts.mountIDReader,
		now:                 time.Now,
	}
	if state.mountIDReader == nil {
		state.mountIDReader = mountIDAt
	}
	for index, root := range roots {
		summary := rootSummary{Alias: root.alias, Mode: root.mode, Complete: true, Counts: map[string]int64{}}
		err = state.scanRoot(root, &summary)
		if err != nil {
			summary.Complete, state.complete = false, false
			if writeErr := state.writeTerminalError(root.alias, nil, "scan_root", err, &summary); writeErr != nil {
				return writeErr
			}
		}
		state.rootSummaries = append(state.rootSummaries, summary)
		if errors.Is(err, errBudget) || errors.Is(err, errOutputBudget) {
			for _, pending := range roots[index+1:] {
				state.rootSummaries = append(state.rootSummaries, rootSummary{
					Alias: pending.alias, Mode: pending.mode, Complete: false,
					Counts: map[string]int64{},
				})
			}
			break
		}
	}
	if err := state.writer.file.Sync(); err != nil {
		return err
	}
	manifestValue, err := buildManifest(opts, state, publication)
	if err != nil {
		return err
	}
	if err := publication.prepareManifest(manifestValue); err != nil {
		return err
	}
	if err := publication.publish(); err != nil {
		return err
	}
	if !state.complete {
		return errIncomplete
	}
	return nil
}
func (state *scanState) scanRoot(root anchoredRoot, summary *rootSummary) error {
	fd := int(root.file.Fd())
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return err
	}
	if err := state.emit(metadataRecord("root", root.alias, root.mode, nil, &stat), summary); err != nil {
		return err
	}
	return state.scanDirectory(root, summary, fd, nil, 0)
}
func (state *scanState) scanDirectory(
	root anchoredRoot,
	summary *rootSummary,
	fd int,
	parent []string,
	parentBytes int64,
) error {
	if err := state.checkBudget(); err != nil {
		return err
	}
	var before unix.Stat_t
	if err := unix.Fstat(fd, &before); err != nil {
		return err
	}
	entries, err := readDirectoryEntries(fd, state.budget.maxDirectoryEntries)
	directoryOverflow := errors.Is(err, errDirectoryBudget)
	if err != nil && !directoryOverflow {
		return err
	}
	if state.afterReadDirectory != nil {
		state.afterReadDirectory(root.alias, parent)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	if directoryOverflow {
		entries = entries[:state.budget.maxDirectoryEntries]
	}
	for _, entry := range entries {
		if err := state.scanEntry(root, summary, fd, parent, parentBytes, entry); err != nil {
			return err
		}
	}
	var after unix.Stat_t
	if err := unix.Fstat(fd, &after); err != nil {
		return err
	}
	if !sameSnapshot(&before, &after) {
		return errChangedDuringScan
	}
	if directoryOverflow {
		return errDirectoryBudget
	}
	return nil
}
