package db

type schemaBackendSpec struct {
	name              string
	supportsBootstrap bool
	renderDDL         func() string
	renderSeedSQL     func() string
}

type bootstrapPlan struct {
	DDL  string
	Seed string
}

func schemaBackendSpecForDriver(driver string) (schemaBackendSpec, bool) {
	switch normalizedDriverName(driver) {
	case "sqlite":
		return schemaBackendSpec{
			name:              "sqlite",
			supportsBootstrap: true,
			renderDDL: func() string {
				return schemaDDLForDriver("sqlite")
			},
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("sqlite")
			},
		}, true
	case "mysql":
		return schemaBackendSpec{
			name:              "mysql",
			supportsBootstrap: true,
			renderDDL: func() string {
				return schemaDDLForDriver("mysql")
			},
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("mysql")
			},
		}, true
	case "postgres":
		return schemaBackendSpec{
			name:              "postgres",
			supportsBootstrap: true,
			renderDDL: func() string {
				return schemaDDLForDriver("postgres")
			},
			renderSeedSQL: func() string {
				return schemaSeedDataForDriver("postgres")
			},
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

func (spec schemaBackendSpec) ddl() string {
	if spec.renderDDL == nil {
		return ""
	}
	return spec.renderDDL()
}

func (spec schemaBackendSpec) seedSQL() string {
	if spec.renderSeedSQL == nil {
		return ""
	}
	return spec.renderSeedSQL()
}

func (spec schemaBackendSpec) bootstrapPlan() (bootstrapPlan, bool) {
	if !spec.supportsBootstrap {
		return bootstrapPlan{}, false
	}
	return bootstrapPlan{
		DDL:  spec.ddl(),
		Seed: spec.seedSQL(),
	}, true
}

func bootstrapPlanForDriver(driver string) (bootstrapPlan, bool) {
	spec, ok := schemaBackendSpecForDriver(driver)
	if !ok {
		return bootstrapPlan{}, false
	}
	return spec.bootstrapPlan()
}
