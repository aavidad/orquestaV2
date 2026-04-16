package coordinacion

// ResolveProjectEffectivePath decide la ruta efectiva de un proyecto usando una pista de
// sesión actual y, en su defecto, una sesión histórica.
//
// Si la pista actual/reciente cae dentro de una worktree activa del proyecto, no se fuerza
// la resolución desde esa ruta.
func ResolveProjectEffectivePath(
	fallback, cwdHint string,
	cwdHintInsideActiveWorktree bool,
	sessionCWD string,
	sessionInsideActiveWorktree bool,
	hasRepoMarkers RepoMarkerResolver,
) string {
	fallback = normalizeRouteSelectorPath(fallback)
	if ruta := CandidateEffectiveProjectPath(cwdHint, cwdHintInsideActiveWorktree, hasRepoMarkers); ruta != "" {
		return ruta
	}
	if ruta := CandidateEffectiveProjectPath(sessionCWD, sessionInsideActiveWorktree, hasRepoMarkers); ruta != "" {
		return ruta
	}
	return fallback
}
