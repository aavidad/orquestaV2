package tooling

import "orquesta/internal/identity"

// LookupSkillTool exposes the narrow, immutable projection consumed by the
// skill registry without coupling that registry to Registry itself.
func (registry *Registry) LookupSkillTool(id, version string) (SkillToolRegistration, bool) {
	registration, found := registry.Lookup(id, version)
	if !found {
		return SkillToolRegistration{}, false
	}
	return SkillToolRegistration{
		ID: registration.Spec.ID, Version: registration.Spec.Version,
		SpecDigest:  registration.Digest,
		Permissions: append([]identity.Permission(nil), registration.Spec.Permissions...),
	}, true
}
