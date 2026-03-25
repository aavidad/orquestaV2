package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type SolicitudDeteccionSkill struct {
	TipoAgente       string
	Nombre           string
	Descripcion      string
	CuandoUsar       string
	Escenario        string
	AliasesJSON      string
	HerramientasJSON string
}

type CoincidenciaSkill struct {
	Skill      *Skill   `json:"skill"`
	Puntuacion int      `json:"puntuacion"`
	Motivos    []string `json:"motivos"`
}

type ResultadoDeteccionSkill struct {
	ConsultoCatalogo  bool                 `json:"consulto_catalogo"`
	Falta             bool                 `json:"falta"`
	Motivo            string               `json:"motivo"`
	Equivalente       *Skill               `json:"equivalente,omitempty"`
	Candidatas        []*CoincidenciaSkill `json:"candidatas,omitempty"`
	Borrador          *Skill               `json:"borrador,omitempty"`
	RecursosSugeridos []string             `json:"recursos_sugeridos,omitempty"`
	Validaciones      []string             `json:"validaciones,omitempty"`
	InvocacionCreador string               `json:"invocacion_creador,omitempty"`
}

func DetectarCarenciaSkill(req *SolicitudDeteccionSkill) (*ResultadoDeteccionSkill, error) {
	borrador, err := construirBorradorSkill(req)
	if err != nil {
		return nil, err
	}
	equivalente, err := BuscarSkillEquivalente(borrador, 0)
	if err != nil {
		return nil, err
	}
	resultado := &ResultadoDeteccionSkill{
		ConsultoCatalogo:  true,
		Borrador:          borrador,
		RecursosSugeridos: recursosSugeridosSkill(req),
		Validaciones:      validacionesSkill(req),
	}
	if equivalente != nil {
		resultado.Falta = false
		resultado.Equivalente = equivalente
		resultado.Motivo = fmt.Sprintf("ya existe una skill equivalente: #%d %s", equivalente.ID, equivalente.Nombre)
		return resultado, nil
	}

	activa := true
	existentes, err := ListarSkills(borrador.TipoAgente, &activa)
	if err != nil {
		return nil, err
	}
	resultado.Candidatas = rankearSkillsCandidatas(borrador, existentes, 3)
	resultado.Falta = true
	if len(resultado.Candidatas) == 0 {
		resultado.Motivo = "no existe skill equivalente ni candidata clara en el catalogo"
	} else {
		resultado.Motivo = "no existe skill equivalente; hay skills cercanas para revisar antes de crear una nueva"
	}
	resultado.InvocacionCreador = construirInvocacionSkillCreator(resultado)
	return resultado, nil
}

func construirBorradorSkill(req *SolicitudDeteccionSkill) (*Skill, error) {
	if req == nil {
		return nil, fmt.Errorf("solicitud nula")
	}
	borrador := &Skill{
		TipoAgente:       strings.TrimSpace(req.TipoAgente),
		Nombre:           nombreTentativoSkill(req),
		Descripcion:      strings.TrimSpace(req.Descripcion),
		CuandoUsar:       strings.TrimSpace(req.CuandoUsar),
		Escenario:        strings.TrimSpace(req.Escenario),
		AliasesJSON:      strings.TrimSpace(req.AliasesJSON),
		HerramientasJSON: strings.TrimSpace(req.HerramientasJSON),
		Origen:           skillOriginLocal,
		Activa:           false,
	}
	if err := normalizarSkillParaCatalogo(borrador); err != nil {
		return nil, err
	}
	return borrador, nil
}

func nombreTentativoSkill(req *SolicitudDeteccionSkill) string {
	if req == nil {
		return ""
	}
	if nombre := strings.TrimSpace(req.Nombre); nombre != "" {
		return nombre
	}
	if escenario := strings.TrimSpace(req.Escenario); escenario != "" {
		return "skill-" + escenario
	}
	if herramientas := parseListaJSON(req.HerramientasJSON); len(herramientas) > 0 {
		return "skill-" + herramientas[0]
	}
	return "skill-nueva"
}

func rankearSkillsCandidatas(objetivo *Skill, existentes []*Skill, limite int) []*CoincidenciaSkill {
	coincidencias := make([]*CoincidenciaSkill, 0, len(existentes))
	for _, item := range existentes {
		if item == nil {
			continue
		}
		puntuacion, motivos := puntuarCoincidenciaSkill(objetivo, item)
		if puntuacion <= 0 {
			continue
		}
		coincidencias = append(coincidencias, &CoincidenciaSkill{
			Skill:      item,
			Puntuacion: puntuacion,
			Motivos:    motivos,
		})
	}
	sort.SliceStable(coincidencias, func(i, j int) bool {
		if coincidencias[i].Puntuacion == coincidencias[j].Puntuacion {
			if coincidencias[i].Skill.Prioridad == coincidencias[j].Skill.Prioridad {
				return coincidencias[i].Skill.Nombre < coincidencias[j].Skill.Nombre
			}
			return coincidencias[i].Skill.Prioridad < coincidencias[j].Skill.Prioridad
		}
		return coincidencias[i].Puntuacion > coincidencias[j].Puntuacion
	})
	if limite > 0 && len(coincidencias) > limite {
		coincidencias = coincidencias[:limite]
	}
	return coincidencias
}

func puntuarCoincidenciaSkill(objetivo, candidata *Skill) (int, []string) {
	if objetivo == nil || candidata == nil {
		return 0, nil
	}
	puntuacion := 0
	motivos := make([]string, 0, 4)
	if objetivo.Escenario != "" && strings.EqualFold(objetivo.Escenario, candidata.Escenario) {
		puntuacion += 4
		motivos = append(motivos, "mismo escenario")
	}
	overlapHerramientas := interseccionListas(parseListaJSON(objetivo.HerramientasJSON), parseListaJSON(candidata.HerramientasJSON))
	if overlapHerramientas > 0 {
		puntuacion += overlapHerramientas * 3
		motivos = append(motivos, "herramientas relacionadas")
	}
	overlapTexto := solapamientoTextoSkill(objetivo, candidata)
	if overlapTexto > 0 {
		puntuacion += overlapTexto
		motivos = append(motivos, "uso parecido")
	}
	return puntuacion, motivos
}

func interseccionListas(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(a))
	for _, item := range a {
		seen[item] = struct{}{}
	}
	total := 0
	for _, item := range b {
		if _, ok := seen[item]; ok {
			total++
		}
	}
	return total
}

func solapamientoTextoSkill(a, b *Skill) int {
	tokensA := tokensSkill(a)
	tokensB := tokensSkill(b)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(tokensA))
	for _, token := range tokensA {
		seen[token] = struct{}{}
	}
	total := 0
	for _, token := range tokensB {
		if _, ok := seen[token]; ok {
			total++
		}
	}
	return total
}

func tokensSkill(s *Skill) []string {
	if s == nil {
		return nil
	}
	parts := strings.Fields(strings.ToLower(strings.Join([]string{
		s.Nombre,
		s.Descripcion,
		s.CuandoUsar,
		s.Escenario,
		strings.Join(parseListaJSON(s.AliasesJSON), " "),
		strings.Join(parseListaJSON(s.HerramientasJSON), " "),
	}, " ")))
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, token := range parts {
		token = strings.Trim(token, " ,.;:()[]{}")
		if len(token) < 3 {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func recursosSugeridosSkill(req *SolicitudDeteccionSkill) []string {
	recursos := []string{"references"}
	if req == nil {
		return recursos
	}
	if len(parseListaJSON(req.HerramientasJSON)) > 0 {
		recursos = append(recursos, "scripts")
	}
	texto := strings.ToLower(strings.Join([]string{req.Nombre, req.Descripcion, req.CuandoUsar, req.Escenario}, " "))
	if strings.Contains(texto, "plantilla") || strings.Contains(texto, "ui") || strings.Contains(texto, "icon") || strings.Contains(texto, "asset") {
		recursos = append(recursos, "assets")
	}
	return recursos
}

func validacionesSkill(req *SolicitudDeteccionSkill) []string {
	return []string{
		"consultar_catalogo_compartido_antes_de_crear",
		"rechazar_duplicados_equivalentes",
		"definir_metadata_minima",
		"proponer_recursos_por_capas",
		"dejar_borrador_revisable_antes_de_alta",
	}
}

func construirInvocacionSkillCreator(resultado *ResultadoDeteccionSkill) string {
	if resultado == nil || resultado.Borrador == nil || !resultado.Falta {
		return ""
	}
	candidatas := make([]string, 0, len(resultado.Candidatas))
	for _, item := range resultado.Candidatas {
		if item == nil || item.Skill == nil {
			continue
		}
		candidatas = append(candidatas, fmt.Sprintf("#%d %s", item.Skill.ID, item.Skill.Nombre))
	}
	candidatasTexto := "ninguna"
	if len(candidatas) > 0 {
		candidatasTexto = strings.Join(candidatas, ", ")
	}
	borradorJSON, _ := json.MarshalIndent(map[string]any{
		"tipo_agente":        resultado.Borrador.TipoAgente,
		"nombre":             resultado.Borrador.Nombre,
		"descripcion":        resultado.Borrador.Descripcion,
		"cuando_usar":        resultado.Borrador.CuandoUsar,
		"escenario":          resultado.Borrador.Escenario,
		"aliases_json":       resultado.Borrador.AliasesJSON,
		"herramientas_json":  resultado.Borrador.HerramientasJSON,
		"recursos_sugeridos": resultado.RecursosSugeridos,
		"validaciones":       resultado.Validaciones,
	}, "", "  ")
	return strings.TrimSpace(fmt.Sprintf(
		"Usa $skill-creator para preparar un borrador de skill nuevo sin mutar el catalogo. Requisitos: consulta previa ya hecha, candidatas revisadas (%s), respetar metadata minima de Orquesta y devolver un borrador revisable. Borrador base:\n%s",
		candidatasTexto,
		string(borradorJSON),
	))
}
