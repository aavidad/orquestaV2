package goal

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"time"
)

// AppSpecGeneration is the causal generation of one confirmed interpretation
// of an IntentManifest. Generation one is the only root generation.
type AppSpecGeneration uint64

const initialAppSpecGeneration AppSpecGeneration = 1

// AppSpecInput contains caller-supplied facts. Generation and parent binding
// are always derived by the domain constructors.
type AppSpecInput struct {
	Ref         AppSpecRef
	Intent      IntentManifest
	Objective   string
	Reason      string
	ConfirmedBy ActorRef
	ConfirmedAt time.Time
}

// AppSpec is an immutable, confirmed interpretation of exactly one immutable
// IntentManifest. Its hash binds the complete causal chain metadata.
type AppSpec struct {
	ref         AppSpecRef
	generation  AppSpecGeneration
	intent      IntentManifest
	parentRef   AppSpecRef
	parentHash  string
	objective   string
	reason      string
	confirmedBy ActorRef
	confirmedAt time.Time
	hash        string
}

// NewInitialAppSpec creates the only root shape: generation one without a
// parent. It preserves Objective and Reason exactly after non-blank checks.
func NewInitialAppSpec(input AppSpecInput) (AppSpec, error) {
	if err := validateAppSpecInput(input); err != nil {
		return AppSpec{}, err
	}
	spec := AppSpec{
		ref: input.Ref, generation: initialAppSpecGeneration, intent: input.Intent,
		objective: input.Objective, reason: input.Reason,
		confirmedBy: input.ConfirmedBy, confirmedAt: canonicalTime(input.ConfirmedAt),
	}
	spec.hash = hashAppSpec(spec)
	return spec, nil
}

// Amend creates the exact next generation. The receiver remains unchanged;
// the new IntentManifest must retain scope and occur after its parent.
func (spec AppSpec) Amend(input AppSpecInput) (AppSpec, error) {
	if !validAppSpec(spec) {
		return AppSpec{}, domainError(ErrorInvalidArgument, "app_spec")
	}
	if err := validateAppSpecInput(input); err != nil {
		return AppSpec{}, err
	}
	if input.Ref == spec.ref || input.Intent.Ref() == spec.intent.Ref() {
		return AppSpec{}, domainError(ErrorInvalidArgument, "amendment_identity")
	}
	if input.Intent.Actor() != spec.intent.Actor() || input.Intent.Project() != spec.intent.Project() {
		return AppSpec{}, domainError(ErrorScopeConflict, "intent_scope")
	}
	if input.Intent.SubmittedAt().Before(spec.confirmedAt) {
		return AppSpec{}, domainError(ErrorInvalidArgument, "intent_submitted_at")
	}
	generation, err := nextAppSpecGeneration(spec.generation)
	if err != nil {
		return AppSpec{}, err
	}
	amended := AppSpec{
		ref: input.Ref, generation: generation, intent: input.Intent,
		parentRef: spec.ref, parentHash: spec.hash,
		objective: input.Objective, reason: input.Reason,
		confirmedBy: input.ConfirmedBy, confirmedAt: canonicalTime(input.ConfirmedAt),
	}
	amended.hash = hashAppSpec(amended)
	return amended, nil
}

func (spec AppSpec) Ref() AppSpecRef               { return spec.ref }
func (spec AppSpec) Generation() AppSpecGeneration { return spec.generation }
func (spec AppSpec) Intent() IntentManifest        { return spec.intent }
func (spec AppSpec) Objective() string             { return spec.objective }
func (spec AppSpec) ParentHash() string            { return spec.parentHash }
func (spec AppSpec) Reason() string                { return spec.reason }
func (spec AppSpec) ConfirmedBy() ActorRef         { return spec.confirmedBy }
func (spec AppSpec) ConfirmedAt() time.Time        { return spec.confirmedAt }
func (spec AppSpec) Hash() string                  { return spec.hash }
func (spec AppSpec) isRoot() bool                  { return spec.generation == initialAppSpecGeneration }
func (spec AppSpec) ParentRef() (AppSpecRef, bool) {
	return spec.parentRef, validAppSpecRef(spec.parentRef)
}

func validateAppSpecInput(input AppSpecInput) error {
	switch {
	case !validAppSpecRef(input.Ref):
		return domainError(ErrorInvalidRef, "app_spec_ref")
	case !validIntentManifest(input.Intent):
		return domainError(ErrorInvalidArgument, "intent_manifest")
	case strings.TrimSpace(input.Objective) == "":
		return domainError(ErrorInvalidArgument, "objective")
	case strings.TrimSpace(input.Reason) == "":
		return domainError(ErrorInvalidArgument, "reason")
	case !validActorRef(input.ConfirmedBy):
		return domainError(ErrorInvalidRef, "confirmed_by")
	case !validTransitionTime(input.ConfirmedAt, input.Intent.SubmittedAt()):
		return domainError(ErrorInvalidArgument, "confirmed_at")
	default:
		return nil
	}
}

func validAppSpec(spec AppSpec) bool {
	if err := validateAppSpecInput(AppSpecInput{
		Ref: spec.ref, Intent: spec.intent, Objective: spec.objective,
		Reason: spec.reason, ConfirmedBy: spec.confirmedBy, ConfirmedAt: spec.confirmedAt,
	}); err != nil {
		return false
	}
	if spec.generation == 0 || spec.hash != hashAppSpec(spec) {
		return false
	}
	if spec.isRoot() {
		return !validAppSpecRef(spec.parentRef) && spec.parentHash == ""
	}
	return validAppSpecRef(spec.parentRef) && spec.parentRef != spec.ref && validCanonicalSHA256(spec.parentHash)
}

func hashAppSpec(spec AppSpec) string {
	digest := sha256.New()
	writeHashField(digest, "orquesta.app-spec")
	writeHashField(digest, spec.ref.String())
	writeHashField(digest, strconv.FormatUint(uint64(spec.generation), 10))
	writeHashField(digest, spec.intent.Ref().String())
	writeHashField(digest, spec.intent.Hash())
	writeHashField(digest, spec.parentRef.String())
	writeHashField(digest, spec.parentHash)
	writeHashField(digest, spec.objective)
	writeHashField(digest, spec.reason)
	writeHashField(digest, spec.confirmedBy.String())
	writeHashField(digest, spec.confirmedAt.Format(time.RFC3339Nano))
	return hex.EncodeToString(digest.Sum(nil))
}

func nextAppSpecGeneration(current AppSpecGeneration) (AppSpecGeneration, error) {
	if current == 0 || uint64(current) == math.MaxUint64 {
		return 0, domainError(ErrorInvalidArgument, "app_spec_generation")
	}
	return current + 1, nil
}

// IsCanonicalAppSpecHash validates the wire and persistence representation of
// an AppSpec hash without making adapters duplicate the format contract.
func IsCanonicalAppSpecHash(value string) bool {
	return validCanonicalSHA256(value)
}

func validCanonicalSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
