//go:build !linux

package orquestaserver

func serverResourceRSSBytesV0() (int64, bool) {
	return 0, false
}

func serverResourceDisksV0(paths []string) ([]ServerDiskResourceV0, []ServerResourceIssueV0) {
	out := make([]ServerDiskResourceV0, 0, len(paths))
	for _, path := range paths {
		out = append(out, ServerDiskResourceV0{
			Path:      path,
			Status:    "unknown",
			ErrorCode: "disk_stats_no_disponible",
		})
	}
	if out == nil {
		out = []ServerDiskResourceV0{}
	}
	return out, []ServerResourceIssueV0{{
		Code:    "disk_stats_no_disponible",
		Field:   "disks",
		Message: "estadisticas de disco no disponibles en este entorno",
	}}
}
