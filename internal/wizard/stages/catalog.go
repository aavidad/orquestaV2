package stages

import "reflect"

type CatalogInput struct {
	Version   CatalogVersion
	Templates []Template
}

// Catalog is an immutable, versioned registry. It contains exactly the six
// canonical V23 plan templates. A later application compiler may consume this
// data, but this package never creates or mutates a Goal.
type Catalog struct {
	version   CatalogVersion
	templates []Template
}

func NewCatalog(input CatalogInput) (Catalog, error) {
	if input.Version.value == "" {
		return Catalog{}, domainError(ErrorInvalidArgument, "catalog.version")
	}
	templates := cloneTemplates(input.Templates)
	if len(templates) != len(requiredTemplateValues) {
		return Catalog{}, domainError(ErrorIncompleteCatalog, "catalog.templates")
	}
	byRef := make(map[string]Template, len(templates))
	for _, template := range templates {
		if err := validateTemplate(template); err != nil {
			return Catalog{}, err
		}
		if previous, exists := byRef[template.ref.value]; exists {
			if !sameTemplate(previous, template) {
				return Catalog{}, domainError(
					ErrorConflictingDefinition,
					"catalog.templates",
				)
			}
			return Catalog{}, domainError(ErrorDuplicateRef, "catalog.templates")
		}
		byRef[template.ref.value] = template
	}
	for _, required := range requiredTemplateValues {
		if _, exists := byRef[required]; !exists {
			return Catalog{}, domainError(ErrorIncompleteCatalog, "catalog.templates")
		}
	}
	sortTemplates(templates)
	return Catalog{version: input.Version, templates: templates}, nil
}

func (value Catalog) Version() CatalogVersion {
	return value.version
}

func (value Catalog) Templates() []Template {
	return cloneTemplates(value.templates)
}

func (value Catalog) Template(ref TemplateRef) (Template, bool) {
	for _, template := range value.templates {
		if template.ref == ref {
			return cloneTemplates([]Template{template})[0], true
		}
	}
	return Template{}, false
}

func sameTemplate(left, right Template) bool {
	return reflect.DeepEqual(left, right)
}
