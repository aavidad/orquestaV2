package orquestaweb

type WebNuevaAppIntakeQuestionV0 struct {
	Field     string `json:"field"`
	SectionID string `json:"section_id"`
	LabelKey  string `json:"label_key"`
	HelpKey   string `json:"help_key,omitempty"`
	Required  bool   `json:"required"`
}

func webNuevaAppIntakeQuestionsV0(fields []string) []WebNuevaAppIntakeQuestionV0 {
	index := webNuevaAppIntakeFieldIndexByFieldV0()
	out := make([]WebNuevaAppIntakeQuestionV0, 0, len(fields))
	for _, field := range compactStringsV0(fields) {
		out = append(out, WebNuevaAppIntakeQuestionV0{
			Field:     field,
			SectionID: index[field],
			LabelKey:  "nueva_app.campo." + field,
			HelpKey:   webNuevaAppIntakeHelpKeyV0(field),
			Required:  webNuevaAppIntakeFieldRequiredV0(field),
		})
	}
	if out == nil {
		return []WebNuevaAppIntakeQuestionV0{}
	}
	return out
}

func webNuevaAppIntakeFieldIndexByFieldV0() map[string]string {
	out := map[string]string{}
	for _, item := range webNuevaAppIntakeFieldIndexV0() {
		out[item.Field] = item.SectionID
	}
	return out
}

func webNuevaAppIntakeHelpKeyV0(field string) string {
	switch field {
	case "request_kind":
		return "nueva_app.ayuda.request_kind"
	case "execution_mode":
		return "nueva_app.ayuda.execution_mode"
	default:
		return ""
	}
}
