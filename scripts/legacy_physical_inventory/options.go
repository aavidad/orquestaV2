// Este fichero normaliza raíces, exclusiones y destinos antes de abrirlos.
package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var aliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type budgetOptions struct {
	maxEntries          int64
	maxDirectoryEntries int64
	maxDepth            int
	maxPathBytes        int64
	maxOutputBytes      int64
	maxHashBytes        int64
	maxFileBytes        int64
	timeout             time.Duration
}
type rootOption struct {
	alias       string
	mode        string
	path        string
	denied      [][]string
	afterAnchor func()
}
type options struct {
	roots               []rootOption
	jsonlPath           string
	manifestPath        string
	budget              budgetOptions
	started             time.Time
	locationReader      physicalLocationReader
	afterRead           func(root string, parts []string)
	beforeOpenDirectory func(root string, parts []string)
	afterReadDirectory  func(root string, parts []string)
	beforeVerifyEntry   func(root string, parts []string)
	mountIDReader       func(directoryFD int, name string, flags int) (uint64, error)
}

func parseRoots(values []string) ([]rootOption, error) {
	if len(values) == 0 {
		return nil, errors.Join(errInvalidInput, errors.New("missing_root"))
	}
	seen := make(map[string]bool, len(values))
	roots := make([]rootOption, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 || !aliasPattern.MatchString(parts[0]) {
			return nil, errors.Join(errInvalidInput, fmt.Errorf("invalid_root_alias: %q", value))
		}
		if seen[parts[0]] {
			return nil, errors.Join(errInvalidInput, fmt.Errorf("duplicate_root_alias: %q", parts[0]))
		}
		mode, path := modeMetadata, parts[1]
		if strings.HasPrefix(path, modeMetadata+":") {
			path = strings.TrimPrefix(path, modeMetadata+":")
		} else if strings.HasPrefix(path, modeContent+":") {
			mode, path = modeContent, strings.TrimPrefix(path, modeContent+":")
		}
		if !filepath.IsAbs(path) {
			return nil, errors.Join(errInvalidInput, fmt.Errorf("root_not_absolute: %q", parts[0]))
		}
		clean := filepath.Clean(path)
		roots = append(roots, rootOption{alias: parts[0], mode: mode, path: clean})
		seen[parts[0]] = true
	}
	return roots, nil
}
func applyDenied(roots []rootOption, values []string) error {
	byAlias := make(map[string]*rootOption, len(roots))
	for index := range roots {
		byAlias[roots[index].alias] = &roots[index]
	}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		root := byAlias[parts[0]]
		if len(parts) != 2 || root == nil {
			return errors.Join(errInvalidInput, fmt.Errorf("deny_unknown_alias: %q", value))
		}
		relative := filepath.Clean(parts[1])
		if relative == "." || filepath.IsAbs(relative) || relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.Join(errInvalidInput, fmt.Errorf("deny_not_relative: %q", value))
		}
		segments := strings.Split(relative, string(filepath.Separator))
		for _, current := range root.denied {
			if slices.Equal(current, segments) {
				return errors.Join(errInvalidInput, fmt.Errorf("duplicate_deny: %q", value))
			}
		}
		root.denied = append(root.denied, segments)
	}
	return nil
}
func validateOptions(opts *options) error {
	if len(opts.roots) == 0 || len(opts.roots) > maxRootInputs {
		return errors.Join(errInvalidInput, errors.New("invalid_root_count"))
	}
	deniedCount := 0
	for _, root := range opts.roots {
		deniedCount += len(root.denied)
	}
	if deniedCount > maxDeniedInputs {
		return errors.Join(errInvalidInput, errors.New("invalid_deny_count"))
	}
	if opts.jsonlPath == "" || opts.manifestPath == "" {
		return errors.Join(errInvalidInput, errors.New("missing_outputs"))
	}
	if opts.budget.maxEntries <= 1 || opts.budget.maxDirectoryEntries <= 0 ||
		opts.budget.maxDepth <= 0 || opts.budget.maxPathBytes <= 0 ||
		opts.budget.maxOutputBytes <= terminalOutputReserve ||
		opts.budget.maxHashBytes < 0 ||
		opts.budget.maxFileBytes <= 0 || opts.budget.timeout <= 0 {
		return errors.Join(errInvalidInput, errors.New("invalid_budget"))
	}
	if opts.budget.maxEntries > defaultMaxEntries ||
		opts.budget.maxDirectoryEntries > defaultMaxDirectoryEntries ||
		opts.budget.maxDepth > defaultMaxDepth ||
		opts.budget.maxPathBytes > defaultMaxPathBytes ||
		opts.budget.maxOutputBytes > defaultMaxOutputBytes ||
		opts.budget.maxHashBytes > defaultMaxHashBytes ||
		opts.budget.maxFileBytes > defaultMaxFileBytes ||
		opts.budget.timeout > defaultTimeout {
		return errors.Join(errInvalidInput, errors.New("budget_above_canonical_maximum"))
	}
	if opts.expired() {
		return errBudget
	}
	jsonl, err := safeOutputPath(opts.jsonlPath)
	if err != nil {
		return errors.Join(errOutputPath, err)
	}
	manifest, err := safeOutputPath(opts.manifestPath)
	if err != nil {
		return errors.Join(errOutputPath, err)
	}
	if jsonl == manifest {
		return errors.Join(errInvalidInput, errors.New("outputs_not_distinct"))
	}
	if filepath.Dir(jsonl) != filepath.Dir(manifest) {
		return errors.Join(errOutputPath, errors.New("outputs_require_same_private_directory"))
	}
	for _, root := range opts.roots {
		if pathWithin(root.path, jsonl) || pathWithin(root.path, manifest) {
			return errors.Join(errOutputPath, fmt.Errorf("output_inside_root: %q", root.alias))
		}
	}
	opts.jsonlPath, opts.manifestPath = jsonl, manifest
	return nil
}
func (opts options) expired() bool {
	return !opts.started.IsZero() && !time.Now().Before(opts.started.Add(opts.budget.timeout))
}
func safeOutputPath(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("la ruta debe ser absoluta")
	}
	clean := filepath.Clean(path)
	return clean, nil
}
func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
