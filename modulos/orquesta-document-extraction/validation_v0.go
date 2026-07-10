package orquestadocumentextraction

import (
	"fmt"
	"strings"
)

func DefaultDocumentExtractionPolicyV0() DocumentExtractionPolicyV0 {
	return DocumentExtractionPolicyV0{DataHandlingMode: DocumentDataHandlingLocalV0}
}

func NewPersonDocumentSchemaV0(schema DocumentSchemaV0) (PersonDocumentSchemaV0, error) {
	if schema.EntityKind != "person" {
		return PersonDocumentSchemaV0{}, fmt.Errorf("document_schema_entity_kind_invalid")
	}
	if err := ValidateDocumentSchemaV0(schema); err != nil {
		return PersonDocumentSchemaV0{}, err
	}
	return PersonDocumentSchemaV0{Schema: schema}, nil
}

func NewInvoiceDocumentSchemaV0(schema DocumentSchemaV0) (InvoiceDocumentSchemaV0, error) {
	if schema.EntityKind != "invoice" {
		return InvoiceDocumentSchemaV0{}, fmt.Errorf("document_schema_entity_kind_invalid")
	}
	if err := ValidateDocumentSchemaV0(schema); err != nil {
		return InvoiceDocumentSchemaV0{}, err
	}
	return InvoiceDocumentSchemaV0{Schema: schema}, nil
}

func ValidateDocumentSchemaV0(schema DocumentSchemaV0) error {
	if blankDocumentValueV0(schema.SchemaRef) || blankDocumentValueV0(schema.DomainRef) || blankDocumentValueV0(schema.EntityKind) || blankDocumentValueV0(schema.SchemaVersion) {
		return fmt.Errorf("document_schema_ref_required")
	}
	if len(schema.Fields) == 0 {
		return fmt.Errorf("document_schema_fields_required")
	}
	seen := map[string]struct{}{}
	for _, field := range schema.Fields {
		if blankDocumentValueV0(field.FieldRef) || !validDocumentFieldDataTypeV0(field.DataType) {
			return fmt.Errorf("document_schema_field_invalid")
		}
		if _, duplicate := seen[field.FieldRef]; duplicate {
			return fmt.Errorf("document_schema_field_duplicate")
		}
		seen[field.FieldRef] = struct{}{}
	}
	return nil
}

func ValidateDocumentExtractionPolicyV0(policy DocumentExtractionPolicyV0) error {
	if policy.DataHandlingMode == "" {
		policy.DataHandlingMode = DocumentDataHandlingLocalV0
	}
	switch policy.DataHandlingMode {
	case DocumentDataHandlingLocalV0:
		return nil
	case DocumentDataHandlingCloudV0:
		if !policy.CloudOptIn || blankDocumentValueV0(policy.PolicyRef) || blankDocumentValueV0(policy.RegionRef) || blankDocumentValueV0(policy.RetentionRef) || blankDocumentValueV0(policy.DeletionRef) || blankDocumentValueV0(policy.EncryptionRef) || blankDocumentValueV0(policy.AuditRef) {
			return fmt.Errorf("document_cloud_policy_incomplete")
		}
		return nil
	default:
		return fmt.Errorf("document_data_handling_mode_invalid")
	}
}

func ValidateDocumentConnectorRegistryV0(registry DocumentConnectorRegistryV0) error {
	if blankDocumentValueV0(registry.RegistryRef) {
		return fmt.Errorf("document_connector_registry_ref_required")
	}
	seen := map[string]struct{}{}
	for _, connector := range registry.Connectors {
		if blankDocumentValueV0(connector.ConnectorRef) || blankDocumentValueV0(connector.ConfigurationRef) || blankDocumentValueV0(connector.AdapterIdentity.AdapterRef) || blankDocumentValueV0(connector.AdapterIdentity.Version) || len(connector.CapabilityRefs) == 0 {
			return fmt.Errorf("document_connector_invalid")
		}
		if connector.Mode != DocumentDataHandlingLocalV0 && connector.Mode != DocumentDataHandlingCloudV0 {
			return fmt.Errorf("document_connector_mode_invalid")
		}
		if _, duplicate := seen[connector.ConnectorRef]; duplicate {
			return fmt.Errorf("document_connector_duplicate")
		}
		seen[connector.ConnectorRef] = struct{}{}
	}
	return nil
}

func validateDocumentExtractionRequestV0(request DocumentExtractionRequestV0) error {
	if blankDocumentValueV0(request.DocumentRef) || blankDocumentValueV0(request.ConfigurationRef) || blankDocumentValueV0(request.ConfigurationHash) || blankDocumentValueV0(request.NormalizationLocale) || blankDocumentValueV0(request.OutputLocale) {
		return fmt.Errorf("document_extraction_request_invalid")
	}
	if err := ValidateDocumentSchemaV0(request.Schema); err != nil {
		return err
	}
	return ValidateDocumentExtractionPolicyV0(request.Policy)
}

func validateDocumentExtractionPortsV0(ports DocumentExtractionPortsV0) error {
	if ports.Source == nil || ports.Normalizer == nil || ports.Parser == nil || ports.Localizer == nil || ports.SchemaExtractor == nil || ports.EvidenceLocator == nil || ports.Validator == nil || ports.ReceiptStore == nil {
		return fmt.Errorf("document_extraction_port_required")
	}
	return nil
}

func validDocumentFieldDataTypeV0(value DocumentFieldDataTypeV0) bool {
	switch value {
	case DocumentFieldDataTypeStringV0, DocumentFieldDataTypeNumberV0, DocumentFieldDataTypeBooleanV0, DocumentFieldDataTypeDateV0, DocumentFieldDataTypeCurrencyV0, DocumentFieldDataTypeObjectV0, DocumentFieldDataTypeArrayV0:
		return true
	default:
		return false
	}
}

func blankDocumentValueV0(value string) bool { return strings.TrimSpace(value) == "" }
