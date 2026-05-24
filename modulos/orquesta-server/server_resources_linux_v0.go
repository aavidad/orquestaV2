//go:build linux

package orquestaserver

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func serverResourceRSSBytesV0() (int64, bool) {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, false
	}
	pages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return pages * int64(os.Getpagesize()), true
}

func serverResourceDisksV0(paths []string) ([]ServerDiskResourceV0, []ServerResourceIssueV0) {
	out := make([]ServerDiskResourceV0, 0, len(paths))
	issues := []ServerResourceIssueV0{}
	for _, path := range paths {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(path, &stat); err != nil {
			out = append(out, ServerDiskResourceV0{
				Path:      path,
				Status:    "error",
				ErrorCode: "statfs_error",
			})
			issues = append(issues, ServerResourceIssueV0{
				Code:    "disk_statfs_error",
				Field:   path,
				Message: err.Error(),
			})
			continue
		}
		blockSize := uint64(stat.Bsize)
		total := stat.Blocks * blockSize
		free := stat.Bfree * blockSize
		available := stat.Bavail * blockSize
		used := total - free
		percent := 0
		if total > 0 {
			percent = int((used * 100) / total)
		}
		out = append(out, ServerDiskResourceV0{
			Path:           path,
			TotalBytes:     total,
			FreeBytes:      free,
			AvailableBytes: available,
			UsedBytes:      used,
			UsedPercent:    percent,
			Status:         "ok",
		})
	}
	if out == nil {
		return []ServerDiskResourceV0{}, issues
	}
	return out, issues
}
