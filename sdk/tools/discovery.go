package tools

import "orquesta/internal/tooling"

const MaxToolsPerNamespace = tooling.MaxToolsPerNamespace

type Namespace struct {
	Name      string `json:"name"`
	ToolCount uint32 `json:"tool_count"`
	Digest    string `json:"digest"`
}

// Discovery is a public read-only projection of TLS-04. It owns neither specs
// nor limits; both remain delegated to the internal canonical registry/index.
type Discovery struct {
	internal *tooling.ToolDiscovery
}

func NewDiscovery(catalog *Catalog) (*Discovery, error) {
	if catalog == nil || catalog.registry == nil {
		return nil, ErrDiscoveryInvalid
	}
	discovery, err := tooling.NewToolDiscovery(catalog.registry)
	if err != nil {
		if tooling.ErrorCode(err) == tooling.ErrorToolNamespaceTooLarge {
			return nil, ErrNamespaceTooLarge
		}
		return nil, ErrDiscoveryInvalid
	}
	return &Discovery{internal: discovery}, nil
}

func (discovery *Discovery) ListNamespaces() []Namespace {
	if discovery == nil || discovery.internal == nil {
		return nil
	}
	internal := discovery.internal.ListNamespaces()
	result := make([]Namespace, len(internal))
	for index, namespace := range internal {
		result[index] = Namespace{
			Name: namespace.Name, ToolCount: namespace.ToolCount, Digest: namespace.Digest,
		}
	}
	return result
}

func (discovery *Discovery) ListNamespace(name string) ([]Registration, bool) {
	if discovery == nil || discovery.internal == nil {
		return nil, false
	}
	internal, found := discovery.internal.ListNamespace(name)
	if !found {
		return nil, false
	}
	result := make([]Registration, len(internal))
	for index, registration := range internal {
		result[index] = fromInternal(registration)
	}
	return result, true
}

func (discovery *Discovery) Digest() string {
	if discovery == nil || discovery.internal == nil {
		return ""
	}
	return discovery.internal.Digest()
}
