package db

type schemaBackendSpec struct {
	name                   string
	supportsBootstrap      bool
	renderDDLParts         func() []string
	renderSeedSQL          func() string
	postMigrationStatements func() []string
}

func schemaBackendSpecForDriver(driver string) (schemaBackendSpec, bool) {
	switch normalizedDriverName(driver) {
	case "sqlite":
		return schemaBackendSpec{
			name:              "sqlite",
			supportsBootstrap: true,
			renderDDLParts: func() []string {
				return []string{renderSQLiteBaseDDL(), renderSQLiteAuxDDL()}
			},
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("sqlite")
			},
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
		}, true
	case "mysql":
		return schemaBackendSpec{
			name:              "mysql",
			supportsBootstrap: false,
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("mysql")
			},
			postMigrationStatements: func() []string { return nil },
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

func (spec schemaBackendSpec) ddlParts() []string {
	if spec.renderDDLParts == nil {
		return nil
	}
	return spec.renderDDLParts()
}

func (spec schemaBackendSpec) seedSQL() string {
	if spec.renderSeedSQL == nil {
		return ""
	}
	return spec.renderSeedSQL()
}

func (spec schemaBackendSpec) bootstrapPlan() (bootstrapPlan, bool) {
	if !spec.supportsBootstrap || spec.renderDDLParts == nil {
		return bootstrapPlan{}, false
	}
	return bootstrapPlan{
		DDL:  joinDDLParts(spec.ddlParts()...),
		Seed: spec.seedSQL(),
	}, true
}

func (spec schemaBackendSpec) postMigrations() []string {
	if spec.postMigrationStatements == nil {
		return nil
	}
	return spec.postMigrationStatements()
}
