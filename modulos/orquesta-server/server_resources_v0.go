package orquestaserver

import (
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	ServerResourcesEndpointV0            = "/api/v0/server/resources"
	ServerRoutesEndpointV0               = "/api/v0/routes"
	ServerResourcesSchemaVersionV0       = "orquesta_server_resources.v0"
	ServerRouteManifestSchemaVersionV0   = "orquesta_route_manifest.v0"
	ServerRouteManifestStatusAvailableV0 = "available"
)

type ServerResourcesV0 struct {
	SchemaVersion string                  `json:"schema_version"`
	Status        string                  `json:"status"`
	CollectedAt   string                  `json:"collected_at"`
	PID           int                     `json:"pid"`
	Process       ServerProcessV0         `json:"process"`
	Disks         []ServerDiskResourceV0  `json:"disks"`
	RouteManifest ServerRouteManifestV0   `json:"route_manifest"`
	Issues        []ServerResourceIssueV0 `json:"issues,omitempty"`
}

type ServerRouteManifestV0 struct {
	SchemaVersion string                  `json:"schema_version"`
	Status        string                  `json:"status"`
	Routes        []ServerRouteResourceV0 `json:"routes"`
	EvidenceRefs  []string                `json:"evidence_refs,omitempty"`
}

type ServerRouteResourceV0 struct {
	Ref               string   `json:"ref"`
	Pattern           string   `json:"pattern"`
	Kind              string   `json:"kind"`
	Owner             string   `json:"owner"`
	Methods           []string `json:"methods"`
	SecurityProfile   string   `json:"security_profile"`
	ContractRefs      []string `json:"contract_refs,omitempty"`
	Mounted           bool     `json:"mounted"`
	ShadowsPrefixRefs []string `json:"shadows_prefix_refs,omitempty"`
}

type ServerProcessV0 struct {
	GoAllocBytes      uint64 `json:"go_alloc_bytes"`
	GoHeapAllocBytes  uint64 `json:"go_heap_alloc_bytes"`
	GoHeapInUseBytes  uint64 `json:"go_heap_in_use_bytes"`
	GoSysBytes        uint64 `json:"go_sys_bytes"`
	GoTotalAllocBytes uint64 `json:"go_total_alloc_bytes"`
	NumGC             uint32 `json:"num_gc"`
	NumGoroutine      int    `json:"num_goroutine"`
	RSSBytes          int64  `json:"rss_bytes,omitempty"`
}

type ServerDiskResourceV0 struct {
	Path           string `json:"path"`
	TotalBytes     uint64 `json:"total_bytes,omitempty"`
	FreeBytes      uint64 `json:"free_bytes,omitempty"`
	AvailableBytes uint64 `json:"available_bytes,omitempty"`
	UsedBytes      uint64 `json:"used_bytes,omitempty"`
	UsedPercent    int    `json:"used_percent,omitempty"`
	Status         string `json:"status"`
	ErrorCode      string `json:"error_code,omitempty"`
}

type ServerResourceIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewServerResourcesV0(state StateV0, now time.Time) ServerResourcesV0 {
	return NewServerResourcesWithRoutesV0(state, now, nil)
}

func NewServerResourcesWithRoutesV0(state StateV0, now time.Time, routes []ServerRouteResourceV0) ServerResourcesV0 {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	disks, issues := serverResourceDisksV0(serverResourcePathsV0(state))
	rss, rssOK := serverResourceRSSBytesV0()
	if !rssOK {
		issues = append(issues, ServerResourceIssueV0{
			Code:    "rss_no_disponible",
			Field:   "process.rss_bytes",
			Message: "rss no disponible en este entorno",
		})
	}
	return ServerResourcesV0{
		SchemaVersion: ServerResourcesSchemaVersionV0,
		Status:        "ok",
		CollectedAt:   now.UTC().Format(time.RFC3339),
		PID:           firstNonZeroIntV0(state.PID, os.Getpid()),
		Process: ServerProcessV0{
			GoAllocBytes:      mem.Alloc,
			GoHeapAllocBytes:  mem.HeapAlloc,
			GoHeapInUseBytes:  mem.HeapInuse,
			GoSysBytes:        mem.Sys,
			GoTotalAllocBytes: mem.TotalAlloc,
			NumGC:             mem.NumGC,
			NumGoroutine:      runtime.NumGoroutine(),
			RSSBytes:          rss,
		},
		Disks: disks,
		RouteManifest: ServerRouteManifestV0{
			SchemaVersion: ServerRouteManifestSchemaVersionV0,
			Status:        ServerRouteManifestStatusAvailableV0,
			Routes:        compactServerRouteResourcesV0(routes),
			EvidenceRefs:  serverRouteResourceEvidenceRefsV0(routes),
		},
		Issues: compactServerResourceIssuesV0(issues),
	}
}

func serverResourcePathsV0(state StateV0) []string {
	return compactServerResourceStringsV0([]string{
		state.ProjectWorkDir,
		state.RuntimeWorkDir,
	})
}

func compactServerResourceStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func compactServerResourceIssuesV0(values []ServerResourceIssueV0) []ServerResourceIssueV0 {
	out := make([]ServerResourceIssueV0, 0, len(values))
	for _, value := range values {
		value.Code = strings.TrimSpace(value.Code)
		value.Field = strings.TrimSpace(value.Field)
		value.Message = strings.TrimSpace(value.Message)
		if value.Code == "" {
			continue
		}
		out = append(out, value)
	}
	if out == nil {
		return []ServerResourceIssueV0{}
	}
	return out
}

func compactServerRouteResourcesV0(values []ServerRouteResourceV0) []ServerRouteResourceV0 {
	seen := map[string]bool{}
	out := make([]ServerRouteResourceV0, 0, len(values))
	for _, value := range values {
		value.Ref = strings.TrimSpace(value.Ref)
		value.Pattern = strings.TrimSpace(value.Pattern)
		value.Kind = strings.TrimSpace(value.Kind)
		value.Owner = strings.TrimSpace(value.Owner)
		value.SecurityProfile = strings.TrimSpace(value.SecurityProfile)
		value.Methods = compactServerResourceStringsV0(value.Methods)
		value.ContractRefs = compactServerResourceStringsV0(value.ContractRefs)
		value.ShadowsPrefixRefs = compactServerResourceStringsV0(value.ShadowsPrefixRefs)
		key := value.Ref + "\x00" + value.Pattern
		if value.Ref == "" || value.Pattern == "" || len(value.Methods) == 0 || seen[key] {
			continue
		}
		if value.Kind == "" {
			value.Kind = "exact"
		}
		if value.Owner == "" {
			value.Owner = "unknown"
		}
		if value.SecurityProfile == "" {
			value.SecurityProfile = "unknown"
		}
		seen[key] = true
		out = append(out, value)
	}
	if out == nil {
		return []ServerRouteResourceV0{}
	}
	return out
}

func serverRouteResourceEvidenceRefsV0(values []ServerRouteResourceV0) []string {
	routes := compactServerRouteResourcesV0(values)
	refs := make([]string, 0, len(routes))
	for _, route := range routes {
		refs = append(refs, route.Ref+"-mounted-"+route.Owner)
	}
	if refs == nil {
		return []string{}
	}
	return refs
}

func firstNonZeroIntV0(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
