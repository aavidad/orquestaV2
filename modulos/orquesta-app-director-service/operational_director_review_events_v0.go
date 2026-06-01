package orquestaappdirectorservice

import (
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func operationalDirectorPlanStateEventReaderV0(
	ports StartAppDirectorPortsV0,
) orquestacionnucleoapp.RunEventReaderPortV0 {
	if ports.EventReader != nil {
		return ports.EventReader
	}
	reader, _ := ports.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	return reader
}
