package orquestaruntimecodexdelivery

import (
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const codexProgressActionTailBytesV0 int64 = 64 * 1024

func codexProgressActionSignatureV0(
	descriptor CodexReceiptDescriptorV0,
) string {
	dir := filepath.Dir(strings.TrimSpace(descriptor.AckPath))
	if dir == "." || dir == "" {
		return ""
	}
	for _, name := range []string{
		orquestaruntimecodex.CodexStderrFileNameV0,
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
	} {
		signature := codexProgressActionSignatureFromFileV0(filepath.Join(dir, name))
		if signature != "" {
			return signature
		}
	}
	return ""
}

func codexProgressActionSignatureFromFileV0(path string) string {
	data, ok := codexProgressReadTailV0(path, codexProgressActionTailBytesV0)
	if !ok {
		return ""
	}
	return codexProgressActionSignatureFromTextV0(string(data))
}

func codexProgressReadTailV0(path string, maxBytes int64) ([]byte, bool) {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return nil, false
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	size := info.Size()
	offset := int64(0)
	if size > maxBytes {
		offset = size - maxBytes
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return nil, false
	}
	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

func codexProgressActionSignatureFromTextV0(text string) string {
	lines := codexProgressActionNonEmptyLinesV0(text)
	if len(lines) == 0 {
		return ""
	}
	if diffSignature := codexProgressDiffActionSignatureV0(lines); diffSignature != "" {
		return diffSignature
	}
	return "action-sig-tail-" + codexProgressActionHashV0(codexProgressActionTailWindowV0(lines, 24))
}

func codexProgressActionNonEmptyLinesV0(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, raw := range rawLines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func codexProgressDiffActionSignatureV0(lines []string) string {
	paths := make([]string, 0, 16)
	for _, line := range lines {
		if !strings.HasPrefix(line, "diff --git ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if path := codexProgressNormalizeDiffPathV0(fields[2]); path != "" {
			paths = append(paths, path)
		}
		if path := codexProgressNormalizeDiffPathV0(fields[3]); path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return ""
	}
	sort.Strings(paths)
	compact := paths[:0]
	var previous string
	for _, path := range paths {
		if path == previous {
			continue
		}
		compact = append(compact, path)
		previous = path
	}
	return "action-sig-diff-" + codexProgressActionHashV0(compact)
}

func codexProgressNormalizeDiffPathV0(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	if index := strings.LastIndex(path, "/project/"); index >= 0 {
		path = path[index+len("/project/"):]
	}
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	compact := make([]string, 0, 3)
	for _, part := range parts {
		part = codexProgressActionTokenV0(part)
		if part == "" || part == "." {
			continue
		}
		compact = append(compact, part)
	}
	if len(compact) > 3 {
		compact = compact[len(compact)-3:]
	}
	return strings.Join(compact, "/")
}

func codexProgressActionTailWindowV0(lines []string, max int) []string {
	if max <= 0 || len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func codexProgressActionTokenV0(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '_', r == '-':
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func codexProgressActionHashV0(values []string) string {
	hash := fnv.New32a()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return strconv.FormatUint(uint64(hash.Sum32()), 36)
}
