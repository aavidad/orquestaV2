package governance

import "strings"

// BudgetScope is one level of the deployment -> project -> Goal hierarchy.
type BudgetScope string

const (
	BudgetScopeDeployment BudgetScope = "deployment"
	BudgetScopeProject    BudgetScope = "project"
	BudgetScopeGoal       BudgetScope = "goal"
)

type BudgetEnvelope struct {
	Ref        string
	SubjectRef string
	Scope      BudgetScope
	Limit      ResourceVector
	Revision   uint64
}

type BudgetDemand struct {
	Ref       string
	Resources ResourceVector
}

type BudgetReservation struct {
	Ref       string
	DemandRef string
	Resources ResourceVector
}

// ResourceDimensions makes missing telemetry explicit. A known zero differs
// from an unknown dimension, which must retain its reservation at settlement.
type ResourceDimensions uint8

const (
	ResourceTokens ResourceDimensions = 1 << iota
	ResourceMoney
	ResourceActiveTime
	ResourceProcessSlots
	ResourceDisk
	AllResourceDimensions = ResourceTokens | ResourceMoney | ResourceActiveTime | ResourceProcessSlots | ResourceDisk
)

type UsageQuality string

const (
	UsageQualityUnknown   UsageQuality = "unknown"
	UsageQualityEstimated UsageQuality = "estimated"
	UsageQualityMeasured  UsageQuality = "measured"
	UsageQualityExact     UsageQuality = "exact"
)

type ResourceUsage struct {
	Resources ResourceVector
	Known     ResourceDimensions
	Quality   UsageQuality
}

// BudgetSettlement separates observed telemetry from conservative accounting.
type BudgetSettlement struct {
	ReservationRef string
	Reserved       ResourceVector
	Observed       ResourceUsage
	Charged        ResourceVector
	Released       ResourceVector
	Overrun        ResourceVector
}

func ValidateBudgetEnvelope(envelope BudgetEnvelope) error {
	if !validOpaqueRef(envelope.Ref) || !validOpaqueRef(envelope.SubjectRef) {
		return domainError(ErrorInvalidRef, "budget_envelope")
	}
	if envelope.Scope != BudgetScopeDeployment && envelope.Scope != BudgetScopeProject && envelope.Scope != BudgetScopeGoal {
		return domainError(ErrorInvalidArgument, "budget_scope")
	}
	if envelope.Revision == 0 {
		return domainError(ErrorInvalidArgument, "budget_revision")
	}
	return ValidateResourceVector(envelope.Limit)
}

func ValidateBudgetDemand(demand BudgetDemand) error {
	if !validOpaqueRef(demand.Ref) {
		return domainError(ErrorInvalidRef, "budget_demand_ref")
	}
	return ValidateResourceVector(demand.Resources)
}

func ValidateBudgetReservation(reservation BudgetReservation) error {
	if !validOpaqueRef(reservation.Ref) || !validOpaqueRef(reservation.DemandRef) {
		return domainError(ErrorInvalidRef, "budget_reservation")
	}
	return ValidateResourceVector(reservation.Resources)
}

func ValidateResourceUsage(usage ResourceUsage) error {
	if err := ValidateResourceVector(usage.Resources); err != nil {
		return err
	}
	if usage.Known&^AllResourceDimensions != 0 {
		return domainError(ErrorInvalidArgument, "known_dimensions")
	}
	if usage.Quality != UsageQualityUnknown && usage.Quality != UsageQualityEstimated &&
		usage.Quality != UsageQualityMeasured && usage.Quality != UsageQualityExact {
		return domainError(ErrorInvalidArgument, "usage_quality")
	}
	if (usage.Known == 0) != (usage.Quality == UsageQualityUnknown) {
		return domainError(ErrorInvalidArgument, "usage_quality")
	}
	if unknownValuePresent(usage) {
		return domainError(ErrorInvalidArgument, "unknown_dimension_value")
	}
	return nil
}

// Reconcile charges observed values for known dimensions and the full
// reservation for unknown ones. Overruns are recorded; they never wrap or
// manufacture a negative release.
func Reconcile(reservation BudgetReservation, usage ResourceUsage) (BudgetSettlement, error) {
	if err := ValidateBudgetReservation(reservation); err != nil {
		return BudgetSettlement{}, err
	}
	if err := ValidateResourceUsage(usage); err != nil {
		return BudgetSettlement{}, err
	}
	return reconcileResources(reservation.Ref, reservation.Resources, usage)
}

func reconcileResources(reservationRef string, reserved ResourceVector, usage ResourceUsage) (BudgetSettlement, error) {
	charged := reserved
	applyKnownUsage(&charged, usage)
	if _, err := mergeCurrency(reserved.Currency, charged.Currency); err != nil {
		return BudgetSettlement{}, err
	}
	charged.Currency, _ = mergeCurrency(reserved.Currency, charged.Currency)
	released, overrun := resourceDifference(reserved, charged), resourceDifference(charged, reserved)
	return BudgetSettlement{
		ReservationRef: reservationRef, Reserved: reserved, Observed: usage,
		Charged: charged, Released: released, Overrun: overrun,
	}, nil
}

// ValidateBudgetSettlement lets persistence reject a tampered accounting fact.
func ValidateBudgetSettlement(settlement BudgetSettlement) error {
	if !validOpaqueRef(settlement.ReservationRef) {
		return domainError(ErrorInvalidRef, "budget_reservation_ref")
	}
	if err := ValidateResourceVector(settlement.Reserved); err != nil {
		return err
	}
	if err := ValidateResourceUsage(settlement.Observed); err != nil {
		return err
	}
	expected, err := reconcileResources(settlement.ReservationRef, settlement.Reserved, settlement.Observed)
	if err != nil {
		return err
	}
	if settlement.Charged != expected.Charged || settlement.Released != expected.Released ||
		settlement.Overrun != expected.Overrun {
		return domainError(ErrorInvalidArgument, "budget_settlement")
	}
	return nil
}

func applyKnownUsage(charged *ResourceVector, usage ResourceUsage) {
	if usage.Known&ResourceTokens != 0 {
		charged.Tokens = usage.Resources.Tokens
	}
	if usage.Known&ResourceMoney != 0 {
		charged.MoneyMicros = usage.Resources.MoneyMicros
		if usage.Resources.Currency != "" {
			charged.Currency = usage.Resources.Currency
		}
	}
	if usage.Known&ResourceActiveTime != 0 {
		charged.ActiveTimeNS = usage.Resources.ActiveTimeNS
	}
	if usage.Known&ResourceProcessSlots != 0 {
		charged.ProcessSlots = usage.Resources.ProcessSlots
	}
	if usage.Known&ResourceDisk != 0 {
		charged.DiskBytes = usage.Resources.DiskBytes
	}
}

func unknownValuePresent(usage ResourceUsage) bool {
	return usage.Known&ResourceTokens == 0 && usage.Resources.Tokens != 0 ||
		usage.Known&ResourceMoney == 0 && (usage.Resources.MoneyMicros != 0 || usage.Resources.Currency != "") ||
		usage.Known&ResourceActiveTime == 0 && usage.Resources.ActiveTimeNS != 0 ||
		usage.Known&ResourceProcessSlots == 0 && usage.Resources.ProcessSlots != 0 ||
		usage.Known&ResourceDisk == 0 && usage.Resources.DiskBytes != 0
}

func resourceDifference(left, right ResourceVector) ResourceVector {
	return ResourceVector{
		Tokens: maxZero(left.Tokens - right.Tokens), MoneyMicros: maxZero(left.MoneyMicros - right.MoneyMicros),
		Currency: left.Currency, ActiveTimeNS: maxZero(left.ActiveTimeNS - right.ActiveTimeNS),
		ProcessSlots: maxZero(left.ProcessSlots - right.ProcessSlots), DiskBytes: maxZero(left.DiskBytes - right.DiskBytes),
	}
}

func maxZero(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func validOpaqueRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}
