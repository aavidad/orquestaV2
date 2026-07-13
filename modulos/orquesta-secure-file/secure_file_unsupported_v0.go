//go:build !linux

package orquestasecurefile

import "os"

func OpenDirectoryV0(string, DirectoryOptionsV0) (*os.File, error) {
	return nil, ErrUnsupportedPlatformV0
}

func ReadFileAtV0(*os.File, string, FileOptionsV0) ([]byte, error) {
	return nil, ErrUnsupportedPlatformV0
}

func ReadFileBeneathV0(string, string, FileOptionsV0) ([]byte, error) {
	return nil, ErrUnsupportedPlatformV0
}

func CreateFileIfAbsentAtV0(*os.File, string, []byte, FileOptionsV0) ([]byte, bool, error) {
	return nil, false, ErrUnsupportedPlatformV0
}
