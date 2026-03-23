package db

type schemaBackendSpec struct {
<<<<<<< HEAD
	name              string
	supportsBootstrap bool
	renderDDL         func() string
	renderSeedSQL     func() string
}

type bootstrapPlan struct {
	DDL  string
	Seed string
=======
	name                   string
	supportsBootstrap      bool
	renderDDLParts         func() []string
	renderSeedSQL          func() string
	postMigrationStatements func() []string
>>>>>>> origin/orq-orquestador-codex2
}

func schemaBackendSpecForDriver(driver string) (schemaBackendSpec, bool) {
	switch normalizedDriverName(driver) {
	case "sqlite":
		return schemaBackendSpec{
			name:              "sqlite",
			supportsBootstrap: true,
<<<<<<< HEAD
			renderDDL: func() string {
				return schemaDDLForDriver("sqlite")
=======
			renderDDLParts: func() []string {
				return []string{renderSQLiteBaseDDL(), renderSQLiteAuxDDL()}
>>>>>>> origin/orq-orquestador-codex2
			},
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("sqlite")
			},
<<<<<<< HEAD
=======
			postMigrationStatements: postMigrationStatementsSQLite,
		}, true
	case "postgres":
		return schemaBackendSpec{
			name:              "postgres",
			supportsBootstrap: true,
			renderDDLParts: func() []string {
				return []string{renderPostgresBaseDDL(), renderPostgresAuxDDL()}
			},
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("postgres")
			},
			postMigrationStatements: func() []string { return nil },
>>>>>>> origin/orq-orquestador-codex2
		}, true
	case "mysql":
		return schemaBackendSpec{
			name:              "mysql",
			supportsBootstrap: false,
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("mysql")
			},
<<<<<<< HEAD
		}, true
	case "postgres":
		return schemaBackendSpec{
			name:              "postgres",
			supportsBootstrap: false,
=======
			postMigrationStatements: func() []string { return nil },
>>>>>>> origin/orq-orquestador-codex2
		}, true
	default:
		return schemaBackendSpec{}, false
	}
}

func fallbackSchemaBackendSpec(driver string) schemaBackendSpec {
	if spec, ok := schemaBackendSpecForDriver(driver); ok {
		return spec
	}
	spec, _ := schemaBackendSpecForDriver("sqlite")
	return spec
}

<<<<<<< HEAD
func (spec schemaBackendSpec) ddl() string {
	if spec.renderDDL == nil {
		return ""
	}
	return spec.renderDDL()
=======
func (spec schemaBackendSpec) ddlParts() []string {
	if spec.renderDDLParts == nil {
		return nil
	}
	return spec.renderDDLParts()
>>>>>>> origin/orq-orquestador-codex2
}

func (spec schemaBackendSpec) seedSQL() string {
	if spec.renderSeedSQL == nil {
		return ""
	}
	return spec.renderSeedSQL()
}

func (spec schemaBackendSpec) bootstrapPlan() (bootstrapPlan, bool) {
<<<<<<< HEAD
	if !spec.supportsBootstrap {
		return bootstrapPlan{}, false
	}
	return bootstrapPlan{
		DDL:  spec.ddl(),
=======
	if !spec.supportsBootstrap || spec.renderDDLParts == nil {
		return bootstrapPlan{}, false
	}
	return bootstrapPlan{
		DDL:  joinDDLParts(spec.ddlParts()...),
>>>>>>> origin/orq-orquestador-codex2
		Seed: spec.seedSQL(),
	}, true
}

<<<<<<< HEAD
func bootstrapPlanForDriver(driver string) (bootstrapPlan, bool) {
	spec, ok := schemaBackendSpecForDriver(driver)
	if !ok {
		return bootstrapPlan{}, false
	}
	return spec.bootstrapPlan()
=======
func (spec schemaBackendSpec) postMigrations() []string {
	if spec.postMigrationStatements == nil {
		return nil
	}
	return spec.postMigrationStatements()
>>>>>>> origin/orq-orquestador-codex2
}
