package autonomiapolicy

// ProjectAdmitsWorkerAutonomy decides whether an autonomous project can accept
// one more worker after applying the current worker count and the caller's
// own active assignment state.
func ProjectAdmitsWorkerAutonomy(enabled bool, maxWorkers, workers int, selfHasActiveAssignment, selfIsReserved bool) bool {
	if !enabled || maxWorkers <= 0 {
		return true
	}
	effectiveWorkers := workers
	if selfHasActiveAssignment && !selfIsReserved && effectiveWorkers > 0 {
		effectiveWorkers--
	}
	return effectiveWorkers < maxWorkers
}
