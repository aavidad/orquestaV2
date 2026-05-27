package orquestadirectoragentfilesource

import (
	"context"
	"io"
	"os"
	"strings"
)

type OSDirectorAgentDecisionFileReaderV0 struct{}

var _ DirectorAgentDecisionFileReaderPortV0 = OSDirectorAgentDecisionFileReaderV0{}

func (reader OSDirectorAgentDecisionFileReaderV0) ReadDirectorAgentDecisionFileV0(
	ctx context.Context,
	path string,
	maxBytes int,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, DirectorAgentFileSourceIssueV0{Field: "path"}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if maxBytes <= 0 {
		return io.ReadAll(file)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, DirectorAgentFileSourceIssueV0{Field: "file_size"}
	}
	return data, nil
}
