package stages

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"strconv"
)

const digestHexLength = sha256.Size * 2

// Digest is a canonical SHA-256 over every semantic field in a Catalog or
// Template. It is content identity, not capability accreditation.
type Digest struct{ value string }

func NewDigest(value string) (Digest, error) {
	if !validDigest(value) {
		return Digest{}, domainError(ErrorInvalidDigest, "digest")
	}
	return Digest{value: value}, nil
}

func (value Digest) String() string { return value.value }

func (value Catalog) Digest() Digest {
	digest := sha256.New()
	writeDigestField(digest, "orquesta.wizard.stages.catalog-digest.v1")
	writeDigestField(digest, value.version.value)
	writeDigestField(digest, strconv.Itoa(len(value.templates)))
	for _, template := range value.templates {
		writeDigestField(digest, template.ref.value)
		writeDigestField(digest, template.Digest().value)
	}
	return digestValue(digest)
}

func (value Template) Digest() Digest {
	digest := sha256.New()
	writeDigestField(digest, "orquesta.wizard.stages.template-digest.v1")
	writeDigestField(digest, value.ref.value)
	writeDigestField(digest, strconv.Itoa(len(value.roadmapCapabilityRefs)))
	for _, ref := range value.roadmapCapabilityRefs {
		writeDigestField(digest, ref.value)
	}
	writeDigestField(digest, strconv.Itoa(len(value.stages)))
	for _, stage := range value.stages {
		writeDigestField(digest, stage.ref.value)
		writeDigestField(digest, strconv.Itoa(stage.sequence))
		writeDigestField(digest, string(stage.phase))
		writeDigestField(digest, strconv.Itoa(len(stage.dependsOn)))
		for _, dependency := range stage.dependsOn {
			writeDigestField(digest, dependency.value)
		}
	}
	writeDigestField(digest, strconv.Itoa(len(value.units)))
	for _, unit := range value.units {
		writeUnitDigest(digest, unit)
	}
	return digestValue(digest)
}

func writeUnitDigest(digest hash.Hash, unit Unit) {
	writeDigestField(digest, unit.ref.value)
	writeDigestField(digest, unit.stageRef.value)
	writeDigestField(digest, strconv.Itoa(len(unit.dependsOn)))
	for _, dependency := range unit.dependsOn {
		writeDigestField(digest, dependency.value)
	}
	writeDigestField(digest, unit.role.value)
	writeDigestField(digest, strconv.Itoa(len(unit.writeSet)))
	for _, scope := range unit.writeSet {
		writeDigestField(digest, scope.Path)
	}
	writeDigestField(digest, strconv.Itoa(len(unit.requiredTests)))
	for _, required := range unit.requiredTests {
		writeDigestField(digest, required.Ref.value)
		writeDigestField(digest, string(required.Kind))
		writeDigestField(digest, strconv.Itoa(len(required.CriterionRefs)))
		for _, criterion := range required.CriterionRefs {
			writeDigestField(digest, criterion.value)
		}
	}
	writeDigestField(digest, strconv.Itoa(len(unit.acceptanceCriteria)))
	for _, criterion := range unit.acceptanceCriteria {
		writeDigestField(digest, criterion.Ref.value)
	}
	writeDigestField(digest, strconv.Itoa(len(unit.effects)))
	for _, effect := range unit.effects {
		writeDigestField(digest, effect.Ref.value)
		writeDigestField(digest, string(effect.Kind))
		writeDigestBool(digest, effect.ApprovalRequired)
	}
	writeDigestField(digest, string(unit.policy.Effort))
	writeDigestField(digest, string(unit.policy.Security.Risk))
	writeDigestField(digest, string(unit.policy.Security.Approval))
	writeDigestBool(digest, unit.policy.Security.LeastPrivilege)
	writeDigestBool(digest, unit.policy.Security.SensitiveInputs)
}

func writeDigestBool(digest hash.Hash, value bool) {
	if value {
		writeDigestField(digest, "true")
		return
	}
	writeDigestField(digest, "false")
}

func writeDigestField(digest hash.Hash, value string) {
	_, _ = digest.Write([]byte(strconv.Itoa(len(value))))
	_, _ = digest.Write([]byte{':'})
	_, _ = digest.Write([]byte(value))
	_, _ = digest.Write([]byte{0})
}

func digestValue(digest hash.Hash) Digest {
	return Digest{value: hex.EncodeToString(digest.Sum(nil))}
}

func validDigest(value string) bool {
	if len(value) != digestHexLength {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
