package tooling

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const (
	MaxToolsPerNamespace       = 32
	ErrorToolDiscoveryInvalid  = "tooling.discovery_invalid"
	ErrorToolNamespaceTooLarge = "tooling.namespace_too_large"
)

type ToolNamespace struct {
	Name      string `json:"name"`
	ToolCount uint32 `json:"tool_count"`
	Digest    string `json:"digest"`
}

type toolIdentity struct {
	id, version string
}

// ToolDiscovery is a derived read-model. It stores only exact identities;
// every returned specification still comes from the immutable TLS-01 registry.
type ToolDiscovery struct {
	registry   *Registry
	namespaces []ToolNamespace
	byName     map[string][]toolIdentity
	digest     string
}

func NewToolDiscovery(registry *Registry) (*ToolDiscovery, error) {
	if registry == nil {
		return nil, contractError(ErrorToolDiscoveryInvalid, "registry")
	}
	registrations := registry.List()
	grouped := make(map[string][]Registration)
	for _, registration := range registrations {
		namespace := strings.SplitN(registration.Spec.ID, ".", 2)[0]
		grouped[namespace] = append(grouped[namespace], registration)
		if len(grouped[namespace]) > MaxToolsPerNamespace {
			return nil, contractError(ErrorToolNamespaceTooLarge, namespace)
		}
	}
	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	sort.Strings(names)
	namespaces := make([]ToolNamespace, 0, len(names))
	byName := make(map[string][]toolIdentity, len(names))
	for _, name := range names {
		entries := grouped[name]
		identities := make([]toolIdentity, len(entries))
		for index, entry := range entries {
			identities[index] = toolIdentity{id: entry.Spec.ID, version: entry.Spec.Version}
		}
		byName[name] = identities
		namespaces = append(namespaces, ToolNamespace{
			Name: name, ToolCount: uint32(len(entries)), Digest: toolNamespaceDigest(name, entries),
		})
	}
	return &ToolDiscovery{
		registry: registry, namespaces: namespaces, byName: byName,
		digest: toolDiscoveryDigest(registry.Digest(), namespaces),
	}, nil
}

// ListNamespaces returns only compact descriptors. Tool schemas remain lazy
// until a caller selects one exact namespace.
func (discovery *ToolDiscovery) ListNamespaces() []ToolNamespace {
	if discovery == nil {
		return nil
	}
	result := make([]ToolNamespace, len(discovery.namespaces))
	copy(result, discovery.namespaces)
	return result
}

// ListNamespace returns at most MaxToolsPerNamespace exact registrations.
func (discovery *ToolDiscovery) ListNamespace(name string) ([]Registration, bool) {
	if discovery == nil || discovery.registry == nil {
		return nil, false
	}
	identities, found := discovery.byName[name]
	if !found {
		return nil, false
	}
	result := make([]Registration, len(identities))
	for index, identity := range identities {
		registration, exists := discovery.registry.Lookup(identity.id, identity.version)
		if !exists {
			return nil, false
		}
		result[index] = registration
	}
	return result, true
}

func (discovery *ToolDiscovery) Digest() string {
	if discovery == nil {
		return ""
	}
	return discovery.digest
}

func toolNamespaceDigest(name string, entries []Registration) string {
	digests := make([]string, len(entries))
	for index, entry := range entries {
		digests[index] = entry.Digest
	}
	return digestToolDiscoveryValue(struct {
		Contract string   `json:"contract"`
		Name     string   `json:"name"`
		Tools    []string `json:"tools"`
	}{"orquesta.tooling.namespace.v1", name, digests})
}

func toolDiscoveryDigest(registryDigest string, namespaces []ToolNamespace) string {
	return digestToolDiscoveryValue(struct {
		Contract   string          `json:"contract"`
		Registry   string          `json:"registry"`
		Namespaces []ToolNamespace `json:"namespaces"`
	}{"orquesta.tooling.discovery.v1", registryDigest, namespaces})
}

func digestToolDiscoveryValue(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("canonical tool discovery cannot fail JSON encoding: " + err.Error())
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:])
}
