package application

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestProviderScoringRanksScoreConfidenceSamplesAndIdentity(t *testing.T) {
	now := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	policy, policyRef := providerScorePolicyFixture(t)
	scores := []ProviderModelScoreObservation{
		providerScoreFixture(t, now, policyRef, "provider:z", "model:z", 8_000, 9_000, 20),
		providerScoreFixture(t, now, policyRef, "provider:c", "model:c", 9_000, 7_000, 40),
		providerScoreFixture(t, now, policyRef, "provider:b", "model:b", 9_000, 8_000, 10),
		providerScoreFixture(t, now, policyRef, "provider:a", "model:aa", 9_000, 8_000, 10),
		providerScoreFixture(t, now, policyRef, "provider:d", "model:d", 9_000, 8_000, 30),
		providerScoreFixture(t, now, policyRef, "provider:a", "model:a", 9_000, 8_000, 10),
	}
	ranking, err := rankProviderModelScores(now, policyRef, policy, scores)
	want := []ProviderRouteCandidate{{"provider:d", "model:d"}, {"provider:a", "model:a"}, {"provider:a", "model:aa"}, {"provider:b", "model:b"}, {"provider:c", "model:c"}, {"provider:z", "model:z"}}
	if err != nil || !reflect.DeepEqual(ranking.RankedCandidates, want) || len(ranking.QualifiedScores) != len(want) {
		t.Fatalf("ranking=%+v want=%+v err=%v", ranking, want, err)
	}

	stale := providerScoreFixture(t, now, policyRef, "provider:stale", "model:s", 10_000, 10_000, 100)
	stale.ExpiresAt = now
	weak := providerScoreFixture(t, now, policyRef, "provider:weak", "model:w", 10_000, 6_999, 100)
	low := providerScoreFixture(t, now, policyRef, "provider:low", "model:l", 10_000, 9_000, 9)
	ranking, err = rankProviderModelScores(now, policyRef, policy, []ProviderModelScoreObservation{weak, stale, low})
	wantRejected := []ProviderScoreRejection{{low.Candidate, ProviderScoreSamplesInsufficient}, {stale.Candidate, ProviderScoreStale}, {weak.Candidate, ProviderScoreConfidenceInsufficient}}
	if err != nil || !reflect.DeepEqual(ranking.Rejections, wantRejected) {
		t.Fatalf("rejections=%+v want=%+v err=%v", ranking.Rejections, wantRejected, err)
	}
}

func TestProviderScoringReaderOwnsCompleteCorpus(t *testing.T) {
	now := time.Date(2026, 8, 21, 11, 0, 0, 0, time.UTC)
	policy, policyRef := providerScorePolicyFixture(t)
	primary := providerScoreFixture(t, now, policyRef, "provider:primary", "model:primary", 9_000, 9_000, 30)
	fallback := providerScoreFixture(t, now, policyRef, "provider:fallback", "model:fallback", 8_000, 9_000, 30)
	reader := providerScoreReaderFixture(policyRef, policy, primary, fallback)
	if len(strings.TrimPrefix(policyRef, providerScorePolicyRefPrefix)) != 64 || len(strings.TrimPrefix(primary.EvidenceReceiptRef, providerScoreEvidenceRefPrefix)) != 64 {
		t.Fatal("policy and evidence refs must retain the full sha256")
	}
	clock := &providerScoreClock{now: now}
	query := mustProviderScoringQuery(t, clock, reader)
	catalog := providerScoreCatalog(now, primary.Candidate, fallback.Candidate)
	request := providerScoreRequestFixture(policyRef)
	decision, err := query.RouteProviderModelByScore(context.Background(), catalog, request)
	if err != nil || !decision.Route.Selected || decision.Route.Candidate != primary.Candidate ||
		len(decision.EvidenceReceiptRefs) != 2 || reader.limit != providerScoreMaximumObservations {
		t.Fatalf("decision=%+v reader=%+v err=%v", decision, reader, err)
	}
}

func TestProviderScoringRejectsUnavailableOrDivergentAuthority(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	policy, policyRef := providerScorePolicyFixture(t)
	valid := providerScoreFixture(t, now, policyRef, "provider:one", "model:one", 8_000, 8_000, 20)
	catalog := providerScoreCatalog(now, valid.Candidate)
	tests := map[string]struct {
		want error
		edit func(*providerScoreEvidenceFake, *ProviderScoredRouteRequest)
	}{
		"unknown policy":    {ErrProviderScoreEvidenceUnavailable, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) { r.policyErr = errors.New("absent") }},
		"incomplete corpus": {ErrProviderScoreEvidenceUnavailable, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) { r.corpus.Complete = false }},
		"requested profile": {ErrProviderScoreEvidenceDivergent, func(_ *providerScoreEvidenceFake, q *ProviderScoredRouteRequest) { q.ProfileRef = "profile:other:v1" }},
		"valid cross policy receipt": {ErrProviderScoreEvidenceDivergent, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) {
			score := valid
			score.PolicyRef = changedProviderScoreDigest(score.PolicyRef)
			r.setScores(sealProviderScoreFixture(t, score))
		}},
		"valid cross profile receipt": {ErrProviderScoreEvidenceDivergent, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) {
			score := valid
			score.ProfileRef = "profile:other:v1"
			r.setScores(sealProviderScoreFixture(t, score))
		}},
		"tampered receipt": {ErrProviderScoreEvidenceDivergent, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) {
			score := valid
			score.SampleCount++
			r.corpus.Observations = []ProviderModelScoreObservation{score}
		}},
		"duplicate candidate": {ErrProviderScoreEvidenceDivergent, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) {
			score := valid
			score.SampleCount++
			r.setScores(valid, sealProviderScoreFixture(t, score))
		}},
		"non canonical corpus": {ErrProviderScoreEvidenceDivergent, func(r *providerScoreEvidenceFake, _ *ProviderScoredRouteRequest) {
			other := providerScoreFixture(t, now, policyRef, "provider:two", "model:two", 8_100, 8_000, 20)
			r.setScores(valid, other)
			r.corpus.Observations[0], r.corpus.Observations[1] = r.corpus.Observations[1], r.corpus.Observations[0]
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			reader, request := providerScoreReaderFixture(policyRef, policy, valid), providerScoreRequestFixture(policyRef)
			test.edit(reader, &request)
			_, err := mustProviderScoringQuery(t, &providerScoreClock{now: now}, reader).RouteProviderModelByScore(context.Background(), catalog, request)
			if !errors.Is(err, test.want) {
				t.Fatalf("err=%v want=%v", err, test.want)
			}
		})
	}
}

func TestProviderScoringReadsClockAfterCorpus(t *testing.T) {
	start := time.Date(2026, 8, 21, 13, 0, 0, 0, time.UTC)
	policy, policyRef := providerScorePolicyFixture(t)
	score := providerScoreFixture(t, start, policyRef, "provider:one", "model:one", 8_000, 8_000, 20)
	score.ExpiresAt = start.Add(time.Minute)
	score = sealProviderScoreFixture(t, score)
	trace, clock := []string{}, &providerScoreClock{now: start}
	clock.trace = &trace
	reader := providerScoreReaderFixture(policyRef, policy, score)
	reader.trace, reader.afterCorpus = &trace, func() { clock.now = score.ExpiresAt }
	catalog := providerScoreCatalog(start, score.Candidate)
	decision, err := mustProviderScoringQuery(t, clock, reader).RouteProviderModelByScore(context.Background(), catalog, providerScoreRequestFixture(policyRef))
	if err != nil || decision.Route.Selected || len(decision.Ranking.Rejections) != 1 ||
		decision.Ranking.Rejections[0].Reason != ProviderScoreStale || !reflect.DeepEqual(trace, []string{"policy", "corpus", "clock"}) {
		t.Fatalf("decision=%+v trace=%v err=%v", decision, trace, err)
	}
}

func TestProviderScoringAppliesEarlyCountAndRefLimits(t *testing.T) {
	now := time.Date(2026, 8, 21, 14, 0, 0, 0, time.UTC)
	policy, policyRef := providerScorePolicyFixture(t)
	valid := providerScoreFixture(t, now, policyRef, "provider:one", "model:one", 8_000, 8_000, 20)
	catalog := providerScoreCatalog(now, valid.Candidate)
	overCount := make([]string, providerScoreMaximumCapabilities+1)
	for name, edit := range map[string]func(*ProviderScoredRouteRequest){
		"capability overcount": func(q *ProviderScoredRouteRequest) { q.RequiredCapabilityRefs = overCount },
		"duplicate capability": func(q *ProviderScoredRouteRequest) {
			q.RequiredCapabilityRefs = []string{"capability:edit", "capability:edit"}
		},
		"oversized profile": func(q *ProviderScoredRouteRequest) {
			q.ProfileRef = strings.Repeat("p", providerScoreMaximumRefBytes+1)
		},
		"oversized capability": func(q *ProviderScoredRouteRequest) {
			q.RequiredCapabilityRefs = []string{"capability:" + strings.Repeat("x", providerScoreMaximumRefBytes)}
		},
	} {
		t.Run(name, func(t *testing.T) {
			reader, clock, request := providerScoreReaderFixture(policyRef, policy, valid), &providerScoreClock{now: now}, providerScoreRequestFixture(policyRef)
			edit(&request)
			_, err := mustProviderScoringQuery(t, clock, reader).RouteProviderModelByScore(context.Background(), catalog, request)
			if !errors.Is(err, ErrProviderScoreInvalid) || len(reader.calls) != 0 || clock.calls != 0 {
				t.Fatalf("err=%v calls=%v/%d", err, reader.calls, clock.calls)
			}
		})
	}

	for name, edit := range map[string]func(*providerScoreEvidenceFake){
		"policy overcount": func(r *providerScoreEvidenceFake) { r.policy.CapabilityRefs = overCount },
		"corpus overcount": func(r *providerScoreEvidenceFake) {
			r.corpus.Observations = make([]ProviderModelScoreObservation, providerScoreMaximumObservations+1)
		},
		"oversized observation ref": func(r *providerScoreEvidenceFake) {
			r.corpus.Observations[0].ObserverRef = "observer:" + strings.Repeat("x", providerScoreMaximumRefBytes)
		},
	} {
		t.Run(name, func(t *testing.T) {
			reader, clock := providerScoreReaderFixture(policyRef, policy, valid), &providerScoreClock{now: now}
			edit(reader)
			_, err := mustProviderScoringQuery(t, clock, reader).RouteProviderModelByScore(context.Background(), catalog, providerScoreRequestFixture(policyRef))
			if !errors.Is(err, ErrProviderScoreEvidenceDivergent) || clock.calls != 0 {
				t.Fatalf("err=%v clock=%d", err, clock.calls)
			}
		})
	}
}

type providerScoreClock struct {
	now   time.Time
	calls int
	trace *[]string
}

func (clock *providerScoreClock) Now() time.Time {
	clock.calls++
	if clock.trace != nil {
		*clock.trace = append(*clock.trace, "clock")
	}
	return clock.now
}

type providerScoreEvidenceFake struct {
	policyRef   string
	policy      ProviderScorePolicy
	corpus      ProviderScoreEvidenceCorpus
	policyErr   error
	calls       []string
	limit       int
	trace       *[]string
	afterCorpus func()
}

func (fake *providerScoreEvidenceFake) ResolveProviderScorePolicy(_ context.Context, ref string) (ProviderScorePolicy, error) {
	fake.calls = append(fake.calls, "policy")
	if fake.trace != nil {
		*fake.trace = append(*fake.trace, "policy")
	}
	if fake.policyErr != nil || ref != fake.policyRef {
		return ProviderScorePolicy{}, errors.New("policy absent")
	}
	policy := fake.policy
	policy.CapabilityRefs = append([]string(nil), policy.CapabilityRefs...)
	return policy, nil
}

func (fake *providerScoreEvidenceFake) ResolveProviderScoreEvidenceCorpus(_ context.Context, _ ProviderScoreEvidenceSubject, limit int) (ProviderScoreEvidenceCorpus, error) {
	fake.calls = append(fake.calls, "corpus")
	if fake.trace != nil {
		*fake.trace = append(*fake.trace, "corpus")
	}
	fake.limit = limit
	if fake.afterCorpus != nil {
		fake.afterCorpus()
	}
	return fake.corpus, nil
}

func (fake *providerScoreEvidenceFake) setScores(scores ...ProviderModelScoreObservation) {
	sort.Slice(scores, func(i, j int) bool { return scores[i].EvidenceReceiptRef < scores[j].EvidenceReceiptRef })
	fake.corpus = ProviderScoreEvidenceCorpus{Complete: true, Observations: scores}
}

func providerScorePolicyFixture(t *testing.T) (ProviderScorePolicy, string) {
	t.Helper()
	policy := ProviderScorePolicy{AuthorityRef: "authority:provider-scoring:v1", ProfileRef: "profile:build:v1",
		CapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium,
		MinimumConfidenceBasisPoints: 7_000, MinimumSampleCount: 10, MaximumSampleCount: 1_000, MaximumObservationAge: time.Hour}
	ref, err := ProviderScorePolicyRef(policy)
	if err != nil {
		t.Fatal(err)
	}
	return policy, ref
}

func providerScoreFixture(t *testing.T, now time.Time, policyRef, providerRef, modelRef string, score, confidence uint16, samples uint64) ProviderModelScoreObservation {
	t.Helper()
	return sealProviderScoreFixture(t, ProviderModelScoreObservation{PolicyRef: policyRef, ProfileRef: "profile:build:v1",
		CapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium,
		Candidate: ProviderRouteCandidate{providerRef, modelRef}, ObserverRef: "observer:delivery-receipts:v1",
		ScoreBasisPoints: score, ConfidenceBasisPoints: confidence, SampleCount: samples,
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(5 * time.Minute)})
}

func sealProviderScoreFixture(t *testing.T, score ProviderModelScoreObservation) ProviderModelScoreObservation {
	t.Helper()
	ref, err := ProviderScoreEvidenceReceiptRef(score)
	if err != nil {
		t.Fatal(err)
	}
	score.EvidenceReceiptRef = ref
	return score
}

func providerScoreReaderFixture(policyRef string, policy ProviderScorePolicy, scores ...ProviderModelScoreObservation) *providerScoreEvidenceFake {
	fake := &providerScoreEvidenceFake{policyRef: policyRef, policy: policy}
	fake.setScores(scores...)
	return fake
}

func providerScoreCatalog(now time.Time, candidates ...ProviderRouteCandidate) ProviderCatalog {
	providers := make(map[string]ports.ProviderCatalogObservation, len(candidates))
	for _, candidate := range candidates {
		providers[candidate.ProviderRef] = ports.ProviderCatalogObservation{ProviderRef: candidate.ProviderRef,
			Models: []ports.ProviderModel{{ProviderRef: candidate.ProviderRef, ModelRef: candidate.ModelRef,
				CapabilityRefs: []string{"capability:edit"}, ReasoningEfforts: []governance.ReasoningEffort{governance.ReasoningEffortMedium}}},
			Availability: ports.ProviderAvailabilityAvailable, Quota: ports.ProviderQuotaAvailable,
			ObservedAt: now, ExpiresAt: now.Add(time.Hour)}
	}
	return ProviderCatalog{observedAt: now, providers: providers, failures: map[string]ProviderCatalogFailure{}}
}

func providerScoreRequestFixture(policyRef string) ProviderScoredRouteRequest {
	return ProviderScoredRouteRequest{PolicyRef: policyRef, ProfileRef: "profile:build:v1",
		RequiredCapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium}
}

func mustProviderScoringQuery(t *testing.T, clock Clock, reader ProviderScoreEvidenceReader) *ProviderScoringQuery {
	t.Helper()
	query, err := NewProviderScoringQuery(clock, reader)
	if err != nil {
		t.Fatal(err)
	}
	return query
}

func changedProviderScoreDigest(ref string) string {
	last := "0"
	if ref[len(ref)-1] == '0' {
		last = "1"
	}
	return ref[:len(ref)-1] + last
}
