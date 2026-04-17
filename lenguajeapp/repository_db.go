package lenguajeapp

import "orquesta/db"

type Repository struct{}

func (Repository) GetLanguagePolicy() (*LanguagePolicy, error) {
	item, err := db.GetLanguagePolicy()
	if err != nil || item == nil {
		return nil, err
	}
	return mapLanguagePolicy(item), nil
}

func (Repository) SetLanguagePolicy(p *LanguagePolicy, updatedBy string) error {
	return db.SetLanguagePolicy(mapDBLanguagePolicy(p), updatedBy)
}

func (Repository) ListLanguageMatrixEntries() ([]*LanguageMatrixEntry, error) {
	items, err := db.ListLanguageMatrixEntries()
	if err != nil {
		return nil, err
	}
	out := make([]*LanguageMatrixEntry, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapLanguageMatrixEntry(item))
	}
	return out, nil
}

func (Repository) SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy string) (*LanguageMatrixEntry, error) {
	item, err := db.SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy)
	if err != nil || item == nil {
		return nil, err
	}
	return mapLanguageMatrixEntry(item), nil
}

func (Repository) DeleteLanguageMatrixEntry(kind, selector, contexto string) error {
	return db.DeleteLanguageMatrixEntry(kind, selector, contexto)
}

func (Repository) ResolveLanguage(project string, taskID *int64, contexto string) (*LanguageResolution, error) {
	item, err := db.ResolveLanguage(project, taskID, contexto)
	if err != nil || item == nil {
		return nil, err
	}
	return mapLanguageResolution(item), nil
}

func mapLanguagePolicy(in *db.LanguagePolicy) *LanguagePolicy {
	if in == nil {
		return nil
	}
	allowed := append([]string(nil), in.AllowedLanguages...)
	return &LanguagePolicy{
		DefaultLanguage:          in.DefaultLanguage,
		DocumentationMultilang:   in.DocumentationMultilang,
		AppsMultilang:            in.AppsMultilang,
		DocumentationDefaultLang: in.DocumentationDefaultLang,
		AppsDefaultLang:          in.AppsDefaultLang,
		AllowedLanguages:         allowed,
		Notes:                    in.Notes,
		UpdatedBy:                in.UpdatedBy,
		UpdatedAt:                in.UpdatedAt,
	}
}

func mapDBLanguagePolicy(in *LanguagePolicy) *db.LanguagePolicy {
	if in == nil {
		return nil
	}
	allowed := append([]string(nil), in.AllowedLanguages...)
	return &db.LanguagePolicy{
		DefaultLanguage:          in.DefaultLanguage,
		DocumentationMultilang:   in.DocumentationMultilang,
		AppsMultilang:            in.AppsMultilang,
		DocumentationDefaultLang: in.DocumentationDefaultLang,
		AppsDefaultLang:          in.AppsDefaultLang,
		AllowedLanguages:         allowed,
		Notes:                    in.Notes,
		UpdatedBy:                in.UpdatedBy,
		UpdatedAt:                in.UpdatedAt,
	}
}

func mapLanguageMatrixEntry(in *db.LanguageMatrixEntry) *LanguageMatrixEntry {
	if in == nil {
		return nil
	}
	return &LanguageMatrixEntry{
		Scope:     in.Scope,
		Selector:  in.Selector,
		Context:   in.Context,
		Language:  in.Language,
		Reason:    in.Reason,
		UpdatedBy: in.UpdatedBy,
		UpdatedAt: in.UpdatedAt,
		ConfigKey: in.ConfigKey,
	}
}

func mapLanguageResolution(in *db.LanguageResolution) *LanguageResolution {
	if in == nil {
		return nil
	}
	return &LanguageResolution{
		Proyecto: in.Proyecto,
		TareaID:  in.TareaID,
		Contexto: in.Contexto,
		Idioma:   in.Idioma,
		Origen:   in.Origen,
		Entrada:  mapLanguageMatrixEntry(in.Entrada),
		Politica: mapLanguagePolicy(in.Politica),
	}
}
