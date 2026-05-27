package orquestaappchangedirectorsource

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func appChangeQuestionRefV0(changeRef string) string {
	return "question-ref-app-change-" + strings.TrimSpace(changeRef)
}

func AppChangeTaskRefV0(changeRef string) string {
	return appChangeRefsV0(changeRef).TaskRef
}

func appChangeSuffixV0(changeRef string) string {
	sum := sha256.Sum256(appChangeCanonicalRefPayloadV0("change_ref", changeRef))
	return "appchange-" + hex.EncodeToString(sum[:])[:32]
}

func appChangeCanonicalRefPayloadV0(fields ...string) []byte {
	var builder strings.Builder
	builder.WriteString("orquesta-app-change-director-source.ref.v0|")
	builder.WriteString(strconv.Itoa(len(fields)))
	builder.WriteByte('|')
	for _, field := range fields {
		value := strings.TrimSpace(field)
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
		builder.WriteByte(';')
	}
	return []byte(builder.String())
}

func appChangeRefsV0(changeRef string) appChangeRefSetV0 {
	suffix := appChangeSuffixV0(changeRef)
	return appChangeRefSetV0{
		Suffix:      suffix,
		QuestionRef: appChangeQuestionRefV0(changeRef),
		EvidenceRef: "evidence-ref-app-change-" + suffix,
		AnswerRef:   "answer-ref-app-change-" + suffix,
		VoteRef:     "vote-ref-app-change-" + suffix,
		DecisionRef: "decision-ref-app-change-" + suffix,
		OptionRef:   "option-ref-app-change-" + suffix,
		TopicRef:    "topic-ref-app-change-" + suffix,
		BrainRef:    "brainstorm-ref-app-change-" + suffix,
		ContractRef: "contract:function:app-change:" + suffix + ":v0",
		TaskRef:     "task-ref-app-change-" + suffix,
	}
}

type appChangeRefSetV0 struct {
	Suffix      string
	QuestionRef string
	EvidenceRef string
	AnswerRef   string
	VoteRef     string
	DecisionRef string
	OptionRef   string
	TopicRef    string
	BrainRef    string
	ContractRef string
	TaskRef     string
}

func appChangeCommandRefV0(action string, refs appChangeRefSetV0) string {
	return "command-ref-app-change-" + action + "-" + refs.Suffix
}

func appChangeDecisionRefV0(action string, refs appChangeRefSetV0) string {
	return "director-decision-app-change-" + action + "-" + refs.Suffix
}
