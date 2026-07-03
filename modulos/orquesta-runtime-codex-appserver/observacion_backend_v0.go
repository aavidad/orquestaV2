package orquestaruntimecodexappserver

import "strings"

type SnapshotObservacionBackendV0 struct {
	ActionRequested          string
	SessionObserved          bool
	OwnerMarkerObserved      bool
	OwnerMarkerValid         bool
	SocketObserved           bool
	PreflightOK              bool
	PanePIDLive              bool
	ProcessBySocketObserved  bool
	ProcessByRuntimeObserved bool
	IssueCode                string
}

func RecolectarObservacionBackendV0(snapshot SnapshotObservacionBackendV0) ObservacionBackendV0 {
	ownerObserved := snapshot.OwnerMarkerObserved || snapshot.OwnerMarkerValid
	socketObserved := snapshot.SocketObserved || snapshot.PreflightOK
	return ObservacionBackendV0{
		ActionRequested:          normalizarAccionBackendV0(snapshot.ActionRequested),
		SessionObserved:          snapshot.SessionObserved,
		OwnerMarkerObserved:      ownerObserved,
		OwnerMarkerValid:         snapshot.OwnerMarkerValid,
		SocketObserved:           socketObserved,
		PreflightOK:              snapshot.PreflightOK,
		PanePIDLive:              snapshot.PanePIDLive,
		ProcessBySocketObserved:  snapshot.ProcessBySocketObserved,
		ProcessByRuntimeObserved: snapshot.ProcessByRuntimeObserved,
		IssueCode:                strings.TrimSpace(snapshot.IssueCode),
	}
}

func normalizarAccionBackendV0(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case AccionBackendEnsureV0:
		return AccionBackendEnsureV0
	case AccionBackendShutdownV0:
		return AccionBackendShutdownV0
	case AccionBackendCleanupV0:
		return AccionBackendCleanupV0
	default:
		return strings.TrimSpace(action)
	}
}
