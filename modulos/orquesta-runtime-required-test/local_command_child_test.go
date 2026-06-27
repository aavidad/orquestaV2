package orquestaruntimerequiredtest

import (
	"fmt"
	"os"
	"testing"
)

const childModeEnvV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_CHILD"
const childInvocationMarkerEnvV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_MARK_INVOCATION"
const childParentEnvLeakMarkerV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_PARENT_LEAK"
const childInvocationMarkerFileV0 = "required-test-invocations.log"

func TestMain(m *testing.M) {
	switch os.Getenv(childModeEnvV0) {
	case "pass":
		recordRequiredTestChildInvocationV0()
		fmt.Fprintln(os.Stdout, "required test passed")
		os.Exit(0)
	case "fail":
		recordRequiredTestChildInvocationV0()
		fmt.Fprintln(os.Stderr, "required test failed")
		os.Exit(7)
	case "empty_scan":
		recordRequiredTestChildInvocationV0()
		fmt.Fprintln(os.Stdout, `{"status":"pass","files_scanned":0,"finding_count":0}`)
		os.Exit(0)
	case "sensitive":
		recordRequiredTestChildInvocationV0()
		fmt.Fprintln(os.Stdout, `access_token=token-real Authorization: Bearer token-real /home/alberto/proyecto`)
		os.Exit(0)
	case "binary":
		recordRequiredTestChildInvocationV0()
		_, _ = os.Stdout.Write([]byte{'o', 'k', 0xff, '\n'})
		os.Exit(0)
	case "wait":
		recordRequiredTestChildInvocationV0()
		requiredTestChildWaitV0()
	}
	os.Exit(m.Run())
}
