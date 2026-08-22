package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"orquesta/internal/adapters/config/effectivefile"
	"orquesta/internal/candidateseal"
	"orquesta/internal/config"
)

func runGateA(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || (arguments[0] != "create" && arguments[0] != "verify" && arguments[0] != "effective-config") {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.arguments_invalid")
		return 2
	}
	operation := arguments[0]
	if operation == "effective-config" {
		return runGateAEffectiveConfig(arguments[1:], stdout, stderr)
	}
	flags := flag.NewFlagSet("orquesta candidate gate-a "+operation, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	sourceRoot := flags.String("source-root", "", "")
	binaryPath := flags.String("binary", "", "")
	configPath := flags.String("effective-config", "", "")
	manifestPath := flags.String("manifest", "", "")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 ||
		*sourceRoot == "" || *binaryPath == "" || *configPath == "" || *manifestPath == "" {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.arguments_invalid")
		return 2
	}

	manifestSubjectPath, err := gateACanonicalSubjectPath(*manifestPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.input_invalid")
		return 1
	}
	input, err := gateAInput(*sourceRoot, *binaryPath, *configPath, manifestSubjectPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.input_invalid")
		return 1
	}
	if operation == "verify" {
		encoded, readErr := readGateARegularFile(manifestSubjectPath, true)
		manifest, decodeErr := candidateseal.Decode(encoded)
		if readErr != nil || decodeErr != nil || candidateseal.Verify(manifest, input) != nil {
			_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.verification_failed")
			return 1
		}
		_, _ = fmt.Fprintln(stdout, manifest.CandidateSHA256)
		return 0
	}

	manifest, err := candidateseal.Build(input)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.build_failed")
		return 1
	}
	encoded, err := candidateseal.Encode(manifest)
	if err != nil || writeGateAManifest(manifestSubjectPath, encoded) != nil {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.manifest_write_failed")
		return 1
	}
	_, _ = fmt.Fprintln(stdout, manifest.CandidateSHA256)
	return 0
}

func runGateAEffectiveConfig(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("orquesta candidate gate-a effective-config", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	output := flags.String("output", "", "")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 || *output == "" {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.arguments_invalid")
		return 2
	}
	snapshot, err := config.Resolve(config.ResolveOptions{})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.effective_config_failed")
		return 1
	}
	content, err := snapshot.EffectiveJSON()
	if err != nil || effectivefile.Write(context.Background(), effectivefile.Options{
		Path: *output, Content: content, MaxExistingBytes: snapshot.ConfigEffectiveMaxExistingBytes(),
	}) != nil {
		_, _ = fmt.Fprintln(stderr, "code=candidate_gate_a.effective_config_failed")
		return 1
	}
	_, _ = fmt.Fprintln(stdout, snapshot.Hash())
	return 0
}

func gateAInput(sourceRoot, binaryPath, configPath, manifestPath string) (candidateseal.Input, error) {
	root, err := filepath.Abs(sourceRoot)
	if err != nil {
		return candidateseal.Input{}, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return candidateseal.Input{}, err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return candidateseal.Input{}, errors.New("source root is not a directory")
	}
	command := exec.Command("git", "-C", root, "ls-files", "-z", "-t", "--cached", "--others", "--exclude-standard")
	listed, err := command.Output()
	if err != nil {
		return candidateseal.Input{}, err
	}
	paths := bytes.Split(listed, []byte{0})
	sort.Slice(paths, func(i, j int) bool { return bytes.Compare(paths[i], paths[j]) < 0 })

	reserved := make(map[string]struct{}, 3)
	subjectPaths := make([]string, 0, 3)
	for _, name := range []string{binaryPath, configPath, manifestPath} {
		absolute, absoluteErr := gateACanonicalSubjectPath(name)
		if absoluteErr != nil {
			return candidateseal.Input{}, absoluteErr
		}
		clean := filepath.Clean(absolute)
		relative, relativeErr := filepath.Rel(root, clean)
		if relativeErr != nil || relative == "." ||
			(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
			return candidateseal.Input{}, errors.New("candidate subjects must stay outside the source tree")
		}
		reserved[clean] = struct{}{}
		subjectPaths = append(subjectPaths, clean)
	}
	files := make([]candidateseal.SourceFile, 0, len(paths))
	for _, raw := range paths {
		if len(raw) == 0 {
			continue
		}
		if len(raw) < 3 || raw[1] != ' ' {
			return candidateseal.Input{}, errors.New("invalid git source entry")
		}
		if raw[0] == 'S' {
			continue
		}
		relative := filepath.ToSlash(string(raw[2:]))
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		if _, collides := reserved[filepath.Clean(absolute)]; collides {
			return candidateseal.Input{}, errors.New("candidate subjects must stay outside the source tree")
		}
		content, info, readErr := readGateAStableRegularFile(absolute, false)
		if readErr != nil {
			return candidateseal.Input{}, readErr
		}
		files = append(files, candidateseal.SourceFile{
			Path: relative, Mode: uint32(info.Mode().Perm()), Content: content,
		})
	}
	binary, err := readGateARegularFile(subjectPaths[0], false)
	if err != nil {
		return candidateseal.Input{}, err
	}
	config, err := readGateARegularFile(subjectPaths[1], true)
	if err != nil {
		return candidateseal.Input{}, err
	}
	if len(strings.TrimSpace(string(config))) == 0 {
		return candidateseal.Input{}, errors.New("effective config is empty")
	}
	return candidateseal.Input{SourceFiles: files, Binary: binary, EffectiveConfig: config}, nil
}

func gateACanonicalSubjectPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err = os.Lstat(absolute); err == nil {
		return filepath.EvalSymlinks(absolute)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func readGateARegularFile(path string, private bool) ([]byte, error) {
	content, _, err := readGateAStableRegularFile(path, private)
	return content, err
}

func readGateAStableRegularFile(path string, private bool) ([]byte, os.FileInfo, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, nil, errors.New("candidate subject is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, nil, errors.New("candidate subject changed during open")
	}
	if private && opened.Mode().Perm() != 0o400 {
		return nil, nil, errors.New("effective config is not owner-read-only")
	}
	first, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, err
	}
	second, err := io.ReadAll(file)
	if err != nil || !bytes.Equal(first, second) {
		return nil, nil, errors.New("candidate subject changed during read")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(opened, after) || opened.Size() != after.Size() ||
		!opened.ModTime().Equal(after.ModTime()) || opened.Mode() != after.Mode() {
		return nil, nil, errors.New("candidate subject changed during read")
	}
	return first, opened, nil
}

func writeGateAManifest(path string, content []byte) (err error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".gate-a-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err = temporary.Chmod(0o400); err != nil {
		return err
	}
	if _, err = temporary.Write(content); err != nil {
		return err
	}
	if err = temporary.Sync(); err != nil {
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Link(temporaryPath, path)
}
