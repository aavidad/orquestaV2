package gaps

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"orquesta/internal/wizard/catalog"
)

func TestResultSnapshotRoundTripIsExactAndDeterministic(t *testing.T) {
	t.Parallel()

	result := richSnapshotResult(t)
	if len(result.Issues()) == 0 ||
		len(result.Questions()) == 0 ||
		len(result.DefaultProposals()) == 0 ||
		len(result.PackRefs()) == 0 {
		t.Fatal("rich result does not exercise every snapshot collection")
	}

	first, firstDigest, err := MarshalResultSnapshot(result)
	if err != nil {
		t.Fatal(err)
	}
	second, secondDigest, err := MarshalResultSnapshot(result)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("equal Results produced different snapshots")
	}
	if firstDigest != secondDigest ||
		firstDigest != snapshotExternalDigest(first) {
		t.Fatal("equal Results produced different external digests")
	}

	document := decodeResultSnapshotDocument(t, first)
	suppliedDigest := document.Digest
	document.Digest = ""
	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	if got, want := suppliedDigest, hex.EncodeToString(digest[:]); got != want {
		t.Fatalf("digest=%q want=%q", got, want)
	}

	restored, err := RestoreResultSnapshot(first, firstDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, result) {
		t.Fatalf("restored Result differs:\ngot:  %#v\nwant: %#v", restored, result)
	}
	reencoded, reencodedDigest, err := MarshalResultSnapshot(restored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(reencoded, first) {
		t.Fatal("restored Result did not reproduce the canonical snapshot")
	}
	if reencodedDigest != firstDigest {
		t.Fatal("restored Result did not reproduce the external digest")
	}
}

func TestResultSnapshotCanonicalEmptyCollectionsGolden(t *testing.T) {
	t.Parallel()

	result, err := Evaluate(Input{
		Facts: Facts{
			IntegrationAuth:        DeclarationDeclared,
			IntegrationCriticality: DeclarationDeclared,
			TargetUsers:            DeclarationDeclared,
		},
		Selections: allDimensionSelections(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Issues()) != 0 ||
		len(result.Questions()) != 0 ||
		len(result.DefaultProposals()) != 0 ||
		len(result.PackRefs()) != 0 {
		t.Fatalf("fully answered result is not empty: %#v", result)
	}

	encoded, digest, err := MarshalResultSnapshot(result)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"schema":"orquesta.wizard.gaps.result-snapshot.v1","result_schema":"orquesta.wizard.gaps.v1","issues":[],"questions":[],"default_proposals":[],"pack_refs":[],"digest":"8befdf498b6beae4c35edf03c821a324233e0d822a43c2250ceb50027173c3d2"}`
	if string(encoded) != want {
		t.Fatalf("snapshot=%s\nwant=%s", encoded, want)
	}
	const wantDigest = "0c60053e0455005e42062583045e13c0763061c25fd47e6c4be91a950fc1a07d"
	if digest != wantDigest {
		t.Fatalf("digest=%s want=%s", digest, wantDigest)
	}

	restored, err := RestoreResultSnapshot(encoded, digest)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, result) {
		t.Fatalf("empty result differs:\ngot:  %#v\nwant: %#v", restored, result)
	}
}

func TestResultSnapshotCoversFrozenEvaluatorCorpus(t *testing.T) {
	t.Parallel()

	seeds := evaluatorV1SuccessfulCorpusSeeds()
	for _, pack := range catalog.DomainPackRefs() {
		seeds = append(seeds, evaluatorCorpusSeed{
			ref:   pack.String(),
			input: Input{PackRefs: []catalog.PackRef{pack}},
		})
	}
	for _, seed := range seeds {
		result, err := EvaluateV1(seed.input)
		if err != nil {
			t.Fatalf("%s evaluate: %v", seed.ref, err)
		}
		encoded, digest, err := MarshalResultSnapshot(result)
		if err != nil {
			t.Fatalf("%s marshal: %v", seed.ref, err)
		}
		restored, err := RestoreResultSnapshot(encoded, digest)
		if err != nil {
			t.Fatalf("%s restore: %v", seed.ref, err)
		}
		if !reflect.DeepEqual(restored, result) {
			t.Fatalf("%s round-trip differs", seed.ref)
		}
	}
}

func TestRestoreResultSnapshotRequiresExternalDigestOfCompleteDocument(
	t *testing.T,
) {
	t.Parallel()

	encoded, digest, err := MarshalResultSnapshot(richSnapshotResult(t))
	if err != nil {
		t.Fatal(err)
	}
	wrongDigest := strings.Repeat("0", sha256.Size*2)
	if wrongDigest == digest {
		wrongDigest = strings.Repeat("1", sha256.Size*2)
	}
	for name, expected := range map[string]string{
		"missing":   "",
		"short":     digest[:len(digest)-1],
		"uppercase": strings.ToUpper(digest),
		"wrong":     wrongDigest,
	} {
		t.Run(name, func(t *testing.T) {
			assertInvalidResultSnapshot(t, encoded, expected)
		})
	}

	for name, mutate := range map[string]func(*resultSnapshotDocument){
		"omitted default": func(document *resultSnapshotDocument) {
			document.DefaultProposals = document.DefaultProposals[1:]
		},
		"omitted issue and owning question": func(
			document *resultSnapshotDocument,
		) {
			removed := make(map[IssueRef]struct{})
			for _, ref := range document.Questions[0].DerivedFrom {
				removed[ref] = struct{}{}
			}
			issues := document.Issues[:0]
			for _, issue := range document.Issues {
				if _, found := removed[issue.Ref]; !found {
					issues = append(issues, issue)
				}
			}
			document.Issues = issues
			document.Questions = document.Questions[1:]
		},
	} {
		t.Run(name, func(t *testing.T) {
			document := decodeResultSnapshotDocument(t, encoded)
			mutate(&document)
			rehashedWire, sealErr := marshalResultSnapshotDocument(document)
			if sealErr != nil {
				t.Fatal(sealErr)
			}
			rehashedDocument := decodeResultSnapshotDocument(t, rehashedWire)
			rehashedDocument.Digest = ""
			rehashedPayload, marshalErr := json.Marshal(rehashedDocument)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if got, want := decodeResultSnapshotDocument(t, rehashedWire).Digest,
				resultSnapshotWireChecksum(rehashedPayload); got != want {
				t.Fatal("test mutation did not recompute the v1 wire checksum")
			}
			assertInvalidResultSnapshot(t, rehashedWire, digest)
		})
	}
}

func TestRestoreResultSnapshotRejectsNonCanonicalJSONAndTampering(t *testing.T) {
	t.Parallel()

	encoded, _, err := MarshalResultSnapshot(richSnapshotResult(t))
	if err != nil {
		t.Fatal(err)
	}
	document := decodeResultSnapshotDocument(t, encoded)
	tamperedDigest := document.Digest[:len(document.Digest)-1] + "0"
	if tamperedDigest == document.Digest {
		tamperedDigest = document.Digest[:len(document.Digest)-1] + "1"
	}

	withoutDigest := document
	withoutDigest.Digest = ""
	missingDigest, err := json.Marshal(withoutDigest)
	if err != nil {
		t.Fatal(err)
	}
	uppercaseDigest := bytes.Replace(
		encoded,
		[]byte(document.Digest),
		[]byte(strings.ToUpper(document.Digest)),
		1,
	)
	oversize := bytes.Repeat([]byte("x"), maxResultSnapshotBytes+1)

	tests := map[string][]byte{
		"empty":    nil,
		"oversize": oversize,
		"unknown top-level field": bytes.Replace(
			encoded,
			[]byte(`,"digest":`),
			[]byte(`,"unknown":true,"digest":`),
			1,
		),
		"unknown nested field": bytes.Replace(
			encoded,
			[]byte(`"issues":[{`),
			[]byte(`"issues":[{"unknown":true,`),
			1,
		),
		"trailing data":       append(append([]byte(nil), encoded...), []byte(`{}`)...),
		"trailing whitespace": append(append([]byte(nil), encoded...), '\n'),
		"leading whitespace":  append([]byte(" "), encoded...),
		"duplicate field": bytes.Replace(
			encoded,
			[]byte(`{"schema":`),
			[]byte(`{"schema":"`+ResultSnapshotSchema+`","schema":`),
			1,
		),
		"changed payload": bytes.Replace(
			encoded,
			[]byte(`"pack:billing"`),
			[]byte(`"pack:changed"`),
			1,
		),
		"changed digest": bytes.Replace(
			encoded,
			[]byte(document.Digest),
			[]byte(tamperedDigest),
			1,
		),
		"missing digest":   missingDigest,
		"uppercase digest": uppercaseDigest,
	}
	for name, candidate := range tests {
		t.Run(name, func(t *testing.T) {
			assertInvalidResultSnapshot(
				t,
				candidate,
				snapshotExternalDigest(candidate),
			)
		})
	}
}

func TestRestoreResultSnapshotRejectsRehashedInvalidStructure(t *testing.T) {
	t.Parallel()

	encoded, _, err := MarshalResultSnapshot(richSnapshotResult(t))
	if err != nil {
		t.Fatal(err)
	}
	valid := decodeResultSnapshotDocument(t, encoded)
	if len(valid.Issues) < 2 ||
		len(valid.Questions) < 2 ||
		len(valid.Questions[0].DerivedFrom) < 2 ||
		len(valid.Questions[0].Options) < 2 ||
		len(valid.DefaultProposals) < 2 ||
		len(valid.PackRefs) < 2 {
		t.Fatal("test fixture lacks required structure")
	}

	tests := []struct {
		name   string
		mutate func(*resultSnapshotDocument)
	}{
		{
			name: "snapshot schema",
			mutate: func(value *resultSnapshotDocument) {
				value.Schema = "orquesta.wizard.gaps.result-snapshot.v2"
			},
		},
		{
			name: "result schema",
			mutate: func(value *resultSnapshotDocument) {
				value.ResultSchema = "orquesta.wizard.gaps.v2"
			},
		},
		{
			name: "issue ref syntax",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0].Ref = "issue:invalid"
			},
		},
		{
			name: "well-formed unknown issue ref",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0].Ref = "intake-issue:wizard.unknown"
			},
		},
		{
			name: "unknown issue kind",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0].Kind = "changed"
			},
		},
		{
			name: "unknown dimension",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0].Dimension = "U99"
			},
		},
		{
			name: "issue structure",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0].Field = "product.changed"
			},
		},
		{
			name: "duplicate issue",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues = append(value.Issues, value.Issues[0])
			},
		},
		{
			name: "omitted issue",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues = append(
					value.Issues[:0],
					value.Issues[1:]...,
				)
			},
		},
		{
			name: "swapped issues",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0], value.Issues[1] =
					value.Issues[1], value.Issues[0]
			},
		},
		{
			name: "issue cardinality",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues = make(
					[]resultSnapshotIssue,
					maxResultSnapshotIssues+1,
				)
			},
		},
		{
			name: "null issue collection",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues = nil
			},
		},
		{
			name: "null issue dependencies",
			mutate: func(value *resultSnapshotDocument) {
				value.Issues[0].DependsOn = nil
			},
		},
		{
			name: "unknown question ref",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].Ref = "intake-question:wizard.unknown"
			},
		},
		{
			name: "duplicate question",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions = append(value.Questions, value.Questions[0])
			},
		},
		{
			name: "omitted question",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions = append(
					value.Questions[:0],
					value.Questions[1:]...,
				)
			},
		},
		{
			name: "swapped questions",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0], value.Questions[1] =
					value.Questions[1], value.Questions[0]
			},
		},
		{
			name: "question cardinality",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions = make(
					[]resultSnapshotQuestion,
					maxResultSnapshotQuestions+1,
				)
			},
		},
		{
			name: "missing derived issue",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].DerivedFrom[0] =
					"intake-issue:wizard.missing"
			},
		},
		{
			name: "duplicate causal ref",
			mutate: func(value *resultSnapshotDocument) {
				ref := value.Questions[0].DerivedFrom[0]
				value.Questions[0].DerivedFrom =
					append(value.Questions[0].DerivedFrom, ref)
			},
		},
		{
			name: "swapped causal refs",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].DerivedFrom[0],
					value.Questions[0].DerivedFrom[1] =
					value.Questions[0].DerivedFrom[1],
					value.Questions[0].DerivedFrom[0]
			},
		},
		{
			name: "wrong causal owner",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].DerivedFrom[0],
					value.Questions[1].DerivedFrom[0] =
					value.Questions[1].DerivedFrom[0],
					value.Questions[0].DerivedFrom[0]
			},
		},
		{
			name: "causal cardinality",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].DerivedFrom = make(
					[]IssueRef,
					maxResultSnapshotCausalRefs+1,
				)
			},
		},
		{
			name: "question dependency",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].DependsOn =
					[]QuestionRef{"intake-question:wizard.u20"}
			},
		},
		{
			name: "null question dependencies",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].DependsOn = nil
			},
		},
		{
			name: "no recommendation",
			mutate: func(value *resultSnapshotDocument) {
				for index := range value.Questions[0].Options {
					value.Questions[0].Options[index].Recommended = false
				}
			},
		},
		{
			name: "recommended free text",
			mutate: func(value *resultSnapshotDocument) {
				for index := range value.Questions[0].Options {
					value.Questions[0].Options[index].Recommended =
						value.Questions[0].Options[index].Kind == OptionFreeText
				}
			},
		},
		{
			name: "recommendation outside evaluator outcomes",
			mutate: func(value *resultSnapshotDocument) {
				for index := range value.Questions[0].Options {
					value.Questions[0].Options[index].Recommended =
						value.Questions[0].Options[index].Ref ==
							optionRef(DimensionU1, "public")
				}
			},
		},
		{
			name: "duplicate option ref",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].Options[1].Ref =
					value.Questions[0].Options[0].Ref
			},
		},
		{
			name: "option cardinality",
			mutate: func(value *resultSnapshotDocument) {
				value.Questions[0].Options = make(
					[]resultSnapshotOption,
					maxResultSnapshotOptions+1,
				)
			},
		},
		{
			name: "unknown pack",
			mutate: func(value *resultSnapshotDocument) {
				value.PackRefs[0] = "pack:missing"
			},
		},
		{
			name: "non-canonical pack order",
			mutate: func(value *resultSnapshotDocument) {
				value.PackRefs[0], value.PackRefs[1] =
					value.PackRefs[1], value.PackRefs[0]
			},
		},
		{
			name: "duplicate pack",
			mutate: func(value *resultSnapshotDocument) {
				value.PackRefs[1] = value.PackRefs[0]
			},
		},
		{
			name: "pack cardinality",
			mutate: func(value *resultSnapshotDocument) {
				value.PackRefs = make([]string, maxResultSnapshotPackRefs+1)
			},
		},
		{
			name: "duplicate default dimension",
			mutate: func(value *resultSnapshotDocument) {
				value.DefaultProposals[1] = value.DefaultProposals[0]
			},
		},
		{
			name: "swapped defaults",
			mutate: func(value *resultSnapshotDocument) {
				value.DefaultProposals[0], value.DefaultProposals[1] =
					value.DefaultProposals[1], value.DefaultProposals[0]
			},
		},
		{
			name: "default option outside evaluator outcomes",
			mutate: func(value *resultSnapshotDocument) {
				proposal := &value.DefaultProposals[0]
				dimension := snapshotDimension(t, proposal.Dimension)
				for _, option := range dimension.Options() {
					if option.Ref() == dimension.DefaultOption() ||
						option.Kind() == OptionFreeText {
						continue
					}
					proposal.Option = option.Ref()
					proposal.RationaleKey = option.RationaleKey()
					return
				}
				panic("test default dimension lacks an alternate preset")
			},
		},
		{
			name: "default also has question",
			mutate: func(value *resultSnapshotDocument) {
				insertSnapshotDefault(
					value,
					snapshotDefaultProposal(t, DimensionT5),
				)
			},
		},
		{
			name: "default cardinality",
			mutate: func(value *resultSnapshotDocument) {
				value.DefaultProposals = make(
					[]resultSnapshotDefaultProposal,
					maxResultSnapshotDefaults+1,
				)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := decodeResultSnapshotDocument(t, encoded)
			test.mutate(&candidate)
			sealed, sealErr := marshalResultSnapshotDocument(candidate)
			if sealErr != nil {
				t.Fatal(sealErr)
			}
			assertInvalidResultSnapshot(
				t,
				sealed,
				snapshotExternalDigest(sealed),
			)
		})
	}
}

func TestMarshalResultSnapshotRejectsInvalidResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Result)
	}{
		{
			name: "result schema",
			mutate: func(value *Result) {
				value.schemaVersion = "orquesta.wizard.gaps.v2"
			},
		},
		{
			name: "pack order",
			mutate: func(value *Result) {
				value.packRefs[0], value.packRefs[1] =
					value.packRefs[1], value.packRefs[0]
			},
		},
		{
			name: "issue ref",
			mutate: func(value *Result) {
				value.issues[0].ref = "intake-issue:wizard.changed"
			},
		},
		{
			name: "issue order",
			mutate: func(value *Result) {
				value.issues[0], value.issues[1] =
					value.issues[1], value.issues[0]
			},
		},
		{
			name: "question order",
			mutate: func(value *Result) {
				value.questions[0], value.questions[1] =
					value.questions[1], value.questions[0]
			},
		},
		{
			name: "question cause",
			mutate: func(value *Result) {
				value.questions[0].derivedFrom[0] =
					value.questions[1].derivedFrom[0]
			},
		},
		{
			name: "causal order",
			mutate: func(value *Result) {
				value.questions[0].derivedFrom[0],
					value.questions[0].derivedFrom[1] =
					value.questions[0].derivedFrom[1],
					value.questions[0].derivedFrom[0]
			},
		},
		{
			name: "default order",
			mutate: func(value *Result) {
				value.defaults[0], value.defaults[1] =
					value.defaults[1], value.defaults[0]
			},
		},
		{
			name: "recommendation outside evaluator outcomes",
			mutate: func(value *Result) {
				for index := range value.questions[0].options {
					value.questions[0].options[index].recommended =
						value.questions[0].options[index].ref ==
							optionRef(DimensionU1, "public")
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := richSnapshotResult(t)
			test.mutate(&result)
			if _, _, err := MarshalResultSnapshot(result); ErrorCodeOf(err) != ErrorInvalidResultSnapshot {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func richSnapshotResult(t *testing.T) Result {
	t.Helper()
	result, err := Evaluate(Input{
		Facts:    Facts{SharingIntent: SharingShared},
		PackRefs: catalog.DomainPackRefs(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func allDimensionSelections() []Selection {
	dimensions := BuiltInV1().Dimensions()
	result := make([]Selection, 0, len(dimensions))
	for _, dimension := range dimensions {
		var selected OptionRef
		for _, option := range dimension.Options() {
			if option.Recommended() {
				selected = option.Ref()
				break
			}
		}
		if selected == "" {
			panic("built-in dimension without recommendation")
		}
		result = append(result, Selection{
			Dimension: dimension.Ref(),
			Option:    selected,
		})
	}
	return result
}

func snapshotDimension(
	t *testing.T,
	ref DimensionRef,
) DimensionDescriptor {
	t.Helper()
	for _, dimension := range BuiltInV1().Dimensions() {
		if dimension.Ref() == ref {
			return dimension
		}
	}
	t.Fatalf("unknown test dimension %s", ref)
	return DimensionDescriptor{}
}

func snapshotDefaultProposal(
	t *testing.T,
	ref DimensionRef,
) resultSnapshotDefaultProposal {
	t.Helper()
	dimension := snapshotDimension(t, ref)
	option, found := findOption(dimension.options, dimension.defaultRef)
	if !found {
		t.Fatalf("dimension %s has no default", ref)
	}
	prefix := MessageKey("wizard.gaps.default." + lowerRef(ref))
	return resultSnapshotDefaultProposal{
		Dimension: ref, DecisionKind: DecisionTechnicalDefault,
		Option: dimension.defaultRef, LabelKey: prefix + ".label",
		HelpKey: prefix + ".help", ExampleKey: prefix + ".example",
		RationaleKey: option.rationaleKey,
	}
}

func insertSnapshotDefault(
	document *resultSnapshotDocument,
	proposal resultSnapshotDefaultProposal,
) {
	order := make(map[DimensionRef]int)
	for index, dimension := range BuiltInV1().Dimensions() {
		order[dimension.Ref()] = index
	}
	at := 0
	for at < len(document.DefaultProposals) &&
		order[document.DefaultProposals[at].Dimension] < order[proposal.Dimension] {
		at++
	}
	document.DefaultProposals = append(
		document.DefaultProposals,
		resultSnapshotDefaultProposal{},
	)
	copy(
		document.DefaultProposals[at+1:],
		document.DefaultProposals[at:],
	)
	document.DefaultProposals[at] = proposal
}

func decodeResultSnapshotDocument(
	t *testing.T,
	encoded []byte,
) resultSnapshotDocument {
	t.Helper()
	var document resultSnapshotDocument
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func assertInvalidResultSnapshot(
	t *testing.T,
	encoded []byte,
	expectedDigest string,
) {
	t.Helper()
	if _, err := RestoreResultSnapshot(
		encoded,
		expectedDigest,
	); ErrorCodeOf(err) != ErrorInvalidResultSnapshot {
		t.Fatalf("error=%v", err)
	}
}

func snapshotExternalDigest(encoded []byte) string {
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
