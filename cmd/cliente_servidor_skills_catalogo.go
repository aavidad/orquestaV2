package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"orquesta/skillsapp"
)

func listarSkillsRemotasPorAPI(filtro string, limite int) ([]*skillsapp.SkillRemota, bool, error) {
	query := url.Values{}
	if strings.TrimSpace(filtro) != "" {
		query.Set("q", strings.TrimSpace(filtro))
	}
	if limite > 0 {
		query.Set("limit", fmt.Sprintf("%d", limite))
	}
	var resp apiSkillsRemotasResponse
	ok, err := apiGetQuery("/api/skills/remotas", query, &resp)
	if err != nil || !ok {
		return nil, ok, err
	}
	return resp.Items, true, nil
}

func borrarSkillPorAPI(id int64, actor string) (bool, error) {
	return apiPost(fmt.Sprintf("/api/skills/%d/borrar", id), apiSkillBorrarRequest{Actor: actor}, nil)
}
