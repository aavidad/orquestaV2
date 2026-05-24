package orquestaautoprogramming

import "strings"

const (
	AutoprogrammingReviewGateDefaultMaxLinesV0 = 300
	AutoprogrammingReviewGateACKCompletedV0    = "completed"
)

type AutoprogrammingReviewGateInputV0 struct {
	ACK             AutoprogrammingReviewGateACKV0
	Ack             AutoprogrammingReviewGateACKV0
	RequiredTests   []string
	Tests           []AutoprogrammingReviewGateTestV0
	TestRuns        []AutoprogrammingReviewGateTestV0
	TestResults     []AutoprogrammingReviewGateTestV0
	Files           []AutoprogrammingReviewGateFileV0
	WriteSet        []string
	AllowedFiles    []string
	MaxLinesPerFile int
	MaxFileLines    int
}

type AutoprogrammingReviewGateACKV0 struct {
	Present bool
	Status  string
}

type AutoprogrammingReviewGateAckV0 = AutoprogrammingReviewGateACKV0

type AutoprogrammingReviewGateTestV0 struct {
	Command string
	Passed  bool
	Status  string
}

type AutoprogrammingReviewGateFileV0 struct {
	Path      string
	LineCount int
	Lines     int
}

type AutoprogrammingReviewGateResultV0 struct {
	Accepted          bool                                         `json:"accepted"`
	PreserveOutput    bool                                         `json:"preserve_output"`
	RequiresFollowup  bool                                         `json:"requires_followup"`
	RecommendedAction AutoprogrammingReviewGateRecommendedActionV0 `json:"recommended_action"`
	Issues            []AutoprogrammingReviewGateIssueV0           `json:"issues,omitempty"`
}

type AutoprogrammingReviewGateIssueV0 struct {
	Code    string
	Field   string
	Message string
}

func EvaluateAutoprogrammingReviewGateV0(
	input AutoprogrammingReviewGateInputV0,
) AutoprogrammingReviewGateResultV0 {
	var issues []AutoprogrammingReviewGateIssueV0
	ack := autoprogrammingReviewGateACKV0(input)
	if !ack.Present {
		issues = append(issues, autoprogrammingReviewGateIssueV0(
			"ack_missing",
			"ack",
			"ack requerido",
		))
	} else if strings.TrimSpace(ack.Status) != AutoprogrammingReviewGateACKCompletedV0 {
		issues = append(issues, autoprogrammingReviewGateIssueV0(
			"ack_not_completed",
			"ack.status",
			"ack debe estar completed",
		))
	}

	tests := autoprogrammingReviewGateTestsV0(input)
	issues = append(issues, autoprogrammingReviewGateRequiredTestIssuesV0(input.RequiredTests, tests)...)
	issues = append(issues, autoprogrammingReviewGateFailedTestIssuesV0(tests)...)
	issues = append(issues, autoprogrammingReviewGateFileIssuesV0(input.Files, autoprogrammingReviewGateMaxLinesV0(input))...)
	issues = append(issues, autoprogrammingReviewGateWriteSetIssuesV0(input)...)

	return autoprogrammingReviewGateResultV0(issues)
}

func autoprogrammingReviewGateACKV0(
	input AutoprogrammingReviewGateInputV0,
) AutoprogrammingReviewGateACKV0 {
	if input.ACK.Present || strings.TrimSpace(input.ACK.Status) != "" {
		return input.ACK
	}
	return input.Ack
}

func autoprogrammingReviewGateTestsV0(
	input AutoprogrammingReviewGateInputV0,
) []AutoprogrammingReviewGateTestV0 {
	tests := make([]AutoprogrammingReviewGateTestV0, 0, len(input.Tests)+len(input.TestRuns)+len(input.TestResults))
	tests = append(tests, input.Tests...)
	tests = append(tests, input.TestRuns...)
	tests = append(tests, input.TestResults...)
	return tests
}

func autoprogrammingReviewGateRequiredTestIssuesV0(
	requiredTests []string,
	tests []AutoprogrammingReviewGateTestV0,
) []AutoprogrammingReviewGateIssueV0 {
	var issues []AutoprogrammingReviewGateIssueV0
	byCommand := map[string]AutoprogrammingReviewGateTestV0{}
	for _, test := range tests {
		command := strings.TrimSpace(test.Command)
		if command != "" {
			byCommand[command] = test
		}
	}
	for _, command := range compactStringsV0(requiredTests) {
		test, ok := byCommand[command]
		if !ok {
			issues = append(issues, autoprogrammingReviewGateIssueV0(
				"required_test_missing",
				"tests",
				"test obligatorio no ejecutado: "+command,
			))
			continue
		}
		if !autoprogrammingReviewGateTestPassedV0(test) {
			issues = append(issues, autoprogrammingReviewGateIssueV0(
				"required_test_failed",
				"tests",
				"test obligatorio fallido: "+command,
			))
		}
	}
	return issues
}

func autoprogrammingReviewGateFailedTestIssuesV0(
	tests []AutoprogrammingReviewGateTestV0,
) []AutoprogrammingReviewGateIssueV0 {
	var issues []AutoprogrammingReviewGateIssueV0
	seenFailed := map[string]struct{}{}
	for _, test := range tests {
		command := strings.TrimSpace(test.Command)
		if command == "" || autoprogrammingReviewGateTestPassedV0(test) {
			continue
		}
		if _, ok := seenFailed[command]; ok {
			continue
		}
		seenFailed[command] = struct{}{}
		issues = append(issues, autoprogrammingReviewGateIssueV0(
			"test_failed",
			"tests",
			"test fallido: "+command,
		))
	}
	return issues
}

func autoprogrammingReviewGateTestPassedV0(
	test AutoprogrammingReviewGateTestV0,
) bool {
	status := strings.TrimSpace(test.Status)
	if status == "" {
		return test.Passed
	}
	return status == "passed" || status == "success"
}

func autoprogrammingReviewGateFileIssuesV0(
	files []AutoprogrammingReviewGateFileV0,
	maxLines int,
) []AutoprogrammingReviewGateIssueV0 {
	var issues []AutoprogrammingReviewGateIssueV0
	for _, file := range files {
		lineCount := autoprogrammingReviewGateLineCountV0(file)
		if lineCount <= maxLines {
			continue
		}
		issues = append(issues, autoprogrammingReviewGateIssueV0(
			"file_too_large",
			"files",
			"fichero supera lineas maximas: "+strings.TrimSpace(file.Path),
		))
	}
	return issues
}

func autoprogrammingReviewGateWriteSetIssuesV0(
	input AutoprogrammingReviewGateInputV0,
) []AutoprogrammingReviewGateIssueV0 {
	allowed := autoprogrammingReviewGateAllowedFilesV0(input)
	if len(allowed) == 0 {
		return []AutoprogrammingReviewGateIssueV0{autoprogrammingReviewGateIssueV0(
			"write_set_missing",
			"write_set",
			"write-set requerido",
		)}
	}
	allowedSet := map[string]struct{}{}
	for _, path := range allowed {
		allowedSet[path] = struct{}{}
	}
	var issues []AutoprogrammingReviewGateIssueV0
	for _, file := range input.Files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			issues = append(issues, autoprogrammingReviewGateIssueV0(
				"file_path_missing",
				"files.path",
				"ruta de fichero requerida",
			))
			continue
		}
		if _, ok := allowedSet[path]; !ok &&
			!autoprogrammingReviewGatePathAllowedByWriteSetV0(path, allowed) {
			issues = append(issues, autoprogrammingReviewGateIssueV0(
				"file_outside_write_set",
				"files.path",
				"fichero fuera del write-set: "+path,
			))
		}
	}
	return issues
}

func autoprogrammingReviewGateAllowedFilesV0(
	input AutoprogrammingReviewGateInputV0,
) []string {
	if len(input.WriteSet) > 0 {
		return compactStringsV0(input.WriteSet)
	}
	return compactStringsV0(input.AllowedFiles)
}

func autoprogrammingReviewGateLineCountV0(
	file AutoprogrammingReviewGateFileV0,
) int {
	if file.LineCount != 0 {
		return file.LineCount
	}
	return file.Lines
}

func autoprogrammingReviewGateMaxLinesV0(
	input AutoprogrammingReviewGateInputV0,
) int {
	if input.MaxLinesPerFile > 0 {
		return input.MaxLinesPerFile
	}
	if input.MaxFileLines > 0 {
		return input.MaxFileLines
	}
	return AutoprogrammingReviewGateDefaultMaxLinesV0
}

func autoprogrammingReviewGateIssueV0(
	code string,
	field string,
	message string,
) AutoprogrammingReviewGateIssueV0 {
	return AutoprogrammingReviewGateIssueV0{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
