package orquestaappdirectorintake

func PrepareAppDirectorIntakeV0(
	request PrepareAppDirectorIntakeRequestV0,
) (AppDirectorIntakePreparedV0, error) {
	request = normalizePrepareAppDirectorIntakeRequestV0(request)
	if err := validatePrepareAppDirectorIntakeRequestV0(request); err != nil {
		return AppDirectorIntakePreparedV0{}, err
	}
	tasks := directorTasksFromAppSpecV0(request.AppSpec)
	if err := validateAppDirectorTasksV0(tasks); err != nil {
		return AppDirectorIntakePreparedV0{}, err
	}
	run, initialEvents, err := buildInitialDirectorIntakeRunV0(request, tasks)
	if err != nil {
		return AppDirectorIntakePreparedV0{}, err
	}
	provider := AppDirectorCandidateProviderV0{
		Task:        tasks[0],
		Tasks:       tasks,
		RequestedBy: request.RequestedBy,
	}
	return AppDirectorIntakePreparedV0{
		SchemaVersion:     AppDirectorIntakePreparedSchemaVersionV0,
		Run:               run,
		InitialEvents:     initialEvents,
		DirectorTask:      tasks[0],
		DirectorTasks:     append([]AppDirectorTaskV0(nil), tasks...),
		CandidateProvider: provider,
		EvidenceRefs: compactDirectorIntakeStringsV0([]string{
			"evidence-ref-app-director-intake-v0",
			"evidence-ref-" + safeDirectorIntakeRefPartV0(request.RunRef),
		}),
	}, nil
}
