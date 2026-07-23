package acceptance_test

import "encoding/json"

type v20Fixture struct {
	SchemaVersion                  int                  `json:"schema_version"`
	ReceiptSchemaVersion           int                  `json:"receipt_schema_version"`
	ContractID                     string               `json:"contract_id"`
	TrustedBaseGitCommitOID        string               `json:"trusted_base_git_commit_oid"`
	ProductDeltaBaseGitCommitOID   string               `json:"product_delta_base_git_commit_oid"`
	ProductDeltaSealedGitCommitOID string               `json:"product_delta_sealed_git_commit_oid"`
	SealStatus                     string               `json:"seal_status"`
	SealNote                       string               `json:"seal_note"`
	Command                        string               `json:"command"`
	ExecutionArgv                  []string             `json:"execution_argv"`
	OutputPath                     string               `json:"output_path"`
	ReceiptPath                    string               `json:"receipt_path"`
	CandidateSubjects              []string             `json:"candidate_subjects"`
	ImplementationStatus           string               `json:"implementation_status"`
	LifecycleGate                  string               `json:"lifecycle_gate"`
	PreflightStatus                string               `json:"preflight_status"`
	PreflightNote                  string               `json:"preflight_note"`
	BlockingDependency             string               `json:"blocking_dependency_vertical"`
	PendingDependencyReceipt       string               `json:"pending_dependency_receipt"`
	DependencyActivation           v20DependencyGate    `json:"dependency_activation_contract"`
	OwnedCapabilityIDs             []string             `json:"owned_capability_ids"`
	DependencyVerticals            []string             `json:"dependency_verticals"`
	CanonicalRegistryPath          string               `json:"canonical_registry_path"`
	GeneratorPath                  string               `json:"generator_path"`
	ApplicationHandlerPath         string               `json:"application_handler_path"`
	CommandCatalogStatus           string               `json:"command_catalog_status"`
	RuntimeRegistryMode            string               `json:"runtime_registry_mode"`
	ConfigurationKeysAdded         []string             `json:"configuration_keys_added"`
	BundledLocales                 []string             `json:"bundled_locales"`
	I18NScope                      string               `json:"i18n_scope"`
	GeneratedFiles                 []string             `json:"generated_files"`
	RequiredDefinitionFields       []string             `json:"required_definition_fields"`
	RequiredApplicationTypes       []v20RequiredType    `json:"required_application_types"`
	RequiredDispatcherMethods      []string             `json:"required_dispatcher_methods"`
	StableErrorCodes               []string             `json:"stable_error_codes"`
	Commands                       []v20Command         `json:"commands"`
	RequiredBehaviorTests          []string             `json:"required_behavior_tests"`
	ForbiddenInterfaceImports      []string             `json:"forbidden_interface_imports"`
	ForbiddenInterfaceMarkers      []string             `json:"forbidden_interface_markers"`
	ForbiddenPrivateAuthorities    []string             `json:"forbidden_private_authorities"`
	MigrationContract              v20MigrationContract `json:"migration_contract"`
	SimplicityBudget               v20SimplicityBudget  `json:"simplicity_budget"`
	DeferredSurfaces               []string             `json:"deferred_surfaces"`
	PSEContract                    v20PSEContract       `json:"pse_contract"`
}

type v20PSEContract struct {
	P string `json:"P"`
	S string `json:"S"`
	E string `json:"E"`
}

type v20DependencyGate struct {
	Contract            string `json:"contract"`
	Result              string `json:"result"`
	SourceWorktreeState string `json:"source_worktree_state"`
	ProductOID          string `json:"product_oid"`
	SealedOID           string `json:"sealed_oid"`
	EvidenceOID         string `json:"evidence_oid"`
	IntegrationHeadOID  string `json:"integration_head_oid"`
}

type v20RequiredType struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

type v20Command struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Handler        string `json:"handler"`
	Permission     string `json:"permission"`
	Audience       string `json:"audience"`
	ExecutionBound bool   `json:"execution_bound"`
	ReplayMode     string `json:"replay_mode"`
}

type v20MigrationContract struct {
	FileGlob        string   `json:"file_glob"`
	RequiredMarkers []string `json:"required_markers"`
	CrashFrontiers  []string `json:"crash_frontiers"`
}

type v20SimplicityBudget struct {
	ManualProductLOC           int `json:"manual_product_loc"`
	CommandsApplicationLOC     int `json:"commands_application_loc"`
	BindingAdapterLOC          int `json:"binding_adapter_loc"`
	SDKLOC                     int `json:"sdk_loc"`
	SQLiteRecoveryBootstrapLOC int `json:"sqlite_recovery_bootstrap_loc"`
	GeneratedLOC               int `json:"generated_loc"`
	MigrationSQLLOC            int `json:"migration_sql_loc"`
	MaximumCommandDefinitions  int `json:"maximum_command_definitions"`
	MaximumLifecycleWriters    int `json:"maximum_lifecycle_writers"`
	MaximumCommandRegistries   int `json:"maximum_command_registries"`
}

type v20RegistryDocument struct {
	SchemaVersion int                     `json:"schema_version"`
	Revision      string                  `json:"revision"`
	Commands      []v20RegistryDefinition `json:"commands"`
}

type v20RegistryDefinition struct {
	ID             string          `json:"id"`
	Version        string          `json:"version"`
	Kind           string          `json:"kind"`
	Handler        string          `json:"handler"`
	Permission     string          `json:"permission"`
	Audience       string          `json:"audience"`
	ExecutionBound bool            `json:"execution_bound"`
	ReplayMode     string          `json:"replay_mode"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	DescriptionKey string          `json:"description_key"`
	ErrorCodes     []string        `json:"error_codes"`
	HTTP           v20HTTPBinding  `json:"http"`
	MCP            v20MCPBinding   `json:"mcp"`
	CLI            v20CLIBinding   `json:"cli"`
}

type v20HTTPBinding struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type v20MCPBinding struct {
	Tool string `json:"tool"`
}

type v20CLIBinding struct {
	Path []string `json:"path"`
}
