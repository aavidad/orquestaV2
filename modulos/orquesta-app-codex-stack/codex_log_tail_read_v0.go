package orquestaappcodexstack

import (
	"io"
	"os"
	"strings"
)

const codexStackRuntimeLogTailMaxBytesV0 int64 = 64 * 1024

func codexStackReadTailFileV0(path string, maxBytes int64) ([]byte, bool) {
	if strings.TrimSpace(path) == "" || maxBytes <= 0 {
		return nil, false
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return nil, false
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	offset := int64(0)
	if info.Size() > maxBytes {
		offset = info.Size() - maxBytes
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil || len(data) == 0 {
		return nil, false
	}
	if int64(len(data)) > maxBytes {
		data = data[len(data)-int(maxBytes):]
	}
	return data, true
}
