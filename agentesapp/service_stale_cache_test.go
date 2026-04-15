package agentesapp

import (
	"testing"
	"time"
)

func TestGetOrBuildCompactDetailDevuelveStaleMientrasRevalida(t *testing.T) {
	prevTTL := compactDetailCacheTTL
	prevStale := compactDetailStaleWhileRevalidateTTL
	compactDetailCacheTTL = 50 * time.Millisecond
	compactDetailStaleWhileRevalidateTTL = time.Second
	t.Cleanup(func() {
		compactDetailCacheTTL = prevTTL
		compactDetailStaleWhileRevalidateTTL = prevStale
	})

	svc := NewService(nil, nil)
	svc.compactDetailCache["Codex2"] = cachedCompactDetail{
		detail:  &Detail{Row: Row{OpenTasks: 1}},
		expires: time.Now().UTC().Add(-50 * time.Millisecond),
	}

	started := make(chan struct{})
	release := make(chan struct{})
	fresh := &Detail{Row: Row{OpenTasks: 2}}

	detail, err := svc.getOrBuildCompactDetail("Codex2", func() (*Detail, error) {
		close(started)
		<-release
		return fresh, nil
	})
	if err != nil {
		t.Fatalf("getOrBuildCompactDetail: %v", err)
	}
	if detail == nil || detail.Row.OpenTasks != 1 {
		t.Fatalf("deberia devolver detalle stale mientras refresca, got=%+v", detail)
	}
	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("deberia lanzar refresh en background")
	}

	close(release)
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		svc.cacheMu.Lock()
		cached := svc.compactDetailCache["Codex2"]
		_, inflight := svc.compactDetailFlight["Codex2"]
		svc.cacheMu.Unlock()
		if !inflight && cached.detail != nil && cached.detail.Row.OpenTasks == 2 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("el refresh de compact detail no actualizo la cache")
}

func TestGetOrBuildActiveReanimationScheduleDevuelveStaleMientrasRevalida(t *testing.T) {
	prevTTL := activeReanimationScheduleCacheTTL
	prevStale := activeReanimationScheduleStaleWhileRevalidateTTL
	activeReanimationScheduleCacheTTL = 50 * time.Millisecond
	activeReanimationScheduleStaleWhileRevalidateTTL = time.Second
	t.Cleanup(func() {
		activeReanimationScheduleCacheTTL = prevTTL
		activeReanimationScheduleStaleWhileRevalidateTTL = prevStale
	})

	svc := NewService(nil, nil)
	key := "active:true:future:true"
	svc.reanimCache[key] = cachedReanimationSchedule{
		rows:    []ReanimationCandidate{{Name: "Codex1", OpenTasks: 1}},
		expires: time.Now().UTC().Add(-50 * time.Millisecond),
	}

	started := make(chan struct{})
	release := make(chan struct{})
	fresh := []ReanimationCandidate{{Name: "Codex1", OpenTasks: 2}}

	rows, err := svc.getOrBuildActiveReanimationSchedule(key, func() ([]ReanimationCandidate, error) {
		close(started)
		<-release
		return fresh, nil
	})
	if err != nil {
		t.Fatalf("getOrBuildActiveReanimationSchedule: %v", err)
	}
	if len(rows) != 1 || rows[0].OpenTasks != 1 {
		t.Fatalf("deberia devolver rows stale mientras refresca, got=%+v", rows)
	}
	select {
	case <-started:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("deberia lanzar refresh de reanimaciones en background")
	}

	close(release)
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		svc.cacheMu.Lock()
		cached := svc.reanimCache[key]
		_, inflight := svc.reanimFlight[key]
		svc.cacheMu.Unlock()
		if !inflight && len(cached.rows) == 1 && cached.rows[0].OpenTasks == 2 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("el refresh de reanimaciones no actualizo la cache")
}
