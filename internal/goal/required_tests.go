package goal

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"strconv"
	"strings"
)

const (
	maxRequiredTestsPerWorkItem  = 32
	maxRequiredTestArguments     = 128
	maxRequiredTestArgumentSize  = 4 * 1024
	maxRequiredTestArgumentsSize = 64 * 1024
	maxRequiredTestRefSize       = 256
	maxRequiredTestCWDSize       = 1024
)

// RequiredTestSpec is immutable plan data. ToolRef selects a registered tool;
// Arguments are argv values, never a shell command string.
type RequiredTestSpec struct {
	ref              RequiredTestRef
	toolRef          ToolRef
	arguments        []string
	workingDirectory string
}

type RequiredTestSpecInput struct {
	Ref              RequiredTestRef
	ToolRef          ToolRef
	Arguments        []string
	WorkingDirectory string
}

func NewRequiredTestSpec(input RequiredTestSpecInput) (RequiredTestSpec, error) {
	spec := RequiredTestSpec{
		ref: input.Ref, toolRef: input.ToolRef,
		arguments:        append([]string(nil), input.Arguments...),
		workingDirectory: input.WorkingDirectory,
	}
	if err := validateRequiredTestSpec(spec); err != nil {
		return RequiredTestSpec{}, err
	}
	return spec, nil
}

func (spec RequiredTestSpec) Ref() RequiredTestRef     { return spec.ref }
func (spec RequiredTestSpec) ToolRef() ToolRef         { return spec.toolRef }
func (spec RequiredTestSpec) Arguments() []string      { return append([]string(nil), spec.arguments...) }
func (spec RequiredTestSpec) WorkingDirectory() string { return spec.workingDirectory }

// Digest returns the canonical digest of this exact test declaration.
func (spec RequiredTestSpec) Digest() string {
	digest := sha256.New()
	writeHashField(digest, "orquesta.required-test-spec.v1")
	writeHashField(digest, spec.ref.String())
	writeHashField(digest, spec.toolRef.String())
	writeHashField(digest, strconv.Itoa(len(spec.arguments)))
	for _, argument := range spec.arguments {
		writeHashField(digest, argument)
	}
	writeHashField(digest, spec.workingDirectory)
	return hex.EncodeToString(digest.Sum(nil))
}

// RequiredTestsDigest binds ordered test declarations to plan/evidence data.
func RequiredTestsDigest(specs []RequiredTestSpec) string {
	digest := sha256.New()
	writeHashField(digest, "orquesta.required-tests.v1")
	writeHashField(digest, strconv.Itoa(len(specs)))
	for _, spec := range specs {
		writeHashField(digest, spec.Digest())
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func validateRequiredTests(specs []RequiredTestSpec) error {
	if len(specs) > maxRequiredTestsPerWorkItem {
		return domainError(ErrorInvalidPlan, "required_tests_limit")
	}
	seen := make(map[RequiredTestRef]struct{}, len(specs))
	for _, spec := range specs {
		if err := validateRequiredTestSpec(spec); err != nil {
			return err
		}
		if _, duplicate := seen[spec.ref]; duplicate {
			return domainError(ErrorInvalidPlan, "duplicate_required_test_ref")
		}
		seen[spec.ref] = struct{}{}
	}
	return nil
}

func validateRequiredTestSpec(spec RequiredTestSpec) error {
	if !validRequiredTestRef(spec.ref) || len(spec.ref.String()) > maxRequiredTestRefSize {
		return domainError(ErrorInvalidRef, "required_test_ref")
	}
	if !validToolRef(spec.toolRef) || len(spec.toolRef.String()) > maxRequiredTestRefSize {
		return domainError(ErrorInvalidRef, "required_test_tool_ref")
	}
	if len(spec.arguments) > maxRequiredTestArguments {
		return domainError(ErrorInvalidPlan, "required_test_arguments_limit")
	}
	total := 0
	for _, argument := range spec.arguments {
		if strings.ContainsRune(argument, '\x00') || len(argument) > maxRequiredTestArgumentSize {
			return domainError(ErrorInvalidPlan, "required_test_argument")
		}
		total += len(argument)
		if total > maxRequiredTestArgumentsSize {
			return domainError(ErrorInvalidPlan, "required_test_arguments_size")
		}
	}
	if !validRequiredTestWorkingDirectory(spec.workingDirectory) {
		return domainError(ErrorInvalidPlan, "required_test_working_directory")
	}
	return nil
}

func validRequiredTestWorkingDirectory(value string) bool {
	if value == "" || len(value) > maxRequiredTestCWDSize || strings.TrimSpace(value) != value ||
		strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") || path.IsAbs(value) {
		return false
	}
	if value == "." {
		return true
	}
	cleaned := path.Clean(value)
	return cleaned == value && cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}

func cloneRequiredTests(specs []RequiredTestSpec) []RequiredTestSpec {
	if len(specs) == 0 {
		return nil
	}
	cloned := make([]RequiredTestSpec, len(specs))
	for index, spec := range specs {
		cloned[index] = spec
		cloned[index].arguments = append([]string(nil), spec.arguments...)
	}
	return cloned
}

func requiredTestsEqual(left, right []RequiredTestSpec) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ref != right[index].ref || left[index].toolRef != right[index].toolRef ||
			left[index].workingDirectory != right[index].workingDirectory ||
			!refsEqual(left[index].arguments, right[index].arguments) {
			return false
		}
	}
	return true
}
