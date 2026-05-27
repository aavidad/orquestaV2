package orquestaopesconnector

import (
	"net/url"
	"strings"
)

func (client RESTClientV0) controlledURLV0(path string) (string, error) {
	path = strings.TrimSpace(path)
	if !opesControlledPathV0(path) {
		return "", connectorErrorV0{code: ErrOPESPathInvalidV0}
	}
	return client.baseURL + path, nil
}

func opesControlledPathV0(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	parsed, err := url.Parse(path)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || strings.Contains(parsed.Path, "\\") {
		return false
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == ".." {
			return false
		}
	}
	return true
}
