package orquestaservershutdown

func missingRequiredServerShutdownDepsV0(
	deps ServerShutdownDepsV0,
) (ServerShutdownResultV0, bool) {
	if deps.QueueReader == nil {
		return missingServerShutdownDepV0(ServerShutdownStatusNoQueueReaderV0), true
	}
	if deps.RunControlReader == nil {
		return missingServerShutdownDepV0(ServerShutdownStatusNoRunControlReaderV0), true
	}
	if deps.RunControlWriter == nil {
		return missingServerShutdownDepV0(ServerShutdownStatusNoRunControlWriterV0), true
	}
	return ServerShutdownResultV0{}, false
}
