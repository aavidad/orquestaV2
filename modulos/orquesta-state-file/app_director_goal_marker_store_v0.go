package orquestastatefile

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type appDirectorGoalMarkerDocumentV0 struct {
	SchemaVersion string                           `json:"schema_version"`
	RunRef        string                           `json:"run_ref"`
	GoalRef       string                           `json:"goal_ref,omitempty"`
	Marker        orquestagoal.GoalWorkRunMarkerV0 `json:"marker"`
}

func (store *StoreV0) SaveGoalWorkRunMarkerV0(
	ctx context.Context,
	marker orquestagoal.GoalWorkRunMarkerV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if err != nil {
		return invalidErrorV0("app_director_goal_marker", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveAppDirectorGoalMarkerLockedV0(normalized)
}

func (store *StoreV0) LoadGoalWorkRunMarkerV0(
	ctx context.Context,
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return orquestagoal.GoalWorkRunMarkerV0{}, invalidErrorV0("run_ref", "run_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[appDirectorGoalMarkerDocumentV0](store.appDirectorGoalMarkerPathV0(runRef))
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	if !ok {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker", "marcador goal-first no encontrado")
	}
	marker, err := validateAppDirectorGoalMarkerDocumentV0(document, runRef)
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	return marker, nil
}

func (store *StoreV0) ListGoalWorkRunMarkersV0(
	ctx context.Context,
	request orquestagoal.GoalWorkRunMarkerListRequestV0,
) ([]orquestagoal.GoalWorkRunMarkerV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request = orquestagoal.NormalizeGoalWorkRunMarkerListRequestV0(request)
	store.mu.Lock()
	defer store.mu.Unlock()
	var markers []orquestagoal.GoalWorkRunMarkerV0
	var err error
	if len(request.RunRefs) > 0 {
		markers, err = store.listAppDirectorGoalMarkersByRunRefsLockedV0(ctx, request)
	} else {
		markers, err = store.listAppDirectorGoalMarkersFromDirLockedV0(ctx, request)
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(markers, func(i, j int) bool {
		return strings.TrimSpace(markers[i].RunRef) < strings.TrimSpace(markers[j].RunRef)
	})
	if request.MaxItems > 0 && len(markers) > request.MaxItems {
		markers = markers[:request.MaxItems]
	}
	if markers == nil {
		return []orquestagoal.GoalWorkRunMarkerV0{}, nil
	}
	return markers, nil
}

func (store *StoreV0) saveAppDirectorGoalMarkerLockedV0(
	marker orquestagoal.GoalWorkRunMarkerV0,
) error {
	runRef := normalizeRefV0(marker.RunRef)
	return writeJSONAtomicV0(store.appDirectorGoalMarkerPathV0(runRef), appDirectorGoalMarkerDocumentV0{
		SchemaVersion: orquestagoal.GoalWorkRunMarkerSchemaV0,
		RunRef:        runRef,
		GoalRef:       strings.TrimSpace(marker.GoalRef),
		Marker:        marker,
	})
}

func (store *StoreV0) listAppDirectorGoalMarkersByRunRefsLockedV0(
	ctx context.Context,
	request orquestagoal.GoalWorkRunMarkerListRequestV0,
) ([]orquestagoal.GoalWorkRunMarkerV0, error) {
	out := make([]orquestagoal.GoalWorkRunMarkerV0, 0, len(request.RunRefs))
	for _, runRef := range request.RunRefs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		marker, ok, err := store.readAppDirectorGoalMarkerFileLockedV0(runRef)
		if err != nil {
			return nil, err
		}
		if !ok || !orquestagoal.GoalWorkRunMarkerMatchesListRequestV0(marker, request) {
			continue
		}
		out = append(out, marker)
	}
	return out, nil
}

func (store *StoreV0) listAppDirectorGoalMarkersFromDirLockedV0(
	ctx context.Context,
	request orquestagoal.GoalWorkRunMarkerListRequestV0,
) ([]orquestagoal.GoalWorkRunMarkerV0, error) {
	dir := filepath.Join(store.rootDir, appDirectorGoalMarkersDirV0)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]orquestagoal.GoalWorkRunMarkerV0, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		document, ok, err := readJSONFileV0[appDirectorGoalMarkerDocumentV0](filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		marker, err := validateAppDirectorGoalMarkerDocumentV0(document, document.RunRef)
		if err != nil {
			return nil, err
		}
		if !orquestagoal.GoalWorkRunMarkerMatchesListRequestV0(marker, request) {
			continue
		}
		out = append(out, marker)
	}
	return out, nil
}

func (store *StoreV0) readAppDirectorGoalMarkerFileLockedV0(
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, bool, error) {
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return orquestagoal.GoalWorkRunMarkerV0{}, false, nil
	}
	document, ok, err := readJSONFileV0[appDirectorGoalMarkerDocumentV0](store.appDirectorGoalMarkerPathV0(runRef))
	if err != nil || !ok {
		return orquestagoal.GoalWorkRunMarkerV0{}, ok, err
	}
	marker, err := validateAppDirectorGoalMarkerDocumentV0(document, runRef)
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, false, err
	}
	return marker, true, nil
}

func validateAppDirectorGoalMarkerDocumentV0(
	document appDirectorGoalMarkerDocumentV0,
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	if document.SchemaVersion != orquestagoal.GoalWorkRunMarkerSchemaV0 {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker.schema_version", "schema_version invalida")
	}
	if normalizeRefV0(document.RunRef) != normalizeRefV0(runRef) {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker.ref", "ref inconsistente")
	}
	marker, err := orquestagoal.NewGoalWorkRunMarkerV0(document.Marker)
	if err != nil {
		return orquestagoal.GoalWorkRunMarkerV0{}, err
	}
	if normalizeRefV0(marker.RunRef) != normalizeRefV0(runRef) {
		return orquestagoal.GoalWorkRunMarkerV0{}, storeErrorV0("app_director_goal_marker.state_ref", "ref interna inconsistente")
	}
	return marker, nil
}
