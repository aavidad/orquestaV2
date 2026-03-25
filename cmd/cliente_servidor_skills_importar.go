package cmd

func importarSkillPorAPI(req apiSkillImportarRequest) (*apiSkillImportarResponse, bool, error) {
	var resp apiSkillImportarResponse
	ok, err := apiPost("/api/skills/importar", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}
