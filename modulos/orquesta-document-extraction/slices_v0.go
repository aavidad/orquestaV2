package orquestadocumentextraction

import (
	"fmt"
	"strings"
)

func BuildDocumentSchemaSlicesV0(
	document DocumentV0,
	schema DocumentSchemaV0,
	locations []DocumentLocationV0,
	documentLanguage string,
	normalizationLocale string,
	maxFieldsPerSlice int,
) ([]DocumentSchemaSliceV0, error) {
	if err := ValidateDocumentSchemaV0(schema); err != nil {
		return nil, err
	}
	if blankDocumentValueV0(document.DocumentRef) || blankDocumentValueV0(normalizationLocale) {
		return nil, fmt.Errorf("document_schema_slice_input_invalid")
	}
	if maxFieldsPerSlice <= 0 {
		maxFieldsPerSlice = len(schema.Fields)
	}
	if len(locations) == 0 {
		return nil, fmt.Errorf("document_location_required")
	}
	var slices []DocumentSchemaSliceV0
	for locationIndex, location := range locations {
		if blankDocumentValueV0(location.LocationRef) || blankDocumentValueV0(location.PageRef) {
			return nil, fmt.Errorf("document_location_invalid")
		}
		for fieldStart := 0; fieldStart < len(schema.Fields); fieldStart += maxFieldsPerSlice {
			fieldEnd := fieldStart + maxFieldsPerSlice
			if fieldEnd > len(schema.Fields) {
				fieldEnd = len(schema.Fields)
			}
			slices = append(slices, DocumentSchemaSliceV0{
				SliceRef:            fmt.Sprintf("%s:location:%d:fields:%d", strings.TrimSpace(schema.SchemaRef), locationIndex, fieldStart),
				SchemaRef:           schema.SchemaRef,
				DocumentRef:         document.DocumentRef,
				Location:            location,
				Fields:              append([]DocumentSchemaFieldV0(nil), schema.Fields[fieldStart:fieldEnd]...),
				DocumentLanguage:    strings.TrimSpace(documentLanguage),
				NormalizationLocale: strings.TrimSpace(normalizationLocale),
			})
		}
	}
	return slices, nil
}

func documentSchemaFieldByRefV0(schema DocumentSchemaV0, fieldRef string) (DocumentSchemaFieldV0, bool) {
	for _, field := range schema.Fields {
		if field.FieldRef == fieldRef {
			return field, true
		}
	}
	return DocumentSchemaFieldV0{}, false
}
