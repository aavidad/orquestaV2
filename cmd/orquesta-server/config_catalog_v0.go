package main

import (
	"fmt"

	orquestaconfig "orquesta/modulos/orquesta-config"
)

type serverProjectConfigPolicySpecV0 struct {
	Sensitive         bool
	EditabilityReason string
	RestartBehavior   orquestaconfig.RestartBehaviorV0
}

func serverProjectConfigCatalogV0() ([]orquestaconfig.CatalogEntryV0, error) {
	leaves, err := orquestaconfig.ReflectJSONLeavesV0(serverProjectConfigFileV0{})
	if err != nil {
		return nil, err
	}
	leavesByPointer := make(map[string]orquestaconfig.JSONLeafV0, len(leaves))
	for _, leaf := range leaves {
		leavesByPointer[leaf.Pointer] = leaf
	}
	policies := make([]orquestaconfig.LeafPolicyV0, 0, len(leaves))
	seen := make(map[string]bool, len(leaves))
	for _, table := range []map[string]serverProjectConfigPolicySpecV0{
		serverProjectConfigAgentPolicyTableV0,
		serverProjectConfigOpsPolicyTableV0,
		serverProjectConfigCorePolicyTableV0,
	} {
		for pointer := range table {
			if seen[pointer] {
				return nil, fmt.Errorf("server project config catalog: duplicate explicit policy for %s", pointer)
			}
			leaf, ok := leavesByPointer[pointer]
			if !ok {
				return nil, fmt.Errorf("server project config catalog: policy does not match a JSON leaf: %s", pointer)
			}
			policy, ok := serverProjectConfigPolicyFromExactTableV0(leaf, table)
			if !ok {
				return nil, fmt.Errorf("server project config catalog: missing explicit policy for %s", pointer)
			}
			seen[pointer] = true
			policies = append(policies, policy)
		}
	}
	if len(seen) != len(leaves) {
		for _, leaf := range leaves {
			if !seen[leaf.Pointer] {
				return nil, fmt.Errorf("server project config catalog: missing explicit policy for %s", leaf.Pointer)
			}
		}
	}
	catalog, err := orquestaconfig.BuildCatalogV0(serverProjectConfigFileV0{}, policies)
	if err != nil {
		return nil, fmt.Errorf("server project config catalog: %w", err)
	}
	return catalog, nil
}

func serverProjectConfigPolicyForLeafV0(leaf orquestaconfig.JSONLeafV0) (orquestaconfig.LeafPolicyV0, bool) {
	if policy, ok := serverProjectConfigAgentsPolicyV0(leaf); ok {
		return policy, true
	}
	if policy, ok := serverProjectConfigOpsPolicyV0(leaf); ok {
		return policy, true
	}
	return serverProjectConfigCorePolicyV0(leaf)
}

func serverProjectConfigPolicyFromExactTableV0(leaf orquestaconfig.JSONLeafV0, table map[string]serverProjectConfigPolicySpecV0) (orquestaconfig.LeafPolicyV0, bool) {
	spec, ok := table[leaf.Pointer]
	if !ok {
		return orquestaconfig.LeafPolicyV0{}, false
	}
	return orquestaconfig.LeafPolicyV0{
		Pointer: leaf.Pointer, Presence: leaf.Presence, Constraints: orquestaconfig.ConstraintsV0{},
		Sensitive: spec.Sensitive, Editable: false, EditabilityReason: spec.EditabilityReason,
		RestartBehavior: spec.RestartBehavior, SetterRef: "",
	}, true
}
