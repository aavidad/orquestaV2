package orquestadomainworkfile

import (
	"fmt"
	"path/filepath"
	"strings"
)

func normalizeDomainWorkFileDirV0(dir string) (string, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "." || dir == "" || !filepath.IsAbs(dir) {
		return "", fmt.Errorf("orquesta_domain_work_file: dir_invalid")
	}
	return dir, nil
}
