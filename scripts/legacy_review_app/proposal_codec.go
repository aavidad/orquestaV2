// Este fichero define la representación JSONL canónica de las propuestas.
// No decide ni publica: valida bytes por encargo de la autoridad del almacén.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const maxProposalLineBytes = 1 << 20

func decodeCanonicalProposal(raw []byte) (proposal, error) {
	var item proposal
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return item, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return item, errors.New("JSON posterior al registro")
	}
	canonical, err := encodeCanonicalProposal(item)
	if err != nil {
		return item, err
	}
	if !bytes.Equal(raw, canonical) {
		return item, errors.New("registro JSON no canónico o ambiguo")
	}
	return item, nil
}

func encodeCanonicalProposal(item proposal) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(item); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'}), nil
}

func validateStoredProposals(proposals []proposal) error {
	latest := map[string]int{}
	seenKeys := map[string]struct{}{}
	for _, item := range proposals {
		request := storedProposalRequest(item)
		normalized := normalizeProposalRequest(request)
		if !equalProposalRequests(request, normalized) {
			return errors.New("historial de propuestas no canónico")
		}
		request = normalized
		if item.SchemaVersion != 1 || item.ProposalRevision != latest[item.ItemRef]+1 ||
			item.ExpectedRevision != latest[item.ItemRef] ||
			validateProposalRequest(inventoryItem{ID: item.ItemRef, Revision: item.ItemRevision}, request) != nil ||
			digestRequest(request) != item.RequestDigest ||
			proposalReference(item.ItemRef, item.ProposalRevision, item.RequestDigest) != item.ProposalRef {
			return errors.New("historial de propuestas no canónico")
		}
		created, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
		if err != nil || created.UTC().Format(time.RFC3339Nano) != item.CreatedAt {
			return errors.New("fecha de propuesta no canónica")
		}
		if _, exists := seenKeys[item.IdempotencyKey]; exists {
			return errors.New("clave idempotente duplicada")
		}
		seenKeys[item.IdempotencyKey] = struct{}{}
		latest[item.ItemRef] = item.ProposalRevision
	}
	return nil
}

func storedProposalRequest(item proposal) proposalRequest {
	return proposalRequest{
		ItemRef:          item.ItemRef,
		ItemRevision:     item.ItemRevision,
		ExpectedRevision: item.ExpectedRevision,
		Disposition:      item.Disposition,
		Reason:           item.Reason,
		FoundedSolution:  item.FoundedSolution,
		Confidence:       item.Confidence,
		RuleCompliance:   item.RuleCompliance,
		RuleNotes:        item.RuleNotes,
		ActorRef:         item.ActorRef,
		ProjectRef:       item.ProjectRef,
		IdempotencyKey:   item.IdempotencyKey,
	}
}

func equalProposalRequests(left, right proposalRequest) bool {
	if left.ItemRef != right.ItemRef ||
		left.ItemRevision != right.ItemRevision ||
		left.ExpectedRevision != right.ExpectedRevision ||
		left.Disposition != right.Disposition ||
		left.Reason != right.Reason ||
		left.FoundedSolution != right.FoundedSolution ||
		left.Confidence != right.Confidence ||
		left.RuleNotes != right.RuleNotes ||
		left.ActorRef != right.ActorRef ||
		left.ProjectRef != right.ProjectRef ||
		left.IdempotencyKey != right.IdempotencyKey ||
		len(left.RuleCompliance) != len(right.RuleCompliance) {
		return false
	}
	for rule, compliant := range left.RuleCompliance {
		if right.RuleCompliance[rule] != compliant {
			return false
		}
	}
	return true
}

func normalizeProposalRequest(request proposalRequest) proposalRequest {
	request.Reason = strings.TrimSpace(request.Reason)
	request.FoundedSolution = strings.TrimSpace(request.FoundedSolution)
	request.RuleNotes = strings.TrimSpace(request.RuleNotes)
	request.ActorRef = strings.TrimSpace(request.ActorRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.RuleCompliance = cloneRules(request.RuleCompliance)
	return request
}

func digestRequest(request proposalRequest) string {
	content, _ := json.Marshal(request)
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func proposalReference(itemID string, revision int, requestDigest string) string {
	digest := sha256.Sum256([]byte(itemID + "\x00" + fmt.Sprint(revision) + "\x00" + requestDigest))
	return "propuesta:sha256:" + hex.EncodeToString(digest[:])
}
