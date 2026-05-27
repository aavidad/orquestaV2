package orquestaopesconnector

import "testing"

func TestOPESControlledPathV0AceptaSoloPathsRelativosControlados(t *testing.T) {
	valid := []string{
		"/api/jobs",
		"/api/jobs?execution_mode=external&limit=1",
		"/api/jobs/job-ref-001",
	}
	for _, path := range valid {
		if !opesControlledPathV0(path) {
			t.Fatalf("path valido rechazado: %q", path)
		}
	}

	invalid := []string{
		"",
		"api/jobs",
		"//evil.example/api/jobs",
		"https://evil.example/api/jobs",
		"/api/../secrets",
		"/api/jobs#fragment",
		`/api\jobs`,
	}
	for _, path := range invalid {
		if opesControlledPathV0(path) {
			t.Fatalf("path invalido aceptado: %q", path)
		}
	}
}
