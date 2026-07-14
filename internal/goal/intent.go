package goal

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strings"
	"time"
)

// IntentManifestInput contains all values frozen into an IntentManifest.
type IntentManifestInput struct {
	Ref         IntentRef
	Actor       ActorRef
	Project     ProjectRef
	Statement   string
	SubmittedAt time.Time
}

// IntentManifest is an immutable record of submitted intent with an integrity
// hash. Its opaque identity is distinct from that hash, which covers every
// field using a length-prefixed canonical representation.
type IntentManifest struct {
	ref         IntentRef
	actor       ActorRef
	project     ProjectRef
	statement   string
	submittedAt time.Time
	hash        string
}

// NewIntentManifest validates and freezes an intent. The caller supplies time
// and identifiers; the domain reads neither clocks nor global configuration.
func NewIntentManifest(input IntentManifestInput) (IntentManifest, error) {
	if !validIntentRef(input.Ref) {
		return IntentManifest{}, domainError(ErrorInvalidRef, "intent_ref")
	}
	if !validActorRef(input.Actor) {
		return IntentManifest{}, domainError(ErrorInvalidRef, "actor_ref")
	}
	if !validProjectRef(input.Project) {
		return IntentManifest{}, domainError(ErrorInvalidRef, "project_ref")
	}
	if strings.TrimSpace(input.Statement) == "" {
		return IntentManifest{}, domainError(ErrorInvalidArgument, "statement")
	}
	if input.SubmittedAt.IsZero() {
		return IntentManifest{}, domainError(ErrorInvalidArgument, "submitted_at")
	}

	submittedAt := canonicalTime(input.SubmittedAt)
	manifest := IntentManifest{
		ref:         input.Ref,
		actor:       input.Actor,
		project:     input.Project,
		statement:   input.Statement,
		submittedAt: submittedAt,
	}
	manifest.hash = hashIntentManifest(manifest)
	return manifest, nil
}

func (manifest IntentManifest) Ref() IntentRef         { return manifest.ref }
func (manifest IntentManifest) Actor() ActorRef        { return manifest.actor }
func (manifest IntentManifest) Project() ProjectRef    { return manifest.project }
func (manifest IntentManifest) Statement() string      { return manifest.statement }
func (manifest IntentManifest) SubmittedAt() time.Time { return manifest.submittedAt }
func (manifest IntentManifest) Hash() string           { return manifest.hash }

func hashIntentManifest(manifest IntentManifest) string {
	digest := sha256.New()
	writeHashField(digest, "orquesta.intent-manifest")
	writeHashField(digest, manifest.ref.String())
	writeHashField(digest, manifest.actor.String())
	writeHashField(digest, manifest.project.String())
	writeHashField(digest, manifest.statement)
	writeHashField(digest, manifest.submittedAt.Format(time.RFC3339Nano))
	return hex.EncodeToString(digest.Sum(nil))
}

func writeHashField(digest hash.Hash, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(value))
}

func canonicalTime(value time.Time) time.Time {
	return value.Round(0).UTC()
}
