package db

import "orquesta/coordinacion"

func CoordinationLockRepository() coordinacion.LockRepository {
	return CoordinationLockSQLRepository{}
}

func CoordinationWorktreeRepository() coordinacion.WorktreeRepository {
	return CoordinationWorktreeSQLRepository{}
}

func CoordinationProjectRepository() coordinacion.ProjectRepository {
	return CoordinationProjectSQLRepository{}
}

func CoordinationSessionRepository() coordinacion.SessionRepository {
	return CoordinationSessionSQLRepository{}
}

func CoordinationConfigRepository() coordinacion.ConfigRepository {
	return CoordinationConfigSQLRepository{}
}
