package application

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

const (
	ProviderScoreBasisPointsMaximum  = uint16(10_000)
	providerScoreMaximumCapabilities = 64
	providerScoreMaximumObservations = 256
	providerScoreMaximumRefBytes     = 256
	providerScorePolicyRefPrefix     = "provider-score-policy:sha256:"
	providerScoreEvidenceRefPrefix   = "provider-score-evidence:sha256:"
)

var (
	ErrProviderScoreInvalid             = errors.New("application.provider_score_invalid")
	ErrProviderScoreEvidenceUnavailable = errors.New("application.provider_score_evidence_unavailable")
	ErrProviderScoreEvidenceDivergent   = errors.New("application.provider_score_evidence_divergent")
)

// ProviderScorePolicy is resolved authority, never request-proposed policy.
type ProviderScorePolicy struct {
	AuthorityRef                 string
	ProfileRef                   string
	CapabilityRefs               []string
	ReasoningEffort              governance.ReasoningEffort
	MinimumConfidenceBasisPoints uint16
	MinimumSampleCount           uint64
	MaximumSampleCount           uint64
	MaximumObservationAge        time.Duration
}

// ProviderModelScoreObservation binds separate score, confidence and samples
// to one exact policy and subject.
type ProviderModelScoreObservation struct {
	PolicyRef             string
	ProfileRef            string
	CapabilityRefs        []string
	ReasoningEffort       governance.ReasoningEffort
	Candidate             ProviderRouteCandidate
	ObserverRef           string
	ScoreBasisPoints      uint16
	ConfidenceBasisPoints uint16
	SampleCount           uint64
	EvidenceReceiptRef    string
	ObservedAt            time.Time
	ExpiresAt             time.Time
}

// ProviderScoreEvidenceSubject is the exact caller-requested lookup subject.
type ProviderScoreEvidenceSubject struct {
	PolicyRef       string
	ProfileRef      string
	CapabilityRefs  []string
	ReasoningEffort governance.ReasoningEffort
}

// ProviderScoreEvidenceCorpus is complete only when no admitted observation
// for its subject was omitted. Observations must be ordered by receipt ref.
type ProviderScoreEvidenceCorpus struct {
	Complete     bool
	Observations []ProviderModelScoreObservation
}

// ProviderScoreEvidenceReader resolves authoritative policy and the complete,
// canonical corpus. Hashes bind content; reader composition authenticates it.
type ProviderScoreEvidenceReader interface {
	ResolveProviderScorePolicy(context.Context, string) (ProviderScorePolicy, error)
	ResolveProviderScoreEvidenceCorpus(context.Context, ProviderScoreEvidenceSubject, int) (ProviderScoreEvidenceCorpus, error)
}

type ProviderScoreRejectionReason string

const (
	ProviderScoreStale                  ProviderScoreRejectionReason = "score_stale"
	ProviderScoreConfidenceInsufficient ProviderScoreRejectionReason = "score_confidence_insufficient"
	ProviderScoreSamplesInsufficient    ProviderScoreRejectionReason = "score_samples_insufficient"
)

type ProviderScoreRejection struct {
	Candidate ProviderRouteCandidate
	Reason    ProviderScoreRejectionReason
}

type ProviderScoreRanking struct {
	RankedCandidates []ProviderRouteCandidate
	QualifiedScores  []ProviderModelScoreObservation
	Rejections       []ProviderScoreRejection
}

// ProviderScoredRouteRequest contains only the policy ref and exact subject.
// The caller cannot submit metrics or select evidence receipts.
type ProviderScoredRouteRequest struct {
	PolicyRef              string
	ProfileRef             string
	RequiredCapabilityRefs []string
	ReasoningEffort        governance.ReasoningEffort
	AllowFallback          bool
}

type ProviderScoredRouteDecision struct {
	Policy              ProviderScorePolicy
	EvidenceReceiptRefs []string
	Ranking             ProviderScoreRanking
	Route               ProviderRouteDecision
}

// ProviderScoringQuery is query-only; durable authenticated composition is a
// later gate.
type ProviderScoringQuery struct {
	clock    Clock
	evidence ProviderScoreEvidenceReader
}

func NewProviderScoringQuery(clock Clock, evidence ProviderScoreEvidenceReader) (*ProviderScoringQuery, error) {
	if clock == nil || evidence == nil {
		return nil, ErrProviderScoreInvalid
	}
	return &ProviderScoringQuery{clock: clock, evidence: evidence}, nil
}

func ProviderScorePolicyRef(policy ProviderScorePolicy) (string, error) {
	normalized, err := normalizeProviderScorePolicy(policy)
	if err != nil {
		return "", err
	}
	fields := []string{
		normalized.AuthorityRef, normalized.ProfileRef, string(normalized.ReasoningEffort),
		strconv.FormatUint(uint64(normalized.MinimumConfidenceBasisPoints), 10),
		strconv.FormatUint(normalized.MinimumSampleCount, 10), strconv.FormatUint(normalized.MaximumSampleCount, 10),
		strconv.FormatInt(int64(normalized.MaximumObservationAge), 10), strconv.Itoa(len(normalized.CapabilityRefs)),
	}
	fields = append(fields, normalized.CapabilityRefs...)
	return providerScorePolicyRefPrefix + fingerprintFields("orquesta.provider-score-policy.v1", fields...), nil
}

func ProviderScoreEvidenceReceiptRef(score ProviderModelScoreObservation) (string, error) {
	normalized, err := normalizeProviderModelScoreObservation(score)
	if err != nil {
		return "", err
	}
	fields := []string{
		normalized.PolicyRef, normalized.ProfileRef, string(normalized.ReasoningEffort),
		normalized.Candidate.ProviderRef, normalized.Candidate.ModelRef, normalized.ObserverRef,
		strconv.FormatUint(uint64(normalized.ScoreBasisPoints), 10), strconv.FormatUint(uint64(normalized.ConfidenceBasisPoints), 10),
		strconv.FormatUint(normalized.SampleCount, 10), normalized.ObservedAt.Format(time.RFC3339Nano),
		normalized.ExpiresAt.Format(time.RFC3339Nano), strconv.Itoa(len(normalized.CapabilityRefs)),
	}
	fields = append(fields, normalized.CapabilityRefs...)
	return providerScoreEvidenceRefPrefix + fingerprintFields("orquesta.provider-score-evidence.v1", fields...), nil
}

// RouteProviderModelByScore resolves policy and the complete corpus before its
// single clock read, then delegates provider facts to the neutral router.
func (query *ProviderScoringQuery) RouteProviderModelByScore(
	ctx context.Context,
	catalog ProviderCatalog,
	request ProviderScoredRouteRequest,
) (ProviderScoredRouteDecision, error) {
	decision := ProviderScoredRouteDecision{Route: ProviderRouteDecision{Reason: ProviderRouteNoRoute}}
	if query == nil || query.clock == nil || query.evidence == nil || ctx == nil {
		return decision, ErrProviderScoreInvalid
	}
	if err := ctx.Err(); err != nil {
		return decision, err
	}
	normalizedRequest, err := normalizeProviderScoredRouteRequest(request)
	if err != nil {
		return decision, err
	}
	policy, err := query.evidence.ResolveProviderScorePolicy(ctx, normalizedRequest.PolicyRef)
	if err != nil {
		return decision, providerScoreResolutionError(ctx)
	}
	policy, err = normalizeProviderScorePolicy(policy)
	if err != nil {
		return decision, ErrProviderScoreEvidenceDivergent
	}
	wantPolicyRef, _ := ProviderScorePolicyRef(policy)
	if wantPolicyRef != normalizedRequest.PolicyRef || policy.ProfileRef != normalizedRequest.ProfileRef ||
		!slices.Equal(policy.CapabilityRefs, normalizedRequest.RequiredCapabilityRefs) ||
		policy.ReasoningEffort != normalizedRequest.ReasoningEffort {
		return decision, ErrProviderScoreEvidenceDivergent
	}
	decision.Policy = policy
	subject := ProviderScoreEvidenceSubject{
		PolicyRef: normalizedRequest.PolicyRef, ProfileRef: normalizedRequest.ProfileRef,
		CapabilityRefs:  append([]string(nil), normalizedRequest.RequiredCapabilityRefs...),
		ReasoningEffort: normalizedRequest.ReasoningEffort,
	}
	corpus, err := query.evidence.ResolveProviderScoreEvidenceCorpus(ctx, subject, providerScoreMaximumObservations)
	if err != nil {
		return decision, providerScoreResolutionError(ctx)
	}
	scores, refs, err := normalizeProviderScoreEvidenceCorpus(corpus, normalizedRequest.PolicyRef, policy)
	if err != nil {
		return decision, err
	}
	decision.EvidenceReceiptRefs = refs
	now := query.clock.Now().Round(0).UTC()
	if now.IsZero() || catalog.ObservedAt().IsZero() || catalog.ObservedAt().After(now) {
		return decision, ErrProviderScoreInvalid
	}
	decision.Ranking, err = rankProviderModelScores(now, normalizedRequest.PolicyRef, policy, scores)
	if err != nil || len(decision.Ranking.RankedCandidates) == 0 {
		return decision, err
	}
	routingCatalog := catalog
	routingCatalog.observedAt = now
	decision.Route, err = RouteProviderModel(routingCatalog, ProviderRouteRequest{
		Candidates: decision.Ranking.RankedCandidates, RequiredCapabilityRefs: normalizedRequest.RequiredCapabilityRefs,
		ReasoningEffort: normalizedRequest.ReasoningEffort, AllowFallback: normalizedRequest.AllowFallback,
	})
	if err != nil {
		return ProviderScoredRouteDecision{}, err
	}
	return decision, nil
}

func normalizeProviderScoreEvidenceCorpus(
	corpus ProviderScoreEvidenceCorpus,
	policyRef string,
	policy ProviderScorePolicy,
) ([]ProviderModelScoreObservation, []string, error) {
	if !corpus.Complete || len(corpus.Observations) == 0 {
		return nil, nil, ErrProviderScoreEvidenceUnavailable
	}
	if len(corpus.Observations) > providerScoreMaximumObservations {
		return nil, nil, ErrProviderScoreEvidenceDivergent
	}
	scores := make([]ProviderModelScoreObservation, len(corpus.Observations))
	refs := make([]string, len(corpus.Observations))
	seenCandidates := make(map[ProviderRouteCandidate]struct{}, len(corpus.Observations))
	for index, raw := range corpus.Observations {
		if !validProviderScoreDigestRef(raw.EvidenceReceiptRef, providerScoreEvidenceRefPrefix) {
			return nil, nil, ErrProviderScoreEvidenceDivergent
		}
		score, err := normalizeProviderModelScoreObservation(raw)
		if err != nil {
			return nil, nil, ErrProviderScoreEvidenceDivergent
		}
		wantRef, _ := ProviderScoreEvidenceReceiptRef(score)
		if wantRef != score.EvidenceReceiptRef || score.PolicyRef != policyRef || score.ProfileRef != policy.ProfileRef ||
			!slices.Equal(score.CapabilityRefs, policy.CapabilityRefs) || score.ReasoningEffort != policy.ReasoningEffort ||
			(index > 0 && refs[index-1] >= score.EvidenceReceiptRef) {
			return nil, nil, ErrProviderScoreEvidenceDivergent
		}
		if _, duplicate := seenCandidates[score.Candidate]; duplicate {
			return nil, nil, ErrProviderScoreEvidenceDivergent
		}
		seenCandidates[score.Candidate] = struct{}{}
		scores[index], refs[index] = score, score.EvidenceReceiptRef
	}
	return scores, refs, nil
}

func rankProviderModelScores(now time.Time, policyRef string, policy ProviderScorePolicy, scores []ProviderModelScoreObservation) (ProviderScoreRanking, error) {
	if len(scores) > providerScoreMaximumObservations {
		return ProviderScoreRanking{}, ErrProviderScoreEvidenceDivergent
	}
	accepted := make([]ProviderModelScoreObservation, 0, len(scores))
	rejections := make([]ProviderScoreRejection, 0, len(scores))
	seenCandidates := make(map[ProviderRouteCandidate]struct{}, len(scores))
	seenReceipts := make(map[string]struct{}, len(scores))
	for _, score := range scores {
		if score.PolicyRef != policyRef || score.ProfileRef != policy.ProfileRef ||
			!slices.Equal(score.CapabilityRefs, policy.CapabilityRefs) || score.ReasoningEffort != policy.ReasoningEffort ||
			score.SampleCount > policy.MaximumSampleCount || score.ExpiresAt.Sub(score.ObservedAt) > policy.MaximumObservationAge ||
			score.ObservedAt.After(now) {
			return ProviderScoreRanking{}, ErrProviderScoreEvidenceDivergent
		}
		if _, duplicate := seenCandidates[score.Candidate]; duplicate {
			return ProviderScoreRanking{}, ErrProviderScoreEvidenceDivergent
		}
		if _, duplicate := seenReceipts[score.EvidenceReceiptRef]; duplicate {
			return ProviderScoreRanking{}, ErrProviderScoreEvidenceDivergent
		}
		seenCandidates[score.Candidate], seenReceipts[score.EvidenceReceiptRef] = struct{}{}, struct{}{}
		reason := ProviderScoreRejectionReason("")
		switch {
		case !now.Before(score.ExpiresAt), score.ObservedAt.Before(now.Add(-policy.MaximumObservationAge)):
			reason = ProviderScoreStale
		case score.ConfidenceBasisPoints < policy.MinimumConfidenceBasisPoints:
			reason = ProviderScoreConfidenceInsufficient
		case score.SampleCount < policy.MinimumSampleCount:
			reason = ProviderScoreSamplesInsufficient
		}
		if reason != "" {
			rejections = append(rejections, ProviderScoreRejection{Candidate: score.Candidate, Reason: reason})
		} else {
			accepted = append(accepted, score)
		}
	}
	sort.Slice(accepted, func(i, j int) bool { return providerScoreRanksBefore(accepted[i], accepted[j]) })
	sort.Slice(rejections, func(i, j int) bool {
		left, right := rejections[i], rejections[j]
		if left.Candidate.ProviderRef != right.Candidate.ProviderRef {
			return left.Candidate.ProviderRef < right.Candidate.ProviderRef
		}
		if left.Candidate.ModelRef != right.Candidate.ModelRef {
			return left.Candidate.ModelRef < right.Candidate.ModelRef
		}
		return left.Reason < right.Reason
	})
	ranked := make([]ProviderRouteCandidate, len(accepted))
	for index := range accepted {
		ranked[index] = accepted[index].Candidate
	}
	return ProviderScoreRanking{RankedCandidates: ranked, QualifiedScores: accepted, Rejections: rejections}, nil
}

func normalizeProviderScoredRouteRequest(request ProviderScoredRouteRequest) (ProviderScoredRouteRequest, error) {
	if !validProviderScoreDigestRef(request.PolicyRef, providerScorePolicyRefPrefix) ||
		!validProviderScoreRef(request.ProfileRef) || governance.ValidateReasoningEffort(request.ReasoningEffort) != nil {
		return ProviderScoredRouteRequest{}, ErrProviderScoreInvalid
	}
	capabilities, err := normalizeProviderScoreCapabilities(request.RequiredCapabilityRefs)
	if err != nil {
		return ProviderScoredRouteRequest{}, err
	}
	request.RequiredCapabilityRefs = capabilities
	return request, nil
}

func normalizeProviderScorePolicy(policy ProviderScorePolicy) (ProviderScorePolicy, error) {
	if !validProviderScoreRef(policy.AuthorityRef) || !validProviderScoreRef(policy.ProfileRef) {
		return ProviderScorePolicy{}, ErrProviderScoreInvalid
	}
	capabilities, err := normalizeProviderScoreCapabilities(policy.CapabilityRefs)
	if err != nil || governance.ValidateReasoningEffort(policy.ReasoningEffort) != nil ||
		policy.MinimumConfidenceBasisPoints == 0 || policy.MinimumConfidenceBasisPoints > ProviderScoreBasisPointsMaximum ||
		policy.MinimumSampleCount == 0 || policy.MaximumSampleCount < policy.MinimumSampleCount || policy.MaximumObservationAge <= 0 {
		return ProviderScorePolicy{}, ErrProviderScoreInvalid
	}
	policy.CapabilityRefs = capabilities
	return policy, nil
}

func normalizeProviderModelScoreObservation(score ProviderModelScoreObservation) (ProviderModelScoreObservation, error) {
	if !validProviderScoreDigestRef(score.PolicyRef, providerScorePolicyRefPrefix) || !validProviderScoreRef(score.ProfileRef) ||
		!validProviderScoreRef(score.Candidate.ProviderRef) || !validProviderScoreRef(score.Candidate.ModelRef) ||
		!validProviderScoreRef(score.ObserverRef) || score.ObserverRef == score.Candidate.ProviderRef {
		return ProviderModelScoreObservation{}, ErrProviderScoreInvalid
	}
	capabilities, err := normalizeProviderScoreCapabilities(score.CapabilityRefs)
	if err != nil || governance.ValidateReasoningEffort(score.ReasoningEffort) != nil ||
		score.ScoreBasisPoints > ProviderScoreBasisPointsMaximum || score.ConfidenceBasisPoints > ProviderScoreBasisPointsMaximum ||
		score.ObservedAt.IsZero() || !score.ExpiresAt.After(score.ObservedAt) {
		return ProviderModelScoreObservation{}, ErrProviderScoreInvalid
	}
	score.CapabilityRefs = capabilities
	score.ObservedAt, score.ExpiresAt = score.ObservedAt.Round(0).UTC(), score.ExpiresAt.Round(0).UTC()
	return score, nil
}

func normalizeProviderScoreCapabilities(values []string) ([]string, error) {
	if len(values) == 0 || len(values) > providerScoreMaximumCapabilities {
		return nil, ErrProviderScoreInvalid
	}
	for _, value := range values {
		if !validProviderScoreRef(value) {
			return nil, ErrProviderScoreInvalid
		}
		capability, err := goal.NewCapabilityRef(value)
		if err != nil || capability.String() != value {
			return nil, ErrProviderScoreInvalid
		}
	}
	values = append([]string(nil), values...)
	sort.Strings(values)
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return nil, ErrProviderScoreInvalid
		}
	}
	return values, nil
}

func providerScoreResolutionError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrProviderScoreEvidenceUnavailable
}

func validProviderScoreRef(value string) bool {
	return len(value) > 0 && len(value) <= providerScoreMaximumRefBytes && validProviderRouteRef(value)
}

func validProviderScoreDigestRef(value, prefix string) bool {
	if len(value) != len(prefix)+64 || value[:len(prefix)] != prefix {
		return false
	}
	for _, digit := range value[len(prefix):] {
		if (digit < '0' || digit > '9') && (digit < 'a' || digit > 'f') {
			return false
		}
	}
	return true
}

func providerScoreRanksBefore(left, right ProviderModelScoreObservation) bool {
	if left.ScoreBasisPoints != right.ScoreBasisPoints {
		return left.ScoreBasisPoints > right.ScoreBasisPoints
	}
	if left.ConfidenceBasisPoints != right.ConfidenceBasisPoints {
		return left.ConfidenceBasisPoints > right.ConfidenceBasisPoints
	}
	if left.SampleCount != right.SampleCount {
		return left.SampleCount > right.SampleCount
	}
	if left.Candidate.ProviderRef != right.Candidate.ProviderRef {
		return left.Candidate.ProviderRef < right.Candidate.ProviderRef
	}
	return left.Candidate.ModelRef < right.Candidate.ModelRef
}
