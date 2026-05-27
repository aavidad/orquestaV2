package main

import (
	"strconv"
	"strings"
)

func idleSelfImprovementFederatedBacklogContextRefsV0(section idleSelfImprovementBacklogSectionV0) []string {
	if section.LocalAlias == "" {
		return nil
	}
	var refs []string
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_federated_source_kind:", section.SourceKind)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_federated_owner:", section.Owner)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_federated_state:", section.LocalState)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_local_alias:", section.LocalAlias)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_related_txx:", section.RelatedTXX)
	refs = appendFederatedBacklogContextRefV0(refs, "backlog_local_entry_hash:", section.LocalEntryHash)
	if section.SourceIndexLine > 0 {
		refs = append(refs, "backlog_federated_source_line:"+strconv.Itoa(section.SourceIndexLine))
	}
	return compactServerStackStringsV0(refs)
}

func appendFederatedBacklogContextRefV0(refs []string, prefix string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return refs
	}
	return append(refs, prefix+strings.TrimSpace(value))
}
