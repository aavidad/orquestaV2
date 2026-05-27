package orquestaruntimecodexdelivery

import (
	pathpkg "path"
	"strings"
)

func codexReviewGateGlobMatchV0(pattern string, path string) bool {
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.Contains(pattern, "**") {
		return codexReviewGateGlobstarMatchV0(pattern, path)
	}
	matched, err := pathpkg.Match(pattern, path)
	return err == nil && matched
}

func codexReviewGateGlobstarMatchV0(pattern string, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	memo := map[[2]int]bool{}
	var match func(int, int) bool
	match = func(patternIndex int, pathIndex int) bool {
		key := [2]int{patternIndex, pathIndex}
		if value, ok := memo[key]; ok {
			return value
		}
		ok := false
		defer func() { memo[key] = ok }()
		if patternIndex == len(patternParts) {
			ok = pathIndex == len(pathParts)
			return ok
		}
		if patternParts[patternIndex] == "**" {
			ok = match(patternIndex+1, pathIndex)
			for next := pathIndex; !ok && next < len(pathParts); next++ {
				ok = match(patternIndex+1, next+1)
			}
			return ok
		}
		if pathIndex >= len(pathParts) {
			return false
		}
		matched, err := pathpkg.Match(patternParts[patternIndex], pathParts[pathIndex])
		ok = err == nil && matched && match(patternIndex+1, pathIndex+1)
		return ok
	}
	return match(0, 0)
}
