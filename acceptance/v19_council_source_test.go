package acceptance_test

import (
	"path/filepath"
	"testing"
)

func TestV19PSourceFixtureEnvelopeAndProductPaths(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v19Fixture](t, v19FixtureRepositoryPath(repositoryRoot))
	if fixture.SchemaVersion != 1 || fixture.ReceiptSchemaVersion != 3 ||
		fixture.TrustedBaseGitCommitOID != fixture.ProductDeltaBaseGitCommitOID ||
		fixture.TrustedBaseGitCommitOID != "6f244a7594141c74dc28e095e0e5e32a05102826" ||
		fixture.OutputPath != v19OutputPath || fixture.ReceiptPath != v19ReceiptPath ||
		fixture.Command == "" || len(fixture.ExecutionArgv) != 3 {
		t.Fatalf("V19 fixture is not V3-envelope ready: %+v", fixture)
	}
	if fixture.ExecutionArgv[0] != "sh" || fixture.ExecutionArgv[1] != "-c" ||
		fixture.ExecutionArgv[2] != v19ValidationShellBody() ||
		fixture.Command != "sh -c '"+fixture.ExecutionArgv[2]+"'" {
		t.Fatalf("V19 frozen command/argv drift: %q %#v", fixture.Command, fixture.ExecutionArgv)
	}
	assertV19ProductPresence(t, fixture)
}

func v19FixtureRepositoryPath(repositoryRoot string) string {
	return filepath.Join(repositoryRoot, "acceptance", v19FixturePath)
}
