package gaps

import (
	"errors"
	"sync"

	"orquesta/internal/intake"
)

const evaluatorV1DigestGolden = "8ab38194e0445940f84ae8d838247cfb40e1e78c3496ad1da3ecff539340a030"
const evaluatorV1SourceDigestGolden = "ad5edf4b38362100a14bbe67556c6a9204e0be1f968fd42cc67105b9ee430e82"

type evaluateFunc func(Input) (Result, error)

// Evaluator couples one frozen implementation with its exact semantic
// identity and inventory. A registry retains historical evaluators for replay.
type Evaluator struct {
	identity   intake.DerivationIdentity
	inventory  Inventory
	evaluateFn evaluateFunc
}

func (value Evaluator) Identity() intake.DerivationIdentity {
	return value.identity
}

func (value Evaluator) Dimensions() []DimensionDescriptor {
	return value.inventory.Dimensions()
}

func (value Evaluator) Rules() []RuleDescriptor {
	return value.inventory.Rules()
}

func (value Evaluator) Evaluate(input Input) (Result, error) {
	if value.evaluateFn == nil {
		return Result{}, domainError(ErrorInvalidArgument, "evaluator")
	}
	return value.evaluateFn(input)
}

type EvaluatorRegistry struct {
	byIdentity map[string]Evaluator
}

func (value EvaluatorRegistry) Empty() bool {
	return len(value.byIdentity) == 0
}

func (value EvaluatorRegistry) Resolve(
	identity intake.DerivationIdentity,
) (Evaluator, error) {
	if value.byIdentity == nil {
		return Evaluator{}, domainError(ErrorUnsupportedEvaluator, "evaluator_identity")
	}
	evaluator, found := value.byIdentity[evaluatorIdentityKey(identity)]
	if !found {
		return Evaluator{}, domainError(ErrorUnsupportedEvaluator, "evaluator_identity")
	}
	return evaluator, nil
}

func BuiltInEvaluatorRegistry() EvaluatorRegistry {
	identity := EvaluatorV1Identity()
	evaluator := Evaluator{
		identity: identity, inventory: BuiltInV1(), evaluateFn: EvaluateV1,
	}
	return EvaluatorRegistry{
		byIdentity: map[string]Evaluator{
			evaluatorIdentityKey(identity): evaluator,
		},
	}
}

var evaluatorV1IdentityCache struct {
	once  sync.Once
	value intake.DerivationIdentity
}

func EvaluatorV1Identity() intake.DerivationIdentity {
	evaluatorV1IdentityCache.once.Do(func() {
		computed := evaluatorV1SemanticDigest()
		if computed != evaluatorV1DigestGolden {
			panic("wizard gaps evaluator V1 semantic digest changed")
		}
		value, err := intake.NewDerivationIdentity(
			evaluatorSchema,
			evaluatorV1Version,
			evaluatorV1DigestGolden,
		)
		if err != nil {
			panic(err)
		}
		evaluatorV1IdentityCache.value = value
	})
	return evaluatorV1IdentityCache.value
}

func evaluatorIdentityKey(value intake.DerivationIdentity) string {
	return value.Schema + "\x00" + value.Version + "\x00" + value.SemanticDigest
}

func IsUnsupportedEvaluator(err error) bool {
	var domainErr *DomainError
	return errors.As(err, &domainErr) &&
		domainErr.Code == ErrorUnsupportedEvaluator
}
