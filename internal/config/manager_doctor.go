package config

import (
	"context"
	"errors"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Doctor classifies registry proposals without reading or mutating the
// document store.
func (manager *Manager) Doctor(ctx context.Context, request DoctorRequest) (DoctorReport, error) {
	if err := manager.ready(ctx, ErrorDoctorInvalid); err != nil {
		return DoctorReport{}, err
	}
	if len(request.Proposals) == 0 {
		return DoctorReport{}, managerError(ErrorDoctorInvalid, errors.New("config_doctor_proposals_required"))
	}
	state := newDoctorState(manager.registry)
	report := DoctorReport{Accepted: []DoctorDecision{}, Conflicts: []DoctorConflict{}}
	for index, proposal := range request.Proposals {
		mode, conflicts := state.classify(index, proposal, manager.registry)
		if len(conflicts) != 0 {
			report.Conflicts = append(report.Conflicts, conflicts...)
			continue
		}
		report.Accepted = append(report.Accepted, DoctorDecision{DoctorProposal: proposal, Mode: mode})
		state.reserve(proposal, mode)
	}
	return report, nil
}

type doctorState struct {
	keys        map[Key]Key
	goNames     map[string]Key
	semantics   map[string]Key
	environment map[string]Key
	aliases     map[string]Key
	proposals   map[Key]Key
}

func newDoctorState(loaded registry) *doctorState {
	state := &doctorState{
		keys: make(map[Key]Key, len(loaded.keys)), goNames: make(map[string]Key, len(loaded.keys)),
		semantics: make(map[string]Key, len(loaded.keys)), environment: make(map[string]Key, len(loaded.keys)+len(loaded.aliases)),
		aliases: make(map[string]Key, len(loaded.aliases)), proposals: make(map[Key]Key),
	}
	for _, definition := range loaded.keys {
		state.keys[definition.Key] = definition.Key
		state.goNames[definition.GoName] = definition.Key
		state.semantics[definition.SemanticRef] = definition.Key
		state.environment[definition.EnvAlias] = definition.Key
	}
	for _, alias := range loaded.aliases {
		if alias.Kind == AliasKindEnvironment {
			state.environment[alias.Name] = alias.Target
		} else if alias.Kind == AliasKindTOMLKey {
			state.aliases[alias.Name] = alias.Target
		}
	}
	return state
}

func (state *doctorState) classify(index int, proposal DoctorProposal, loaded registry) (DoctorMode, []DoctorConflict) {
	mode := DoctorNew
	if proposal.TargetKey != "" {
		mode = DoctorReuse
		if proposal.Alias != "" || proposal.RemoveAfterRevision != "" {
			mode = DoctorReplace
		}
	}
	conflicts := make([]DoctorConflict, 0)
	add := func(field string, code DoctorConflictCode, existing Key) {
		conflicts = append(conflicts, DoctorConflict{
			ProposalIndex: index, Key: proposal.Key, Field: field, Code: code, ExistingKey: existing,
		})
	}

	if !validDoctorKey(proposal.Key) {
		add("key", DoctorConflictKey, "")
	} else if existing, found := state.overlappingKey(proposal.Key); found {
		add("key", DoctorConflictKey, existing)
	}
	if existing, found := state.proposals[proposal.Key]; found {
		add("key", DoctorConflictKey, existing)
	}
	target, targetFound := loaded.definition(proposal.TargetKey)
	if proposal.TargetKey != "" && !targetFound {
		add("target_key", DoctorConflictTarget, proposal.TargetKey)
	}

	switch mode {
	case DoctorReuse:
		if targetFound && proposal.SemanticRef != target.SemanticRef {
			add("semantic_ref", DoctorConflictSemantic, target.Key)
		}
		if targetFound && proposal.GoName != "" && proposal.GoName != target.GoName {
			add("go_name", DoctorConflictGoName, target.Key)
		}
		if targetFound && proposal.EnvAlias != "" && proposal.EnvAlias != target.EnvAlias {
			add("env_alias", DoctorConflictEnvAlias, target.Key)
		}
		if targetFound && proposal.Type != "" && proposal.Type != string(target.Type) {
			add("type", DoctorConflictType, target.Key)
		}
		if targetFound && proposal.Scope != "" && proposal.Scope != target.Scope {
			add("scope", DoctorConflictScope, target.Key)
		}
		if existing, found := state.keys[proposal.Key]; found && existing != proposal.TargetKey {
			add("key", DoctorConflictKey, existing)
		}
		if existing, found := state.aliases[string(proposal.Key)]; found && existing != proposal.TargetKey {
			add("key", DoctorConflictAlias, existing)
		}
		if proposal.Alias != "" {
			add("alias", DoctorConflictShape, "")
		}
		if proposal.RemoveAfterRevision != "" {
			add("remove_after_revision", DoctorConflictShape, "")
		}
	case DoctorReplace:
		if existing, found := state.keys[proposal.Key]; found {
			add("key", DoctorConflictKey, existing)
		}
		if existing, found := state.aliases[string(proposal.Key)]; found {
			add("key", DoctorConflictAlias, existing)
		}
		if proposal.Alias == "" || proposal.Alias != string(proposal.TargetKey) {
			add("alias", DoctorConflictAlias, proposal.TargetKey)
		} else if existing, found := state.aliases[proposal.Alias]; found {
			add("alias", DoctorConflictAlias, existing)
		}
		removeAfter, removeAfterErr := parseRegistryRevision(proposal.RemoveAfterRevision)
		currentRevision, currentRevisionErr := parseRegistryRevision(loaded.revision)
		if removeAfterErr != nil || currentRevisionErr != nil || removeAfter.compare(currentRevision) <= 0 {
			add("remove_after_revision", DoctorConflictRetirement, proposal.TargetKey)
		}
		state.checkOptionalIdentity(proposal, add)
	case DoctorNew:
		if existing, found := state.keys[proposal.Key]; found {
			add("key", DoctorConflictKey, existing)
		}
		if existing, found := state.aliases[string(proposal.Key)]; found {
			add("key", DoctorConflictAlias, existing)
		}
		if proposal.Alias != "" {
			add("alias", DoctorConflictAlias, "")
		}
		if proposal.RemoveAfterRevision != "" {
			add("remove_after_revision", DoctorConflictRetirement, "")
		}
		state.checkRequiredIdentity(proposal, add)
	}
	return mode, conflicts
}

func (state *doctorState) checkOptionalIdentity(proposal DoctorProposal, add func(string, DoctorConflictCode, Key)) {
	if strings.TrimSpace(proposal.SemanticRef) == "" {
		add("semantic_ref", DoctorConflictSemantic, "")
	} else if existing, found := state.semantics[proposal.SemanticRef]; found {
		add("semantic_ref", DoctorConflictSemantic, existing)
	}
	if proposal.GoName != "" {
		if !validDoctorGoName(proposal.GoName) {
			add("go_name", DoctorConflictGoName, "")
		} else if existing, found := state.goNames[proposal.GoName]; found {
			add("go_name", DoctorConflictGoName, existing)
		}
	}
	if proposal.EnvAlias != "" {
		if !validEnvironmentName(proposal.EnvAlias) {
			add("env_alias", DoctorConflictEnvAlias, "")
		} else if existing, found := state.environment[proposal.EnvAlias]; found {
			add("env_alias", DoctorConflictEnvAlias, existing)
		}
	}
	if proposal.Type != "" && !validDoctorType(proposal.Type) {
		add("type", DoctorConflictType, "")
	}
	if proposal.Scope != "" && strings.TrimSpace(proposal.Scope) != proposal.Scope {
		add("scope", DoctorConflictScope, "")
	}
}

func (state *doctorState) checkRequiredIdentity(proposal DoctorProposal, add func(string, DoctorConflictCode, Key)) {
	state.checkOptionalIdentity(proposal, add)
	if proposal.GoName == "" {
		add("go_name", DoctorConflictGoName, "")
	}
	if proposal.EnvAlias == "" {
		add("env_alias", DoctorConflictEnvAlias, "")
	}
	if proposal.Type == "" {
		add("type", DoctorConflictType, "")
	}
	if strings.TrimSpace(proposal.Scope) == "" {
		add("scope", DoctorConflictScope, "")
	}
}

func (state *doctorState) reserve(proposal DoctorProposal, mode DoctorMode) {
	reservation := proposal.Key
	if proposal.TargetKey != "" {
		reservation = proposal.TargetKey
	}
	state.proposals[proposal.Key] = reservation
	if mode == DoctorReuse {
		return
	}
	state.keys[proposal.Key] = proposal.Key
	state.semantics[proposal.SemanticRef] = proposal.Key
	if proposal.GoName != "" {
		state.goNames[proposal.GoName] = proposal.Key
	}
	if proposal.EnvAlias != "" {
		state.environment[proposal.EnvAlias] = proposal.Key
	}
	if proposal.Alias != "" {
		state.aliases[proposal.Alias] = proposal.Key
	}
}

func validDoctorType(value string) bool {
	switch valueType(value) {
	case valueTypeString, valueTypeInteger, valueTypeDuration, valueTypePath, valueTypeStringList, valueTypeCredentialRef:
		return true
	default:
		return false
	}
}

func validDoctorGoName(value string) bool {
	if !token.IsIdentifier(value) {
		return false
	}
	first, _ := utf8.DecodeRuneInString(value)
	return unicode.IsUpper(first)
}

func validDoctorKey(key Key) bool {
	value := string(key)
	separator := strings.LastIndexByte(value, '.')
	return validCanonicalKey(value) && separator > 0 && separator < len(value)-1
}

func (state *doctorState) overlappingKey(candidate Key) (Key, bool) {
	var match Key
	consider := func(existing Key) {
		if doctorKeysOverlap(candidate, existing) && (match == "" || existing < match) {
			match = existing
		}
	}
	for existing := range state.keys {
		consider(existing)
	}
	for alias := range state.aliases {
		consider(Key(alias))
	}
	for existing := range state.proposals {
		consider(existing)
	}
	return match, match != ""
}

func doctorKeysOverlap(left, right Key) bool {
	leftValue, rightValue := string(left), string(right)
	return strings.HasPrefix(leftValue, rightValue+".") || strings.HasPrefix(rightValue, leftValue+".")
}
