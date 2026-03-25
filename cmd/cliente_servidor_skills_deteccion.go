package cmd

func detectarCarenciaSkillPorAPI(req apiSkillDeteccionRequest) (*apiSkillDeteccionResponse, bool, error) {
	var resp apiSkillDeteccionResponse
	ok, err := apiPost("/api/skills/detectar-carencia", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}
