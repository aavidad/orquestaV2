package db

func renderSupervisorSchemaObjectsForDriver(driver string, prefixes ...string) []string {
	statements := extractSchemaObjects(prefixes...)
	out := make([]string, 0, len(statements))
	for _, stmt := range statements {
		out = append(out, renderDriverColumnSyntax(driver, stmt))
	}
	return out
}

func ensureSupervisorSchemaObjects(prefixes ...string) error {
	for _, stmt := range renderSupervisorSchemaObjectsForDriver(CurrentStorageDriver(), prefixes...) {
		if _, err := DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
