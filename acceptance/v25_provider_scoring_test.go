package acceptance_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

const v25ProviderScoringFixturePath = "acceptance/fixtures/v25_provider_scoring.json"

type v25ProviderScoringFixture struct {
	SchemaVersion      int                   `json:"schema_version"`
	ContractID         string                `json:"contract_id"`
	Status             string                `json:"status"`
	Capability         v25RoadmapCapability  `json:"capability"`
	AcceptanceContract v25AcceptanceContract `json:"acceptance_contract"`
	AuthorityRules     []string              `json:"authority_rules"`
	NegativeCases      []string              `json:"negative_cases"`
	SubjectFiles       []string              `json:"subject_files"`
	Deferred           []string              `json:"deferred"`
}

func TestAcceptanceV25ORC26ProviderScoringUsesCompleteAuthoritativeCorpus(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v25ProviderScoringFixture](t, filepath.Join(root, filepath.FromSlash(v25ProviderScoringFixturePath)))
	if fixture.SchemaVersion != 1 || fixture.ContractID != "ORC-26-PROVIDER-MODEL-SCORING" || fixture.Status != "implemented_query_not_wired_not_accredited" ||
		fixture.Capability.ID != "ORC-26" || fixture.Capability.Status != "declared" || len(fixture.Capability.EvidenceRefs) != 0 ||
		fixture.AcceptanceContract != (v25AcceptanceContract{ID: "AC-V25-PROVIDER-ADAPTERS", Status: "planned"}) {
		t.Fatalf("invalid ORC-26 envelope: %+v", fixture)
	}
	assertV25ProviderStrings(t, fixture.AuthorityRules, []string{
		"application_ranks", "reader_owns_complete_canonical_bounded_corpus", "caller_cannot_select_receipts_or_submit_metrics",
		"explicit_profile_capabilities_and_effort_are_exact", "content_addressed_receipts_bind_observation",
		"policy_then_corpus_then_clock_controls_freshness", "fallback_is_explicit", "scoring_is_query_only"})
	assertV25ProviderStrings(t, fixture.NegativeCases, []string{
		"missing_or_incomplete_corpus", "divergent_policy_or_receipt", "valid_cross_policy_or_profile_receipt",
		"stale_or_future_observation", "confidence_or_samples_outside_policy", "duplicate_or_noncanonical_observation",
		"capability_or_observation_overcount", "oversized_ref", "self_observed_score", "provider_quota_unknown"})
	assertV25ProviderStrings(t, fixture.Deferred, []string{
		"durable authenticated policy and evidence reader adapter", "score persistence and recalculation after accepted delivery receipts",
		"public command bindings", "concrete non-Codex provider adapters", "real provider end-to-end evidence", "V25 acceptance accreditation"})
	wantFiles := []string{"internal/application/provider_scoring.go", "internal/application/provider_scoring_test.go",
		"acceptance/v25_provider_scoring_test.go", "acceptance/fixtures/v25_provider_scoring.json",
		"docs/reconstruccion/corte_v25_orc26_scoring_2026-08-21.md"}
	assertV25ProviderStrings(t, fixture.SubjectFiles, wantFiles)
	assertV25ORC26RoadmapDeclared(t, root, fixture)
	assertV25ORC26Runtime(t)
}

func assertV25ORC26RoadmapDeclared(t *testing.T, root string, fixture v25ProviderScoringFixture) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "product/roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var roadmap struct {
		AcceptanceContracts []v25AcceptanceContract `json:"acceptance_contracts"`
		CapabilityEntries   []v25RoadmapCapability  `json:"capability_entries"`
	}
	if err = json.Unmarshal(raw, &roadmap); err != nil {
		t.Fatal(err)
	}
	foundCapability, foundContract := false, false
	for _, entry := range roadmap.CapabilityEntries {
		if entry.ID == fixture.Capability.ID {
			foundCapability = entry.Status == "declared" && len(entry.EvidenceRefs) == 0
		}
	}
	for _, entry := range roadmap.AcceptanceContracts {
		if entry.ID == fixture.AcceptanceContract.ID {
			foundContract = entry.Status == "planned"
		}
	}
	if !foundCapability || !foundContract {
		t.Fatalf("roadmap authority changed: capability=%t contract=%t", foundCapability, foundContract)
	}
}

func assertV25ORC26Runtime(t *testing.T) {
	t.Helper()
	now := time.Date(2026, 8, 21, 15, 0, 0, 0, time.UTC)
	policy := application.ProviderScorePolicy{AuthorityRef: "authority:provider-scoring:v1", ProfileRef: "profile:acceptance:v1",
		CapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium,
		MinimumConfidenceBasisPoints: 7_000, MinimumSampleCount: 10, MaximumSampleCount: 1_000, MaximumObservationAge: time.Hour}
	policyRef, err := application.ProviderScorePolicyRef(policy)
	if err != nil {
		t.Fatal(err)
	}
	more := v25ScoreObservation(t, now, policyRef, policy.ProfileRef, "provider:more", "model:more", 30)
	fewer := v25ScoreObservation(t, now, policyRef, policy.ProfileRef, "provider:fewer", "model:fewer", 20)
	reader := &v25ScoreEvidenceReader{policyRef: policyRef, policy: policy}
	reader.setScores(more, fewer)
	query, err := application.NewProviderScoringQuery(v25ScoreClock{now}, reader)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := application.ObserveProviderCatalog(context.Background(), now, []application.ProviderCatalogSource{
		v25ScoreCatalogSource{v25ProviderObservation(now, more.Candidate)}, v25ScoreCatalogSource{v25ProviderObservation(now, fewer.Candidate)}})
	if err != nil {
		t.Fatal(err)
	}
	request := application.ProviderScoredRouteRequest{PolicyRef: policyRef, ProfileRef: policy.ProfileRef,
		RequiredCapabilityRefs: []string{"capability:edit"}, ReasoningEffort: governance.ReasoningEffortMedium}
	decision, err := query.RouteProviderModelByScore(context.Background(), catalog, request)
	if err != nil || !decision.Route.Selected || decision.Route.Candidate != more.Candidate || len(decision.EvidenceReceiptRefs) != 2 || reader.subject.PolicyRef != policyRef || reader.subject.ProfileRef != policy.ProfileRef {
		t.Fatalf("complete corpus decision=%+v subject=%+v err=%v", decision, reader.subject, err)
	}
	reader.corpus.Complete = false
	if _, err = query.RouteProviderModelByScore(context.Background(), catalog, request); !errors.Is(err, application.ErrProviderScoreEvidenceUnavailable) {
		t.Fatalf("incomplete corpus err=%v", err)
	}
	otherPolicy := policy
	otherPolicy.MinimumSampleCount++
	otherPolicyRef, err := application.ProviderScorePolicyRef(otherPolicy)
	if err != nil {
		t.Fatal(err)
	}
	for name, cross := range map[string]application.ProviderModelScoreObservation{
		"policy":  v25ScoreObservation(t, now, otherPolicyRef, policy.ProfileRef, "provider:more", "model:more", 30),
		"profile": v25ScoreObservation(t, now, policyRef, "profile:other:v1", "provider:more", "model:more", 30),
	} {
		t.Run("valid cross "+name+" receipt", func(t *testing.T) {
			reader.setScores(cross)
			if _, routeErr := query.RouteProviderModelByScore(context.Background(), catalog, request); !errors.Is(routeErr, application.ErrProviderScoreEvidenceDivergent) {
				t.Fatalf("err=%v", routeErr)
			}
		})
	}
}

type v25ScoreClock struct{ now time.Time }

func (clock v25ScoreClock) Now() time.Time { return clock.now }

type v25ScoreEvidenceReader struct {
	policyRef string
	policy    application.ProviderScorePolicy
	corpus    application.ProviderScoreEvidenceCorpus
	subject   application.ProviderScoreEvidenceSubject
}

func (reader *v25ScoreEvidenceReader) ResolveProviderScorePolicy(_ context.Context, ref string) (application.ProviderScorePolicy, error) {
	if ref != reader.policyRef {
		return application.ProviderScorePolicy{}, errors.New("policy absent")
	}
	return reader.policy, nil
}
func (reader *v25ScoreEvidenceReader) ResolveProviderScoreEvidenceCorpus(_ context.Context, subject application.ProviderScoreEvidenceSubject, limit int) (application.ProviderScoreEvidenceCorpus, error) {
	reader.subject = subject
	if len(reader.corpus.Observations) > limit {
		return application.ProviderScoreEvidenceCorpus{}, errors.New("limit")
	}
	return reader.corpus, nil
}
func (reader *v25ScoreEvidenceReader) setScores(scores ...application.ProviderModelScoreObservation) {
	sort.Slice(scores, func(i, j int) bool { return scores[i].EvidenceReceiptRef < scores[j].EvidenceReceiptRef })
	reader.corpus = application.ProviderScoreEvidenceCorpus{Complete: true, Observations: scores}
}

type v25ScoreCatalogSource struct {
	observation ports.ProviderCatalogObservation
}

func (source v25ScoreCatalogSource) ProviderRef() string { return source.observation.ProviderRef }
func (source v25ScoreCatalogSource) ObserveProviderCatalog(context.Context) (ports.ProviderCatalogObservation, error) {
	return source.observation, nil
}
func v25ProviderObservation(now time.Time, candidate application.ProviderRouteCandidate) ports.ProviderCatalogObservation {
	return ports.ProviderCatalogObservation{ProviderRef: candidate.ProviderRef,
		Models:       []ports.ProviderModel{{ProviderRef: candidate.ProviderRef, ModelRef: candidate.ModelRef, CapabilityRefs: []string{"capability:edit"}, ReasoningEfforts: []governance.ReasoningEffort{governance.ReasoningEffortMedium}}},
		Availability: ports.ProviderAvailabilityAvailable, Quota: ports.ProviderQuotaAvailable, Usage: governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt: now, ExpiresAt: now.Add(time.Hour)}
}

func v25ScoreObservation(t *testing.T, now time.Time, policyRef, profileRef, providerRef, modelRef string, samples uint64) application.ProviderModelScoreObservation {
	t.Helper()
	score := application.ProviderModelScoreObservation{PolicyRef: policyRef, ProfileRef: profileRef, CapabilityRefs: []string{"capability:edit"},
		ReasoningEffort: governance.ReasoningEffortMedium, Candidate: application.ProviderRouteCandidate{ProviderRef: providerRef, ModelRef: modelRef},
		ObserverRef: "observer:accepted-delivery:v1", ScoreBasisPoints: 8_000, ConfidenceBasisPoints: 8_000, SampleCount: samples,
		ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(10 * time.Minute)}
	ref, err := application.ProviderScoreEvidenceReceiptRef(score)
	if err != nil {
		t.Fatal(err)
	}
	score.EvidenceReceiptRef = ref
	return score
}
