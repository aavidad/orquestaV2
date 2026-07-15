package orquesta_test

import (
	"strconv"
	"strings"
	"testing"

	"orquesta/internal/config"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
)

func TestCredentialRefsMatchConfigAndGoalOpaqueRefs(t *testing.T) {
	for _, value := range []string{"actor:alice", "AD Subject 42", "urn:example:subject/one", "usuario:á"} {
		actor, actorErr := goal.NewActorRef(value)
		project, projectErr := goal.NewProjectRef(value)
		if actorErr != nil || projectErr != nil || credentials.ValidateOwnerRef(credentials.OwnerRef(actor.String())) != nil ||
			credentials.ValidateScopeRef(credentials.ScopeRef(project.String())) != nil {
			t.Fatalf("valid opaque ref %q diverged: actor=%v project=%v", value, actorErr, projectErr)
		}
	}
	for _, value := range []string{"", " padded", "padded ", "nul\x00ref"} {
		_, actorErr := goal.NewActorRef(value)
		if actorErr == nil || credentials.ValidateOwnerRef(credentials.OwnerRef(value)) == nil ||
			credentials.ValidateScopeRef(credentials.ScopeRef(value)) == nil {
			t.Fatalf("invalid opaque ref %q accepted by one boundary", value)
		}
	}

	valid := []string{"credential:a", "credential:a-b_1.x", "credential:" + strings.Repeat("z", 128)}
	invalid := []string{"credential:", "credential:.leading", "credential:a:b", "credential:UPPER", "credential:" + strings.Repeat("z", 129)}
	for _, value := range append(valid, invalid...) {
		credentialErr := credentials.ValidateCredentialRef(credentials.CredentialRef(value))
		_, configErr := config.ParseExplicit([]byte("[runtime.codex]\ncredential_ref = " + strconv.Quote(value) + "\n"))
		wantValid := credentialRefContains(valid, value)
		if (credentialErr == nil) != wantValid || (configErr == nil) != wantValid {
			t.Fatalf("credential ref %q parity: credentials=%v config=%v want_valid=%v", value, credentialErr, configErr, wantValid)
		}
	}
}

func credentialRefContains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
