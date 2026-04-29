package db

import "sync"

var schemaEnsureMu sync.Mutex

func renderSchemaObjectsForDriver(driver string, prefixes ...string) []string {
	statements := extractSchemaObjects(prefixes...)
	out := make([]string, 0, len(statements))
	for _, stmt := range statements {
		out = append(out, renderDriverColumnSyntax(driver, stmt))
	}
	return out
}

func ensureSchemaObjects(prefixes ...string) error {
	schemaEnsureMu.Lock()
	defer schemaEnsureMu.Unlock()
	for _, stmt := range renderSchemaObjectsForDriver(CurrentStorageDriver(), prefixes...) {
		if _, err := DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func ensureRenderedSchemaStatements(statements []string) error {
	schemaEnsureMu.Lock()
	defer schemaEnsureMu.Unlock()
	for _, stmt := range statements {
		if _, err := DB.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}
